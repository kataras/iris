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
	if ctx := r.cache.GetItem(req.Method, req.URL.Path); ctx != nil { //TODO: edw 700ms
		ctx.Request = req
		ctx.ResponseWriter = res
		ctx.Renderer.responseWriter = res
		ctx.Do()
		return
	}

	ctx := r.poolContextFor(res, req)

	if r.processRequest(ctx) {
		//if something found and served then add this to the cache
		r.cache.AddItem(req.Method, req.URL.Path, ctx.Clone())
	}

	r.station.pool.Put(ctx)

}
