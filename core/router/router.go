package router

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/schollz/closestmatch"
)

// Router is the "director".
// Caller should provide a request handler (router implementation or root handler).
// Router is responsible to build the received request handler and run it
// to serve requests, based on the received context.Pool.
//
// User can refresh the router with `RefreshRouter` whenever a route's field is changed by him.
type Router struct {
	mu sync.Mutex // for Downgrade, WrapRouter & BuildRouter,

	requestHandler RequestHandler   // build-accessible, can be changed to define a custom router or proxy, used on RefreshRouter too.
	mainHandler    http.HandlerFunc // init-accessible
	wrapperFunc    WrapperFunc
	// wrappers to be built on BuildRouter state,
	// first is executed first at this case.
	// Case:
	// - SubdomainRedirect on user call, registers a wrapper, on design state
	// - i18n,if loaded and Subdomain or PathRedirect is true, registers a wrapper too, on build state
	// the SubdomainRedirect should be the first(subdomainWrap(i18nWrap)) wrapper
	// to be executed instead of last(i18nWrap(subdomainWrap)).
	wrapperFuncs []WrapperFunc

	cPool          *context.Pool // used on RefreshRouter
	routesProvider RoutesProvider

	// key = subdomain
	// value = closest of static routes, filled on `BuildRouter/RefreshRouter`.
	closestPaths map[string]*closestmatch.ClosestMatch
}

// NewRouter returns a new empty Router.
func NewRouter() *Router {
	return &Router{}
}

// RefreshRouter re-builds the router. Should be called when a route's state
// changed (i.e Method changed at serve-time).
//
// Note that in order to use RefreshRouter while in serve-time,
// you have to set the `EnableDynamicHandler` Iris Application setting to true,
// e.g. `app.Listen(":8080", iris.WithEnableDynamicHandler)`
func (router *Router) RefreshRouter() error {
	return router.BuildRouter(router.cPool, router.requestHandler, router.routesProvider, true)
}

// AddRouteUnsafe adds a route directly to the router's request handler.
// Works before or after Build state.
// Mainly used for internal cases like `iris.WithSitemap`.
// Do NOT use it on serve-time.
func (router *Router) AddRouteUnsafe(routes ...*Route) error {
	if h := router.requestHandler; h != nil {
		if v, ok := h.(RouteAdder); ok {
			for _, r := range routes {
				return v.AddRoute(r)
			}
		}
	}

	return ErrNotRouteAdder
}

// FindClosestPaths returns a list of "n" paths close to "path" under the given "subdomain".
//
// Order may change.
func (router *Router) FindClosestPaths(subdomain, searchPath string, n int) []string {
	if router.closestPaths == nil {
		return nil
	}

	cm, ok := router.closestPaths[subdomain]
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

func (router *Router) buildMainHandler(cPool *context.Pool, requestHandler RequestHandler) {
	router.mainHandler = func(w http.ResponseWriter, r *http.Request) {
		ctx := cPool.Acquire(w, r)
		router.requestHandler.HandleRequest(ctx)
		cPool.Release(ctx)
	}
}

func (router *Router) buildMainHandlerWithFilters(routerFilters map[Party]*Filter, cPool *context.Pool, requestHandler RequestHandler) {
	sortedFilters := make([]*Filter, 0, len(routerFilters))
	// key was just there to enforce uniqueness on API level.
	for _, f := range routerFilters {
		sortedFilters = append(sortedFilters, f)
		// append it as one handlers so execution rules are being respected in that step too.
		f.Handlers = append(f.Handlers, func(ctx *context.Context) {
			// set the handler index back to 0 so the route's handlers can be executed as expected.
			ctx.HandlerIndex(0)
			// execute the main request handler, this will fire the found route's handlers
			// or if error the error code's associated handler.
			router.requestHandler.HandleRequest(ctx)
		})
	}

	sort.SliceStable(sortedFilters, func(i, j int) bool {
		left, right := sortedFilters[i], sortedFilters[j]
		var (
			leftSubLen  = len(left.Subdomain)
			rightSubLen = len(right.Subdomain)

			leftSlashLen  = strings.Count(left.Path, "/")
			rightSlashLen = strings.Count(right.Path, "/")
		)

		if leftSubLen == rightSubLen {
			if leftSlashLen > rightSlashLen {
				return true
			}
		}

		if leftSubLen > rightSubLen {
			return true
		}

		if leftSlashLen > rightSlashLen {
			return true
		}

		if leftSlashLen == rightSlashLen {
			return len(left.Path) > len(right.Path)
		}

		return len(left.Path) > len(right.Path)
	})

	router.mainHandler = func(w http.ResponseWriter, r *http.Request) {
		ctx := cPool.Acquire(w, r)

		filterExecuted := false
		for _, f := range sortedFilters { // from subdomain, largest path to shortest.
			// fmt.Printf("Sorted filter execution: [%s] [%s]\n", f.Subdomain, f.Path)
			if f.Matcher.Match(ctx) {
				// fmt.Printf("Matched [%s] and execute [%d] handlers [%s]\n\n", ctx.Path(), len(f.Handlers), context.HandlersNames(f.Handlers))
				filterExecuted = true
				// execute the final handlers chain.
				ctx.Do(f.Handlers)
				break // and break on first found.
			}
		}

		if !filterExecuted {
			// If not at least one match filter found and executed,
			// then just run the router.
			router.requestHandler.HandleRequest(ctx)
		}

		cPool.Release(ctx)
	}
}

// BuildRouter builds the router based on
// the context factory (explicit pool in this case),
// the request handler which manages how the main handler will multiplexes the routes
// provided by the third parameter, routerProvider (it's the api builder in this case) and
// its wrapper.
//
// Use of RefreshRouter to re-build the router if needed.
func (router *Router) BuildRouter(cPool *context.Pool, requestHandler RequestHandler, routesProvider RoutesProvider, force bool) error {
	if requestHandler == nil {
		return errors.New("router: request handler is nil")
	}

	if cPool == nil {
		return errors.New("router: context pool is nil")
	}

	// build the handler using the routesProvider
	if err := requestHandler.Build(routesProvider); err != nil {
		return err
	}

	router.mu.Lock()
	defer router.mu.Unlock()

	// store these for RefreshRouter's needs.
	if force {
		router.cPool = cPool
		router.requestHandler = requestHandler
		router.routesProvider = routesProvider
	} else {
		if router.cPool == nil {
			router.cPool = cPool
		}

		if router.requestHandler == nil {
			router.requestHandler = requestHandler
		}

		if router.routesProvider == nil && routesProvider != nil {
			router.routesProvider = routesProvider
		}
	}

	// the important stuff.
	if routerFilters := routesProvider.GetRouterFilters(); len(routerFilters) > 0 {
		router.buildMainHandlerWithFilters(routerFilters, cPool, requestHandler)
	} else {
		router.buildMainHandler(cPool, requestHandler)
	}

	for i := len(router.wrapperFuncs) - 1; i >= 0; i-- {
		w := router.wrapperFuncs[i]
		if w == nil {
			continue
		}
		router.WrapRouter(w)
	}

	if router.wrapperFunc != nil { // if wrapper used then attach that as the router service
		router.mainHandler = newWrapper(router.wrapperFunc, router.mainHandler).ServeHTTP
	}

	// build closest.
	subdomainPaths := make(map[string][]string)
	for _, r := range router.routesProvider.GetRoutes() {
		if !r.IsStatic() {
			continue
		}

		subdomainPaths[r.Subdomain] = append(subdomainPaths[r.Subdomain], r.Path)
	}

	router.closestPaths = make(map[string]*closestmatch.ClosestMatch)
	for subdomain, paths := range subdomainPaths {
		router.closestPaths[subdomain] = closestmatch.New(paths, []int{3, 4, 6})
	}

	return nil
}

// Downgrade "downgrades", alters the router supervisor service(Router.mainHandler)
// algorithm to a custom one,
// be aware to change the global variables of 'ParamStart' and 'ParamWildcardStart'.
// can be used to implement a custom proxy or
// a custom router which should work with raw ResponseWriter, *Request
// instead of the Context(which again, can be retrieved by the Framework's context pool).
//
// Note: Downgrade will by-pass the Wrapper, the caller is responsible for everything.
// Downgrade is thread-safe.
func (router *Router) Downgrade(newMainHandler http.HandlerFunc) {
	router.mu.Lock()
	router.mainHandler = newMainHandler
	router.mu.Unlock()
}

// Downgraded returns true if this router is downgraded.
func (router *Router) Downgraded() bool {
	return router.mainHandler != nil && router.requestHandler == nil
}

// SetTimeoutHandler overrides the main handler with a timeout handler.
//
// TimeoutHandler supports the Pusher interface but does not support
// the Hijacker or Flusher interfaces.
//
// All previous registered wrappers and middlewares are still executed as expected.
func (router *Router) SetTimeoutHandler(timeout time.Duration, msg string) {
	if timeout <= 0 {
		return
	}

	mainHandler := router.mainHandler
	h := func(w http.ResponseWriter, r *http.Request) {
		mainHandler(w, r)
	}

	router.mainHandler = http.TimeoutHandler(http.HandlerFunc(h), timeout, msg).ServeHTTP
}

// WrapRouter adds a wrapper on the top of the main router.
// Usually it's useful for third-party middleware
// when need to wrap the entire application with a middleware like CORS.
//
// Developers can add more than one wrappers,
// those wrappers' execution comes from last to first.
// That means that the second wrapper will wrap the first, and so on.
//
// Before build.
func (router *Router) WrapRouter(wrapperFunc WrapperFunc) {
	// logger := context.DefaultLogger("router wrapper")
	// file, line := context.HandlerFileLineRel(wrapperFunc)
	// if router.wrapperFunc != nil {
	// 	wrappedFile, wrappedLine := context.HandlerFileLineRel(router.wrapperFunc)
	// 	logger.Infof("%s:%d wraps %s:%d", file, line, wrappedFile, wrappedLine)
	// } else {
	// 	logger.Infof("%s:%d wraps the main router", file, line)
	// }
	router.wrapperFunc = makeWrapperFunc(router.wrapperFunc, wrapperFunc)
}

// AddRouterWrapper adds a router wrapper.
// Unlike `WrapRouter` the first registered will be executed first
// so a wrapper wraps its next not the previous one.
// it defers the wrapping until the `BuildRouter`.
// Redirection wrappers should be added using this method
// e.g. SubdomainRedirect.
func (router *Router) AddRouterWrapper(wrapperFunc WrapperFunc) {
	router.wrapperFuncs = append(router.wrapperFuncs, wrapperFunc)
}

// PrependRouterWrapper like `AddRouterWrapper` but this wrapperFunc
// will always be executed before the previous `AddRouterWrapper`.
// Path form (no modification) wrappers should be added using this method
// e.g. ForceLowercaseRouting.
func (router *Router) PrependRouterWrapper(wrapperFunc WrapperFunc) {
	router.wrapperFuncs = append([]WrapperFunc{wrapperFunc}, router.wrapperFuncs...)
}

// ServeHTTPC serves the raw context, useful if we have already a context, it by-pass the wrapper.
func (router *Router) ServeHTTPC(ctx *context.Context) {
	router.requestHandler.HandleRequest(ctx)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mainHandler(w, r)
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (router *Router) RouteExists(ctx *context.Context, method, path string) bool {
	return router.requestHandler.RouteExists(ctx, method, path)
}
