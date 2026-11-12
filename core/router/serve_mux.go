package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/netutil"
	macroHandler "github.com/kataras/iris/v14/core/router/macro/handler"
	"github.com/kataras/iris/v14/internal/errgroup"

	"github.com/kataras/golog"
	"github.com/schollz/closestmatch"
)

// ServeMux is the interface that wraps the FindClosestPaths, HandleRequest and Build methods.
// It's used to build the router and serve the incoming requests.
//
// See `BuildServeMux` package-level function to construct the default ServeMux.
type ServeMux interface {
	// RouteExists(ctx *context.Context, method, path string) (exists bool)
	FindClosestPaths(subdomain, searchPath string, n int) []string

	HandleRequest(ctx *context.Context)
	// ServeStatusNotFound(ctx *context.Context) // current route is nil.
	// ServeStatusMethodNotAllowed(ctx *context.Context) // current route is nil.

	Build(noRoute, noMethod *Route, routes *RouteRegistry) error
}

// BuildServeMux returns a new ServeMux built from the given router options, logger,
// the pre-routing, no-route and no-method routes and the route registry.
// It's used to construct the default ServeMux.
// Used internally by the `core/router` package.
func BuildServeMux(
	options *RouterOptions,
	logger *golog.Logger,
	preRoute, noRoute, noMethod *Route,
	routes *RouteRegistry,
	// executed after mux standard http wrapper and before serve mux router.
) (ServeMux, error) {
	mux := newDefaultServeMux(options, logger)
	if preRoute.HandlerList.Count() > 0 {
		mux = newServeMuxMiddleware(mux, preRoute)
	}
	if err := mux.Build(noRoute, noMethod, routes); err != nil {
		return nil, err
	}

	return mux, nil
}

type dynamicServeMux struct { // a wrapper for the serveMux, to enable dynamic route registration and request handler.
	*serveMux
}

var _ ServeMux = (*dynamicServeMux)(nil)

func (h *dynamicServeMux) HandleRequest(ctx *context.Context) {
	// Read-lock guards against concurrent route rebuilds while serving.
	h.rw.RLock()
	defer h.rw.RUnlock()
	h.serveMux.HandleRequest(ctx)
}

type serveMux struct {
	// Config.
	rootHost                         string // the server's host option. MUST be not empty if hosts is true on later steps.
	disablePathCorrection            bool
	disablePathCorrectionRedirection bool
	fireMethodNotAllowed             bool
	enablePathIntelligence           bool
	enablePathEscape                 bool
	forceLowercaseRouting            bool
	//
	logger *golog.Logger

	tries []*trie
	// key = subdomain
	// value = closest of static routes, filled on `BuildRouter`.
	closestPaths map[string]*closestmatch.ClosestMatch

	hosts bool // true if at least one route contains a Subdomain.

	noRoute, noMethod *Route // handles 404 and 405. All handlers, including middlewares for handling errors, get status code by ctx.GetStatusCode().
	// httpBeginErrorHandlers context.Handlers // handlers that ran before the httpErrorHandler, forwarded by ctx.Next.
	rw sync.RWMutex
}

var _ ServeMux = (*serveMux)(nil)

// newDefaultServeMux returns the handler which is responsible
// to map the request with a route (aka mux implementation).
func newDefaultServeMux(options *RouterOptions, logger *golog.Logger) ServeMux {
	var (
		rootHost                         string
		disablePathCorrection            bool
		disablePathCorrectionRedirection bool
		fireMethodNotAllowed             bool
		enablePathIntelligence           bool
		enablePathEscape                 bool
		forceLowercaseRouting            bool
		dynamicHandlerEnabled            bool
	)

	if options != nil { // #2147
		rootHost = options.Host
		disablePathCorrection = options.DisablePathCorrection
		disablePathCorrectionRedirection = options.DisablePathCorrectionRedirection
		fireMethodNotAllowed = options.FireMethodNotAllowed
		enablePathIntelligence = options.EnablePathIntelligence
		enablePathEscape = options.EnablePathEscape
		forceLowercaseRouting = options.ForceLowercaseRouting
		dynamicHandlerEnabled = options.EnableDynamicHandler
	}

	mux := &serveMux{
		rootHost:                         rootHost,
		disablePathCorrection:            disablePathCorrection,
		disablePathCorrectionRedirection: disablePathCorrectionRedirection,
		fireMethodNotAllowed:             fireMethodNotAllowed,
		enablePathIntelligence:           enablePathIntelligence,
		enablePathEscape:                 enablePathEscape,
		forceLowercaseRouting:            forceLowercaseRouting,
		noRoute:                          nil, // see router handler builder.
		noMethod:                         nil, // >>
		logger:                           logger,
	}

	if dynamicHandlerEnabled {
		return &dynamicServeMux{mux}
	}

	return mux
}

// Build is used to build the router based on the registered routes and error middlewares.
func (h *serveMux) Build(noRoute *Route, noMethod *Route, routes *RouteRegistry) error {
	// set the error handlers.
	noRoute.BuildHandlers()
	h.noRoute = noRoute
	noMethod.BuildHandlers()
	h.noMethod = noMethod

	// build routes.
	if err := h.buildRoutes(routes); err != nil {
		return err
	}

	// build closest.
	h.buildClosest(routes)

	return nil
}

const (
	// NoRouteName is the name of the default 404 not found route.
	NoRouteName = "NO_ROUTE"
	// NoMethodName is the name of the default 405 method not allowed route.
	NoMethodName = "NO_METHOD"
	// PreRoutingName is the name of the default pre-routing route.
	PreRoutingName = "PRE_ROUTING"
)

// buildRoutes builds the routes and error handlers.
func (h *serveMux) buildRoutes(routes *RouteRegistry) error {
	h.tries = h.tries[0:0] // reset, just in case.

	rp := errgroup.New("Routes Builder")
	registeredRoutes := routes.Build()

	noLogCount := 0

	for _, r := range registeredRoutes {
		if r.NoLog {
			noLogCount++
		}

		if h.forceLowercaseRouting {
			// only in that state, keep everything else as end-developer registered.
			r.NodePath = strings.ToLower(r.NodePath)
		}

		if r.Subdomain != "" {
			if h.rootHost == "" {
				return fmt.Errorf("root host is required when at least one route contains a subdomain, please set the host option on the server or router configuration")
				// we can also continue and set it to ctx.Domain() (root host domain by current request context on the HandleRequest method but it's not recommended, it's better to be explicit and set it on the configuration level).
			}
			h.hosts = true
		}

		if r.topLink == nil {
			// build the r.Handlers based on begin and done handlers, if any.
			r.BuildHandlers()

			// the only "bad" with this is if the user made an error
			// on route, it will be stacked shown in this build state
			// and no in the lines of the user's action, they should read
			// the docs better. Or TODO: add a link here in order to help new users.
			if err := h.addRoute(r); err != nil {
				// node errors:
				rp.Addf("%s: %w", r.String(), err)
				continue
			}
		}
	}

	printRoutesInfo(h.logger, registeredRoutes, noLogCount)

	return errgroup.Check(rp)
}

// bindMultiParamTypesHandler binds the multi-param types handler to the route.
func bindMultiParamTypesHandler(r *Route) { // like overlap feature but specifically for path parameters.
	r.BuildHandlers()

	h := r.Handlers[1:] // remove the macro evaluator handler as we manually check below on "decisionHandler".
	f := macroHandler.MakeFilter(r.tmpl)
	if f == nil {
		return // should never happen, previous checks made to set the top link.
	}

	decisionHandler := func(ctx *context.Context) {
		// println("core/router/handler.go: decision handler; " + ctx.Path() + " route.Name: " + r.Name + " vs context's " + ctx.CurrentRoute().Name())
		currentRoute := ctx.CurrentRoute()

		// Different path parameters types in the same path, fallback should registered first e.g. {path} {string},
		// because the handler on this case is executing from bottom/last to top/first.
		if f(ctx) {
			// println("core/router/handler.go: filter for : " + r.Name + " passed")
			ctx.SetCurrentRoute(r)
			// Note: error handlers will be the same, routes came from the same router,
			// no need to update them.
			ctx.HandlerIndex(0)
			ctx.Do(h)
			return
		}

		ctx.SetCurrentRoute(currentRoute)
		ctx.StatusCode(http.StatusOK)
		ctx.Next()
	}

	// r.topLink.builtinBeginHandlers = append(context.Handlers{decisionHandler}, r.topLink.builtinBeginHandlers...)
	r.topLink.builtinBeginHandlerList.Prepend(decisionHandler)
}

// buildClosest builds the closest paths for each subdomain.
func (h *serveMux) buildClosest(routes *RouteRegistry) {
	subdomainPaths := make(map[string][]string)

	for _, r := range routes.List() {
		if !r.IsStatic() {
			continue
		}

		subdomainPaths[r.Subdomain] = append(subdomainPaths[r.Subdomain], r.NodePath)
	}

	h.closestPaths = make(map[string]*closestmatch.ClosestMatch)
	for subdomain, paths := range subdomainPaths {
		h.closestPaths[subdomain] = closestmatch.New(paths, []int{3, 4, 6})
	}
}

// getTree returns the trie based on the status code, method and subdomain.
func (h *serveMux) getTree(method, subdomain string) *trie {
	for i := range h.tries {
		t := h.tries[i]
		if t.method == method && t.subdomain == subdomain {
			return t
		}
	}

	return nil
}

// addRoute registers a route to the tries or error tries, it is used internally on Build method.
func (h *serveMux) addRoute(r *Route) error {
	var (
		method    = r.Method
		subdomain = r.Subdomain
		path      = r.NodePath
		handlers  = r.Handlers
	)

	t := h.getTree(method, subdomain)

	if t == nil {
		n := newTrieNode()
		// first time we register a route to this method with this subdomain
		t = &trie{method: method, subdomain: subdomain, root: n}
		h.tries = append(h.tries, t)
	}

	t.insert(path, r, handlers)

	return nil
}

// // RouteExists reports whether a particular route exists.
// // It will search from the current subdomain of context's host, if not inside the root domain.
// func (h *serveMux) RouteExists(ctx *context.Context, method, path string) (exists bool) {
// 	if h == nil {
// 		return false
// 	}
// 	h.lock(false, func() error {
// 		exists = h.routeExists(ctx, method, path)
// 		return nil
// 	})
// 	return
// }
// func (h *serveMux) routeExists(ctx *context.Context, method, path string) bool {
// 	for i := range h.tries {
// 		t := h.tries[i]
// 		if h.subdomainAndPathAndMethodExists(ctx, t, method, path) {
// 			return true
// 		}
// 	}
// 	return false
// }

// subdomainAndPathAndMethodExists reports whether a particular subdomain, path and method trie node exists.
func (h *serveMux) subdomainAndPathAndMethodExists(ctx *context.Context, t *trie, method, path string) bool {
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
			serverHost := h.rootHost
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

// FindClosestPaths returns a list of "n" paths close to "path" under the given "subdomain".
//
// Order may change.
func (h *serveMux) FindClosestPaths(subdomain, searchPath string, n int) []string {
	if h == nil {
		return nil
	}

	if h.closestPaths == nil {
		return nil
	}

	cm, ok := h.closestPaths[subdomain]
	if !ok {
		return nil
	}

	list := cm.ClosestN(searchPath, n)
	if len(list) == 1 && list[0] == "" {
		// yes, it may return empty string as its first slice element when not found.
		return nil
	}

	return list
}

// HandleRequest is the main router handler, it's responsible to route the incoming request to the correct handler based on the request's host, method and path.
func (h *serveMux) HandleRequest(ctx *context.Context) {
	r := ctx.Request()

	method := r.Method
	if h.enablePathEscape {
		r.URL.Path = r.URL.EscapedPath()
	}
	path := r.URL.Path

	if !h.disablePathCorrection {
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			// Remove trailing slash and client-permanent rule for redirection,
			// if confgiuration allows that and path has an extra slash.

			// update the new path and redirect.
			u := r.URL
			// use Trim to ensure there is no open redirect due to two leading slashes
			path = "/" + strings.Trim(path, "/")
			u.Path = path
			if !h.disablePathCorrectionRedirection {
				// do redirect, else continue with the modified path without the last "/".
				url := u.String()

				// Fixes https://github.com/kataras/iris/issues/921
				// This is caused for security reasons, imagine a payment shop,
				// you can't just permantly redirect a POST request, so just 307 (RFC 7231, 6.4.7).
				// The same goes for every method that carries a body: a 301 lets clients drop it.
				if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == MethodQuery {
					ctx.Redirect(url, http.StatusTemporaryRedirect)
					return
				}

				ctx.Redirect(url, http.StatusMovedPermanently)
				return
			}

		}
	}

	for i := range h.tries {
		t := h.tries[i]
		if method != t.method {
			continue
		}

		if h.hosts && !canHandleSubdomain(ctx, h.rootHost, t.subdomain) {
			continue
		}

		n := t.search(path, ctx.Params())
		if n != nil {
			ctx.SetCurrentRoute(n.Route)
			ctx.HandlerIndex(0)
			ctx.Do(n.Handlers)
			// found

			// Do not try to serve the error here, let the dev handlers do this.
			return
		}
		// not found or method not allowed.
		break
	}

	if h.fireMethodNotAllowed {
		var allowedMethods []string
		for i := range h.tries {
			t := h.tries[i]

			// If the method is not allowed, then check if the path is registered to another method.
			// Collect the allowed method(s).
			if h.subdomainAndPathAndMethodExists(ctx, t, "", path) {
				// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
				// The response MUST include an Allow header containing a list of valid methods for the requested resource.
				allowedMethods = append(allowedMethods, t.method)
				continue
			}
		}

		if len(allowedMethods) > 0 { // it's actually a 405.
			ctx.SetHeader("Allow", strings.Join(allowedMethods, ", "))
			ctx.StatusCode(http.StatusMethodNotAllowed)
			ctx.HandlerIndex(0)
			ctx.SetCurrentRoute(h.noMethod)
			ctx.Do(h.noMethod.Handlers)
			return
		}
	}

	if h.enablePathIntelligence && method == http.MethodGet {
		closestPaths := ctx.FindClosest(1)
		if len(closestPaths) > 0 {
			u := r.URL
			u.Path = closestPaths[0]
			ctx.Redirect(u.String(), http.StatusMovedPermanently)
			return
		}
	}

	ctx.StatusCode(http.StatusNotFound)
	ctx.HandlerIndex(0)
	ctx.SetCurrentRoute(h.noRoute)
	ctx.Do(h.noRoute.Handlers)
}

func canHandleSubdomain(ctx *context.Context, rootHost, subdomain string) bool {
	if subdomain == "" {
		return true
	}

	if rootHost == "" {
		rootHost = ctx.Domain() // if it's empty (on default router matcher) then set it to the current request's root domain (slow method).
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
		serverHost := rootHost
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

func defaultHTTPErrorHandler(ctx *context.Context) {
	if ctx.ResponseWriter().Written() > 0 {
		return
	}

	if ok, err := ctx.ErrPublic(); ok {
		// If an error is stored and it's not a private one
		// write it to the response body.
		ctx.WriteString(err.Error())
		return
	}

	// Otherwise, write the code's text instead.
	ctx.WriteString(context.StatusText(ctx.GetStatusCode()))
}
