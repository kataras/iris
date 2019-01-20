package router

import (
	"net/http"
	"sync"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

// Router is the "director".
// Caller should provide a request handler (router implementation or root handler).
// Router is responsible to build the received request handler and run it
// to serve requests, based on the received context.Pool.
//
// User can refresh the router with `RefreshRouter` whenever a route's field is changed by him.
type Router struct {
	mu sync.Mutex // for Downgrade, WrapRouter & BuildRouter,
	// not indeed but we don't to risk its usage by third-parties.
	requestHandler RequestHandler   // build-accessible, can be changed to define a custom router or proxy, used on RefreshRouter too.
	mainHandler    http.HandlerFunc // init-accessible
	wrapperFunc    func(http.ResponseWriter, *http.Request, http.HandlerFunc)

	cPool          *context.Pool // used on RefreshRouter
	routesProvider RoutesProvider
}

// NewRouter returns a new empty Router.
func NewRouter() *Router { return &Router{} }

// RefreshRouter re-builds the router. Should be called when a route's state
// changed (i.e Method changed at serve-time).
func (router *Router) RefreshRouter() error {
	return router.BuildRouter(router.cPool, router.requestHandler, router.routesProvider, true)
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
	router.mainHandler = func(w http.ResponseWriter, r *http.Request) {
		ctx := cPool.Acquire(w, r)
		router.requestHandler.HandleRequest(ctx)
		cPool.Release(ctx)
	}

	if router.wrapperFunc != nil { // if wrapper used then attach that as the router service
		router.mainHandler = NewWrapper(router.wrapperFunc, router.mainHandler).ServeHTTP
	}

	return nil
}

// Downgrade "downgrades", alters the router supervisor service(Router.mainHandler)
//  algorithm to a custom one,
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

// WrapperFunc is used as an expected input parameter signature
// for the WrapRouter. It's a "low-level" signature which is compatible
// with the net/http.
// It's being used to run or no run the router based on a custom logic.
type WrapperFunc func(w http.ResponseWriter, r *http.Request, firstNextIsTheRouter http.HandlerFunc)

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
	if wrapperFunc == nil {
		return
	}

	router.mu.Lock()
	defer router.mu.Unlock()

	if router.wrapperFunc != nil {
		// wrap into one function, from bottom to top, end to begin.
		nextWrapper := wrapperFunc
		prevWrapper := router.wrapperFunc
		wrapperFunc = func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if next != nil {
				nexthttpFunc := http.HandlerFunc(func(_w http.ResponseWriter, _r *http.Request) {
					prevWrapper(_w, _r, next)
				})
				nextWrapper(w, r, nexthttpFunc)
			}
		}
	}

	router.wrapperFunc = wrapperFunc
}

// ServeHTTPC serves the raw context, useful if we have already a context, it by-pass the wrapper.
func (router *Router) ServeHTTPC(ctx context.Context) {
	router.requestHandler.HandleRequest(ctx)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mainHandler(w, r)
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (router *Router) RouteExists(ctx context.Context, method, path string) bool {
	return router.requestHandler.RouteExists(ctx, method, path)
}

type wrapper struct {
	router      http.HandlerFunc // http.HandlerFunc to catch the CURRENT state of its .ServeHTTP on case of future change.
	wrapperFunc func(http.ResponseWriter, *http.Request, http.HandlerFunc)
}

// NewWrapper returns a new http.Handler wrapped by the 'wrapperFunc'
// the "next" is the final "wrapped" input parameter.
//
// Application is responsible to make it to work on more than one wrappers
// via composition or func clojure.
func NewWrapper(wrapperFunc func(w http.ResponseWriter, r *http.Request, routerNext http.HandlerFunc), wrapped http.HandlerFunc) http.Handler {
	return &wrapper{
		wrapperFunc: wrapperFunc,
		router:      wrapped,
	}
}

func (wr *wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wr.wrapperFunc(w, r, wr.router)
}
