package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/errgroup"
	"github.com/kataras/iris/v12/core/netutil"
	macroHandler "github.com/kataras/iris/v12/macro/handler"

	"github.com/kataras/golog"
	"github.com/kataras/pio"
)

type (
	// RequestHandler the middle man between acquiring a context and releasing it.
	// By-default is the router algorithm.
	RequestHandler interface {
		// Note: A different interface in order  to hide the rest of the implementation.
		// We only need the `FireErrorCode` to be accessible through the Iris application (see `iris.go#Build`)
		HTTPErrorHandler

		// HandleRequest should handle the request based on the Context.
		HandleRequest(ctx context.Context)
		// Build should builds the handler, it's being called on router's BuildRouter.
		Build(provider RoutesProvider) error
		// RouteExists reports whether a particular route exists.
		RouteExists(ctx context.Context, method, path string) bool
	}

	// HTTPErrorHandler should contain a method `FireErrorCode` which
	// handles http unsuccessful status codes.
	HTTPErrorHandler interface {
		FireErrorCode(ctx context.Context)
	}
)

type routerHandler struct {
	config context.ConfigurationReadOnly
	logger *golog.Logger

	trees      []*trie
	errorTrees []*trie

	hosts      bool // true if at least one route contains a Subdomain.
	errorHosts bool // true if error handlers are registered to at least one Subdomain.
}

var _ RequestHandler = (*routerHandler)(nil)
var _ HTTPErrorHandler = (*routerHandler)(nil)

// NewDefaultHandler returns the handler which is responsible
// to map the request with a route (aka mux implementation).
func NewDefaultHandler(config context.ConfigurationReadOnly, logger *golog.Logger) RequestHandler {
	return &routerHandler{
		config: config,
		logger: logger,
	}
}

func (h *routerHandler) getTree(statusCode int, method, subdomain string) *trie {
	if statusCode > 0 {
		for i := range h.errorTrees {
			t := h.errorTrees[i]
			if t.statusCode == statusCode && t.subdomain == subdomain {
				return t
			}
		}
		return nil
	}

	for i := range h.trees {
		t := h.trees[i]
		if t.method == method && t.subdomain == subdomain {
			return t
		}
	}

	return nil
}

// AddRoute registers a route. See `Router.AddRouteUnsafe`.
func (h *routerHandler) AddRoute(r *Route) error {
	var (
		method     = r.Method
		statusCode = r.StatusCode
		subdomain  = r.Subdomain
		path       = r.Path
		handlers   = r.Handlers
	)

	t := h.getTree(statusCode, method, subdomain)

	if t == nil {
		n := newTrieNode()
		// first time we register a route to this method with this subdomain
		t = &trie{statusCode: statusCode, method: method, subdomain: subdomain, root: n}
		if statusCode > 0 {
			h.errorTrees = append(h.errorTrees, t)
		} else {
			h.trees = append(h.trees, t)
		}
	}

	t.insert(path, r.ReadOnly, handlers)

	return nil
}

// RoutesProvider should be implemented by
// iteral which contains the registered routes.
type RoutesProvider interface { // api builder
	GetRoutes() []*Route
	GetRoute(routeName string) *Route
}

func (h *routerHandler) Build(provider RoutesProvider) error {
	h.trees = h.trees[0:0] // reset, inneed when rebuilding.
	rp := errgroup.New("Routes Builder")
	registeredRoutes := provider.GetRoutes()

	// before sort.
	for _, r := range registeredRoutes {
		if r.topLink != nil {
			bindMultiParamTypesHandler(r.topLink, r)
		}
	}

	// sort, subdomains go first.
	sort.Slice(registeredRoutes, func(i, j int) bool {
		first, second := registeredRoutes[i], registeredRoutes[j]
		lsub1 := len(first.Subdomain)
		lsub2 := len(second.Subdomain)

		firstSlashLen := strings.Count(first.Path, "/")
		secondSlashLen := strings.Count(second.Path, "/")

		if lsub1 == lsub2 && first.Method == second.Method {
			if secondSlashLen < firstSlashLen {
				// fixes order when wildcard root is registered before other wildcard paths
				return true
			}

			if secondSlashLen == firstSlashLen {
				// fixes order when static path with the same prefix with a wildcard path
				// is registered after the wildcard path, although this is managed
				// by the low-level node but it couldn't work if we registered a root level wildcard, this fixes it.
				if len(first.tmpl.Params) == 0 {
					return false
				}
				if len(second.tmpl.Params) == 0 {
					return true
				}

				// No don't fix the order by framework's suggestion,
				// let it as it is today; {string} and {path} should be registered before {id} {uint} and e.t.c.
				// see `bindMultiParamTypesHandler` for the reason. Order of registration matters.
			}
		}

		// the rest are handled inside the node
		return lsub1 > lsub2
	})

	for _, r := range registeredRoutes {
		if h.config != nil && h.config.GetForceLowercaseRouting() {
			// only in that state, keep everything else as end-developer registered.
			r.Path = strings.ToLower(r.Path)
		}

		if r.Subdomain != "" {
			if r.StatusCode > 0 {
				h.errorHosts = true
			} else {
				h.hosts = true
			}
		}

		if r.topLink == nil {
			// build the r.Handlers based on begin and done handlers, if any.
			r.BuildHandlers()

			// the only "bad" with this is if the user made an error
			// on route, it will be stacked shown in this build state
			// and no in the lines of the user's action, they should read
			// the docs better. Or TODO: add a link here in order to help new users.
			if err := h.AddRoute(r); err != nil {
				// node errors:
				rp.Addf("%s: %w", r.String(), err)
				continue
			}
		}
	}

	// TODO: move this and make it easier to read when all cases are, visually, tested.
	if logger := h.logger; logger != nil && logger.Level == golog.DebugLevel {
		// group routes by method and print them without the [DBUG] and time info,
		// the route logs are colorful.
		// Note: don't use map, we need to keep registered order, use
		// different slices for each method.
		collect := func(method string) (methodRoutes []*Route) {
			for _, r := range registeredRoutes {
				if r.Method == method {
					methodRoutes = append(methodRoutes, r)
				}
			}

			return
		}

		type MethodRoutes struct {
			method string
			routes []*Route
		}

		allMethods := append(AllMethods, []string{MethodNone, ""}...)
		methodRoutes := make([]MethodRoutes, 0, len(allMethods))

		for _, method := range allMethods {
			routes := collect(method)
			if len(routes) > 0 {
				methodRoutes = append(methodRoutes, MethodRoutes{method, routes})
			}
		}

		if n := len(methodRoutes); n > 0 {
			tr := "routes"
			if len(registeredRoutes) == 1 {
				tr = tr[0 : len(tr)-1]
			}

			bckpNewLine := logger.NewLine
			logger.NewLine = false
			logger.Debugf("API: %d registered %s (", len(registeredRoutes), tr)
			logger.NewLine = bckpNewLine

			for i, m := range methodRoutes {
				// @method: @count
				if i > 0 {
					if i == n-1 {
						fmt.Fprint(logger.Printer, " and ")
					} else {
						fmt.Fprint(logger.Printer, ", ")
					}
				}
				if m.method == "" {
					m.method = "ERROR"
				}
				fmt.Fprintf(logger.Printer, "%d ", len(m.routes))
				pio.WriteRich(logger.Printer, m.method, traceMethodColor(m.method))
			}

			fmt.Fprint(logger.Printer, ")\n")
		}

		for i, m := range methodRoutes {
			for _, r := range m.routes {
				r.Trace(logger.Printer)
			}

			if i != len(allMethods)-1 {
				logger.Printer.Write(pio.NewLine)
			}
		}
	}

	return errgroup.Check(rp)
}

func bindMultiParamTypesHandler(top *Route, r *Route) {
	r.BuildHandlers()

	// println("here for top: " + top.Name + " and current route: " + r.Name)
	h := r.Handlers[1:] // remove the macro evaluator handler as we manually check below.
	f := macroHandler.MakeFilter(r.tmpl)
	if f == nil {
		return // should never happen, previous checks made to set the top link.
	}

	currentStatusCode := r.StatusCode
	if currentStatusCode == 0 {
		currentStatusCode = http.StatusOK
	}

	decisionHandler := func(ctx context.Context) {
		// println("core/router/handler.go: decision handler; " + ctx.Path() + " route.Name: " + r.Name + " vs context's " + ctx.GetCurrentRoute().Name())
		currentRoute := ctx.GetCurrentRoute()

		// Different path parameters types in the same path, fallback should registered first e.g. {path} {string},
		// because the handler on this case is executing from last to top.
		if f(ctx) {
			// println("core/router/handler.go: filter for : " + r.Name + " passed")
			ctx.SetCurrentRoute(r.ReadOnly)
			// Note: error handlers will be the same, routes came from the same party,
			// no need to update them.
			ctx.HandlerIndex(0)
			ctx.Do(h)
			return
		}

		ctx.SetCurrentRoute(currentRoute)
		ctx.StatusCode(currentStatusCode)
		ctx.Next()
	}

	r.topLink.beginHandlers = append(context.Handlers{decisionHandler}, r.topLink.beginHandlers...)
}

func (h *routerHandler) canHandleSubdomain(ctx context.Context, subdomain string) bool {
	if subdomain == "" {
		return true
	}

	requestHost := ctx.Host()
	if netutil.IsLoopbackSubdomain(requestHost) {
		// this fixes a bug when listening on
		// 127.0.0.1:8080 for example
		// and have a wildcard subdomain and a route registered to root domain.
		return false // it's not a subdomain, it's something like 127.0.0.1 probably
	}
	// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
	if subdomain == SubdomainWildcardIndicator {
		// mydomain.com -> invalid
		// localhost -> invalid
		// sub.mydomain.com -> valid
		// sub.localhost -> valid
		serverHost := ctx.Application().ConfigurationReadOnly().GetVHost()
		if serverHost == requestHost {
			return false // it's not a subdomain, it's a full domain (with .com...)
		}

		dotIdx := strings.IndexByte(requestHost, '.')
		slashIdx := strings.IndexByte(requestHost, '/')
		if dotIdx > 0 && (slashIdx == -1 || slashIdx > dotIdx) {
			// if "." was found anywhere but not at the first path segment (host).
		} else {
			return false
		}
		// continue to that, any subdomain is valid.
	} else if !strings.HasPrefix(requestHost, subdomain) { // subdomain contains the dot.
		return false
	}

	return true
}

func (h *routerHandler) HandleRequest(ctx context.Context) {
	method := ctx.Method()
	path := ctx.Path()
	config := h.config // ctx.Application().GetConfigurationReadOnly()

	if !config.GetDisablePathCorrection() {
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			// Remove trailing slash and client-permanent rule for redirection,
			// if confgiuration allows that and path has an extra slash.

			// update the new path and redirect.
			u := ctx.Request().URL
			// use Trim to ensure there is no open redirect due to two leading slashes
			path = "/" + strings.Trim(path, "/")
			u.Path = path
			if !config.GetDisablePathCorrectionRedirection() {
				// do redirect, else continue with the modified path without the last "/".
				url := u.String()

				// Fixes https://github.com/kataras/iris/issues/921
				// This is caused for security reasons, imagine a payment shop,
				// you can't just permantly redirect a POST request, so just 307 (RFC 7231, 6.4.7).
				if method == http.MethodPost || method == http.MethodPut {
					ctx.Redirect(url, http.StatusTemporaryRedirect)
					return
				}

				ctx.Redirect(url, http.StatusMovedPermanently)
				return
			}

		}
	}

	for i := range h.trees {
		t := h.trees[i]
		if method != t.method {
			continue
		}

		if h.hosts && !h.canHandleSubdomain(ctx, t.subdomain) {
			continue
		}

		n := t.search(path, ctx.Params())
		if n != nil {
			ctx.SetCurrentRoute(n.Route)
			ctx.Do(n.Handlers)
			// found
			return
		}
		// not found or method not allowed.
		break
	}

	if config.GetFireMethodNotAllowed() {
		for i := range h.trees {
			t := h.trees[i]
			// if `Configuration#FireMethodNotAllowed` is kept as defaulted(false) then this function will not
			// run, therefore performance kept as before.
			if h.subdomainAndPathAndMethodExists(ctx, t, "", path) {
				// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
				// The response MUST include an Allow header containing a list of valid methods for the requested resource.
				ctx.Header("Allow", t.method)
				ctx.StatusCode(http.StatusMethodNotAllowed)
				return
			}
		}
	}

	if config.GetEnablePathIntelligence() && method == http.MethodGet {
		closestPaths := ctx.FindClosest(1)
		if len(closestPaths) > 0 {
			u := ctx.Request().URL
			u.Path = closestPaths[0]
			ctx.Redirect(u.String(), http.StatusMovedPermanently)
			return
		}
	}

	ctx.StatusCode(http.StatusNotFound)
}

func (h *routerHandler) FireErrorCode(ctx context.Context) {
	statusCode := ctx.GetStatusCode() // the response's cached one.

	// if we can reset the body
	if w, ok := ctx.IsRecording(); ok {
		if statusCodeSuccessful(w.StatusCode()) { // if not an error status code
			w.WriteHeader(statusCode) // then set it manually here, otherwise it should be set via ctx.StatusCode(...)
		}
		// reset if previous content and it's recorder, keep the status code.
		w.ClearHeaders()
		w.ResetBody()
	} else if w, ok := ctx.ResponseWriter().(*context.GzipResponseWriter); ok {
		// reset and disable the gzip in order to be an expected form of http error result
		w.ResetBody()
		w.Disable()
	} else {
		// if we can't reset the body and the body has been filled
		// which means that the status code already sent,
		// then do not fire this custom error code.
		if ctx.ResponseWriter().Written() > 0 { // != -1, rel: context/context.go#EndRequest
			return
		}
	}

	for i := range h.errorTrees {
		t := h.errorTrees[i]

		if statusCode != t.statusCode {
			continue
		}

		if h.errorHosts && !h.canHandleSubdomain(ctx, t.subdomain) {
			continue
		}

		n := t.search(ctx.Path(), ctx.Params())
		if n == nil {
			// try to take the root's one.
			n = t.root.getChild(pathSep)
		}

		if n != nil {
			// Note: handlers can contain macro filters here,
			// they are registered as error handlers, see macro/handler.go#42.

			// fmt.Println("Error Handlers")
			// for _, h := range n.Handlers {

			// 	f, l := context.HandlerFileLine(h)
			// 	fmt.Printf("%s: %s:%d\n", ctx.Path(), f, l)
			// }

			// fire this http status code's error handlers chain.

			// ctx.StopExecution() // not uncomment this, is here to remember why to.
			// note for me: I don't stopping the execution of the other handlers
			// because may the user want to add a fallback error code
			// i.e
			// users := app.Party("/users")
			// users.Done(func(ctx context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})

			// use .HandlerIndex
			// that sets the current handler index to zero
			// in order to:
			// ignore previous runs that may changed the handler index,
			// via ctx.Next or ctx.StopExecution, if any.
			//
			// use .Do
			// that overrides the existing handlers and sets and runs these error handlers.
			// in order to:
			// ignore the route's after-handlers, if any.
			ctx.SetCurrentRoute(n.Route)
			// Should work with:
			// Manual call of ctx.Application().FireErrorCode(ctx) (handlers length > 0)
			// And on `ctx.SetStatusCode`: Context -> EndRequest -> FireErrorCode (handlers length > 0)
			// And on router: HandleRequest -> SetStatusCode -> Context ->
			//                EndRequest -> FireErrorCode (handlers' length is always 0)
			ctx.HandlerIndex(0)
			ctx.Do(n.Handlers)
			return
		}

		break
	}

	// not error handler found, write a default message.
	ctx.WriteString(StatusText(statusCode))
}

func statusCodeSuccessful(statusCode int) bool {
	return !context.StatusCodeNotSuccessful(statusCode)
}

func (h *routerHandler) subdomainAndPathAndMethodExists(ctx context.Context, t *trie, method, path string) bool {
	if method != "" && method != t.method {
		return false
	}

	if h.hosts && t.subdomain != "" {
		requestHost := ctx.Host()
		if netutil.IsLoopbackSubdomain(requestHost) {
			// this fixes a bug when listening on
			// 127.0.0.1:8080 for example
			// and have a wildcard subdomain and a route registered to root domain.
			return false // it's not a subdomain, it's something like 127.0.0.1 probably
		}
		// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
		if t.subdomain == SubdomainWildcardIndicator {
			// mydomain.com -> invalid
			// localhost -> invalid
			// sub.mydomain.com -> valid
			// sub.localhost -> valid
			serverHost := ctx.Application().ConfigurationReadOnly().GetVHost()
			if serverHost == requestHost {
				return false // it's not a subdomain, it's a full domain (with .com...)
			}

			dotIdx := strings.IndexByte(requestHost, '.')
			slashIdx := strings.IndexByte(requestHost, '/')
			if dotIdx > 0 && (slashIdx == -1 || slashIdx > dotIdx) {
				// if "." was found anywhere but not at the first path segment (host).
			} else {
				return false
			}
			// continue to that, any subdomain is valid.
		} else if !strings.HasPrefix(requestHost, t.subdomain) { // t.subdomain contains the dot.
			return false
		}
	}

	n := t.search(path, ctx.Params())
	return n != nil
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (h *routerHandler) RouteExists(ctx context.Context, method, path string) bool {
	for i := range h.trees {
		t := h.trees[i]
		if h.subdomainAndPathAndMethodExists(ctx, t, method, path) {
			return true
		}
	}

	return false
}
