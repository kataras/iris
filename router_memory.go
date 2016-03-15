package iris

import (
	"net/http"
	"time"
)

// MemoryRouter is the cached version of the Router
type MemoryRouter struct {
	*Router
	cache IRouterCache
}

// NewMemoryRouter returns a MemoryRouter
// receives an underline *Router object and int options like MaxItems and ResetDurationTime
func NewMemoryRouter(underlineRouter *Router, maxitems int, resetDuration time.Duration) *MemoryRouter {
	r := &MemoryRouter{}
	r.Router = underlineRouter
	//moved to the station.go r.Router = NewRouter(station) // extends all methods from the standar router
	r.cache = NewMemoryRouterCache()
	r.cache.SetMaxItems(maxitems) //no max items just clear every 5 minutes
	ticker := NewTicker()
	ticker.OnTick(r.cache.OnTick) // registers the cache to the ticker
	ticker.Start(resetDuration)   //starts the ticker now

	return r
}

// HandleFunc same as Router.HandleFunc
func (r *MemoryRouter) HandleFunc(method string, registedPath string, handlerFn HandlerFunc) *Route {
	return r.Router.HandleFunc(method, registedPath, handlerFn)
}

// Handle same as Router.Handle
func (r *MemoryRouter) Handle(method string, registedPath string, handler Handler) *Route {
	return r.Router.Handle(method, registedPath, handler)
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *MemoryRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//defer r.mu.Unlock() defer is slow.
	if ctx := r.cache.GetItem(req.Method, req.URL.Path); ctx != nil { //TODO: edw 700ms
		ctx.Request = req
		ctx.ResponseWriter = res
		ctx.Renderer.responseWriter = res
		ctx.handler.Serve(ctx)
		return
	}
	ctx := r.station.pool.Get().(*Context)
	_root := r.garden[req.Method]
	if _root != nil {

		ctx.Request = req
		ctx.ResponseWriter = res
		ctx.Renderer.responseWriter = ctx.ResponseWriter
		handler, params, _ := _root.getValue(req.URL.Path, ctx.Params) // pass the parameters here for 0 allocation
		if handler != nil {

			ctx.handler = handler
			ctx.Params = params
			handler.Serve(ctx)

			r.cache.AddItem(req.Method, req.URL.Path, ctx.Clone())
			ctx.ResponseWriter = nil
			ctx.Renderer.responseWriter = nil
			ctx.Params = ctx.Params[0:0]
			r.station.pool.Put(ctx)
			return
		}

	}
	r.httpErrors.NotFound(res)
}
