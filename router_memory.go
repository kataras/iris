package iris

import (
	"net/http"
	"time"
)

type MemoryRouter struct {
	*Router
	cache IRouterCache
}

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

func (r *MemoryRouter) HandleFunc(method string, registedPath string, handlerFn HandlerFunc) *Route {
	return r.Router.HandleFunc(method, registedPath, handlerFn)
}

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

	var reqPath = req.URL.Path
	var method = req.Method
	ctx := r.station.pool.Get().(*Context)
	ctx.Request = req
	ctx.handler = nil
	ctx.ResponseWriter = res
	ctx.Params = ctx.Params[0:0]
	_root := r.garden[method]
	if _root != nil {

		handler, params, _ := _root.getValue(reqPath, ctx.Params) // pass the parameters here for 0 allocation
		if handler != nil {

			ctx.Params = params
			ctx.Renderer.responseWriter = ctx.ResponseWriter
			ctx.handler = handler
			handler.Serve(ctx)
			r.station.pool.Put(ctx)
			r.cache.AddItem(method, reqPath, ctx)
			return
		}

	}
	r.httpErrors.NotFound(res)
}
