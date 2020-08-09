package router

import (
	"errors"
	"net/http"
	"sync"

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

	preHandlers    context.Handlers // run before requestHandler, as middleware, same way context's handlers run, see `UseRouter`.
	requestHandler RequestHandler   // build-accessible, can be changed to define a custom router or proxy, used on RefreshRouter too.
	mainHandler    http.HandlerFunc // init-accessible
	wrapperFunc    WrapperFunc

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
func (router *Router) RefreshRouter() error {
	return router.BuildRouter(router.cPool, router.requestHandler, router.routesProvider, true)
}

// ErrNotRouteAdder throws on `AddRouteUnsafe` when a registered `RequestHandler`
// does not implements the optional `AddRoute(*Route) error` method.
var ErrNotRouteAdder = errors.New("request handler does not implement AddRoute method")

// AddRouteUnsafe adds a route directly to the router's request handler.
// Works before or after Build state.
// Mainly used for internal cases like `iris.WithSitemap`.
// Do NOT use it on serve-time.
func (router *Router) AddRouteUnsafe(routes ...*Route) error {
	if h := router.requestHandler; h != nil {
		if v, ok := h.(interface {
			AddRoute(*Route) error
		}); ok {
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

// UseRouter registers one or more handlers that are fired
// before the main router's request handler.
//
// Use this method to register handlers, that can ran
// independently of the incoming request's method and path values,
// that they will be executed ALWAYS against ALL incoming requests.
// Example of use-case: CORS.
//
// Note that because these are executed before the router itself
// the Context should not have access to the `GetCurrentRoute`
// as it is not decided yet which route is responsible to handle the incoming request.
// It's one level higher than the `WrapRouter`.
// The context SHOULD call its `Next` method in order to proceed to
// the next handler in the chain or the main request handler one.
// ExecutionRules are NOT applied here.
func (router *Router) UseRouter(handlers ...context.Handler) {
	router.preHandlers = append(router.preHandlers, handlers...)
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

	// the important
	if len(router.preHandlers) > 0 {
		handlers := append(router.preHandlers, func(ctx *context.Context) {
			// set the handler index back to 0 so the route's handlers can be executed as exepcted.
			ctx.HandlerIndex(0)
			// execute the main request handler, this will fire the found route's handlers
			// or if error the error code's associated handler.
			router.requestHandler.HandleRequest(ctx)
		})

		router.mainHandler = func(w http.ResponseWriter, r *http.Request) {
			ctx := cPool.Acquire(w, r)
			// execute the final handlers chain.
			ctx.Do(handlers)
			cPool.Release(ctx)
		}
	} else {
		router.mainHandler = func(w http.ResponseWriter, r *http.Request) {
			ctx := cPool.Acquire(w, r)
			router.requestHandler.HandleRequest(ctx)
			cPool.Release(ctx)
		}
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
	router.wrapperFunc = makeWrapperFunc(router.wrapperFunc, wrapperFunc)
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
