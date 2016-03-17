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

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *MemoryRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//16/03/2016 Tried to get/pass only middlewares but it slow me 8k nanoseconds, so I re-do it as I had before.
	if ctx := r.cache.GetItem(req.Method, req.URL.Path); ctx != nil {
		ctx.Request = req
		ctx.ResponseWriter = NewResponseWriter(res)
		ctx.Renderer.responseWriter = res
		ctx.do()
		return
	}

	ctx := r.poolContextFor(req)
	if r.processRequest(ctx, res) {
		//if something found and served then add this to the cache
		r.cache.AddItem(req.Method, req.URL.Path, ctx.Clone())
	}
	r.station.pool.Put(ctx)

}
