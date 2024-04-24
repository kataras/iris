package router

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/errgroup"
	"github.com/kataras/iris/v12/core/netutil"
	macroHandler "github.com/kataras/iris/v12/macro/handler"

	"github.com/kataras/golog"
)

type (
	// RequestHandler the middle man between acquiring a context and releasing it.
	// By-default is the router algorithm.
	RequestHandler interface {
		// Note: A different interface in order  to hide the rest of the implementation.
		// We only need the `FireErrorCode` to be accessible through the Iris application (see `iris.go#Build`)
		HTTPErrorHandler

		// HandleRequest should handle the request based on the Context.
		HandleRequest(ctx *context.Context)
		// Build should builds the handler, it's being called on router's BuildRouter.
		Build(provider RoutesProvider) error
		// RouteExists reports whether a particular route exists.
		RouteExists(ctx *context.Context, method, path string) bool
	}

	// HTTPErrorHandler should contain a method `FireErrorCode` which
	// handles http unsuccessful status codes.
	HTTPErrorHandler interface {
		// FireErrorCode should send an error response to the client based
		// on the given context's response status code.
		FireErrorCode(ctx *context.Context)
	}

	// RouteAdder is an optional interface that can be implemented by a `RequestHandler`.
	RouteAdder interface {
		// AddRoute should add a route to the request handler directly.
		AddRoute(*Route) error
	}
)

// ErrNotRouteAdder throws on `AddRouteUnsafe` when a registered `RequestHandler`
// does not implements the optional `AddRoute(*Route) error` method.
var ErrNotRouteAdder = errors.New("request handler does not implement AddRoute method")

type routerHandler struct {
	// Config.
	disablePathCorrection            bool
	disablePathCorrectionRedirection bool
	fireMethodNotAllowed             bool
	enablePathIntelligence           bool
	forceLowercaseRouting            bool
	//
	logger *golog.Logger

	trees      []*trie
	errorTrees []*trie

	hosts                bool             // true if at least one route contains a Subdomain.
	errorHosts           bool             // true if error handlers are registered to at least one Subdomain.
	errorDefaultHandlers context.Handlers // the main handler(s) for default error code handlers, when not registered directly by the end-developer.
}

var (
	_ RequestHandler   = (*routerHandler)(nil)
	_ HTTPErrorHandler = (*routerHandler)(nil)
)

type routerHandlerDynamic struct {
	RequestHandler
	rw sync.RWMutex

	locked uint32
}

// RouteExists reports whether a particular route exists.
func (h *routerHandlerDynamic) RouteExists(ctx *context.Context, method, path string) (exists bool) {
	h.lock(false, func() error {
		exists = h.RequestHandler.RouteExists(ctx, method, path)
		return nil
	})

	return
}

func (h *routerHandlerDynamic) AddRoute(r *Route) error {
	if v, ok := h.RequestHandler.(RouteAdder); ok {
		return h.lock(true, func() error {
			return v.AddRoute(r)
		})
	}

	return ErrNotRouteAdder
}

func (h *routerHandlerDynamic) lock(writeAccess bool, fn func() error) error {
	if atomic.CompareAndSwapUint32(&h.locked, 0, 1) {
		if writeAccess {
			h.rw.Lock()
		} else {
			h.rw.RLock()
		}

		err := fn()

		// check agan because fn may called the unlock method.
		if atomic.CompareAndSwapUint32(&h.locked, 1, 0) {
			if writeAccess {
				h.rw.Unlock()
			} else {
				h.rw.RUnlock()
			}
		}

		return err
	}

	return fn()
}

func (h *routerHandlerDynamic) Build(provider RoutesProvider) error {
	// Build can be called inside HandleRequest if the route handler
	// calls the RefreshRouter method, and it will stuck on the rw.Lock() call,
	// so use a custom version of it.
	// h.rw.Lock()
	// defer h.rw.Unlock()

	return h.lock(true, func() error {
		return h.RequestHandler.Build(provider)
	})
}

func (h *routerHandlerDynamic) HandleRequest(ctx *context.Context) {
	h.lock(false, func() error {
		h.RequestHandler.HandleRequest(ctx)
		return nil
	})
}

func (h *routerHandlerDynamic) FireErrorCode(ctx *context.Context) {
	h.lock(false, func() error {
		h.RequestHandler.FireErrorCode(ctx)
		return nil
	})
}

// NewDynamicHandler returns a new router handler which is responsible handle each request
// with routes that can be added in serve-time.
// It's a wrapper of the `NewDefaultHandler`.
// It's being used when the `ConfigurationReadOnly.GetEnableDynamicHandler` is true.
func NewDynamicHandler(config context.ConfigurationReadOnly, logger *golog.Logger) RequestHandler /* #2167 */ {
	handler := NewDefaultHandler(config, logger)
	return wrapDynamicHandler(handler)
}

func wrapDynamicHandler(handler RequestHandler) RequestHandler {
	return &routerHandlerDynamic{
		RequestHandler: handler,
	}
}

// NewDefaultHandler returns the handler which is responsible
// to map the request with a route (aka mux implementation).
func NewDefaultHandler(config context.ConfigurationReadOnly, logger *golog.Logger) RequestHandler {
	var (
		disablePathCorrection            bool
		disablePathCorrectionRedirection bool
		fireMethodNotAllowed             bool
		enablePathIntelligence           bool
		forceLowercaseRouting            bool
		dynamicHandlerEnabled            bool
	)

	if config != nil { // #2147
		disablePathCorrection = config.GetDisablePathCorrection()
		disablePathCorrectionRedirection = config.GetDisablePathCorrectionRedirection()
		fireMethodNotAllowed = config.GetFireMethodNotAllowed()
		enablePathIntelligence = config.GetEnablePathIntelligence()
		forceLowercaseRouting = config.GetForceLowercaseRouting()
		dynamicHandlerEnabled = config.GetEnableDynamicHandler()
	}

	handler := &routerHandler{
		disablePathCorrection:            disablePathCorrection,
		disablePathCorrectionRedirection: disablePathCorrectionRedirection,
		fireMethodNotAllowed:             fireMethodNotAllowed,
		enablePathIntelligence:           enablePathIntelligence,
		forceLowercaseRouting:            forceLowercaseRouting,
		logger:                           logger,
	}

	if dynamicHandlerEnabled {
		return wrapDynamicHandler(handler)
	}

	return handler
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
	// GetRouterFilters returns the app's router filters.
	// Read `UseRouter` for more.
	// The map can be altered before router built.
	GetRouterFilters() map[Party]*Filter
	// GetDefaultErrorMiddleware should return
	// the default error handler middleares.
	GetDefaultErrorMiddleware() context.Handlers
}

func defaultErrorHandler(ctx *context.Context) {
	if ok, err := ctx.GetErrPublic(); ok {
		// If an error is stored and it's not a private one
		// write it to the response body.
		ctx.WriteString(err.Error())
		return
	}
	// Otherwise, write the code's text instead.
	ctx.WriteString(context.StatusText(ctx.GetStatusCode()))
}

func (h *routerHandler) Build(provider RoutesProvider) error {
	h.trees = h.trees[0:0] // reset, inneed when rebuilding.
	h.errorTrees = h.errorTrees[0:0]

	// set the default error code handler, will be fired on error codes
	// that are not handled by a specific handler (On(Any)ErrorCode).
	h.errorDefaultHandlers = append(provider.GetDefaultErrorMiddleware(), defaultErrorHandler)

	rp := errgroup.New("Routes Builder")
	registeredRoutes := provider.GetRoutes()

	// before sort.
	for _, r := range registeredRoutes {
		if r.topLink != nil {
			bindMultiParamTypesHandler(r)
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

	noLogCount := 0

	for _, r := range registeredRoutes {
		if r.NoLog {
			noLogCount++
		}

		if h.forceLowercaseRouting {
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

	printRoutesInfo(h.logger, registeredRoutes, noLogCount)

	return errgroup.Check(rp)
}

func bindMultiParamTypesHandler(r *Route) { // like overlap feature but specifically for path parameters.
	r.BuildHandlers()

	h := r.Handlers[1:] // remove the macro evaluator handler as we manually check below.
	f := macroHandler.MakeFilter(r.tmpl)
	if f == nil {
		return // should never happen, previous checks made to set the top link.
	}

	currentStatusCode := r.StatusCode
	if currentStatusCode == 0 {
		currentStatusCode = http.StatusOK
	}

	decisionHandler := func(ctx *context.Context) {
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

	r.topLink.builtinBeginHandlers = append(context.Handlers{decisionHandler}, r.topLink.builtinBeginHandlers...)
}

func canHandleSubdomain(ctx *context.Context, subdomain string) bool {
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
	} else if !strings.HasPrefix(requestHost, subdomain) { // subdomain contains the dot, e.g. "admin."
		return false
	}

	return true
}

func (h *routerHandler) HandleRequest(ctx *context.Context) {
	method := ctx.Method()
	path := ctx.Path()

	if !h.disablePathCorrection {
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			// Remove trailing slash and client-permanent rule for redirection,
			// if confgiuration allows that and path has an extra slash.

			// update the new path and redirect.
			u := ctx.Request().URL
			// use Trim to ensure there is no open redirect due to two leading slashes
			path = "/" + strings.Trim(path, "/")
			u.Path = path
			if !h.disablePathCorrectionRedirection {
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

		if h.hosts && !canHandleSubdomain(ctx, t.subdomain) {
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

	if h.fireMethodNotAllowed {
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

	if h.enablePathIntelligence && method == http.MethodGet {
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

func statusCodeSuccessful(statusCode int) bool {
	return !context.StatusCodeNotSuccessful(statusCode)
}

// FireErrorCode handles the response's error response.
// If `Configuration.ResetOnFireErrorCode()` is true
// and the response writer was a recorder one
// then it will try to reset the headers and the body before calling the
// registered (or default) error handler for that error code set by
// `ctx.StatusCode` method.
func (h *routerHandler) FireErrorCode(ctx *context.Context) {
	// On common response writer, always check
	// if we can't reset the body and the body has been filled
	// which means that the status code already sent,
	// then do not fire this custom error code,
	// rel: context/context.go#EndRequest.
	//
	// Note that, this is set to 0 on recorder because it holds the response before sent,
	// so we check their len(Body()) instead, look below.
	if ctx.ResponseWriter().Written() > 0 {
		return
	}

	statusCode := ctx.GetStatusCode() // the response's cached one.

	if ctx.Application().ConfigurationReadOnly().GetResetOnFireErrorCode() /* could be an argument too but we must not break the method */ {
		// if we can reset the body, probably manual call of `Application.FireErrorCode`.
		if w, ok := ctx.IsRecording(); ok {
			if statusCodeSuccessful(w.StatusCode()) { // if not an error status code
				w.WriteHeader(statusCode) // then set it manually here, otherwise it should be set via ctx.StatusCode(...)
			}
			// reset if previous content and it's recorder, keep the status code.
			w.ClearHeaders()
			w.ResetBody()

			if cw, ok := w.ResponseWriter.(*context.CompressResponseWriter); ok {
				// recorder wraps a compress writer.
				cw.Disabled = true
			}
		} else if w, ok := ctx.ResponseWriter().(*context.CompressResponseWriter); ok {
			// reset and disable the gzip in order to be an expected form of http error result
			w.Disabled = true
		}
	} else {
		// check if a body already set (the error response is handled by the handler itself,
		// see `Context.EndRequest`)
		if w, ok := ctx.IsRecording(); ok {
			if len(w.Body()) > 0 {
				return
			}
		}
	}

	for i := range h.errorTrees {
		t := h.errorTrees[i]

		if statusCode != t.statusCode {
			continue
		}

		if h.errorHosts && !canHandleSubdomain(ctx, t.subdomain) {
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
			// users.Done(func(ctx *context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})

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

	// not error handler found,
	// see if failed with a stored error, and if so
	// then render it, otherwise write a default message.
	ctx.Do(h.errorDefaultHandlers)
}

func (h *routerHandler) subdomainAndPathAndMethodExists(ctx *context.Context, t *trie, method, path string) bool {
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
func (h *routerHandler) RouteExists(ctx *context.Context, method, path string) bool {
	for i := range h.trees {
		t := h.trees[i]
		if h.subdomainAndPathAndMethodExists(ctx, t, method, path) {
			return true
		}
	}

	return false
}
