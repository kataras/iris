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
	method := req.Method
	path := req.URL.Path
	if ctx := r.cache.GetItem(method, path); ctx != nil {
		ctx.Request = req
		ctx.ResponseWriter = res
		ctx.do()

		return
	}
	ctx := r.station.pool.Get().(*Context)
	ctx.ResponseWriter = res
	ctx.Request = req
	ctx.clear()

	if r.processRequest(ctx) {
		//TODO: if isGet { we lose 8k nanoseconds here(100k operations 28knanoseconds per op), it is not important for now, I will find a way to automative it
		//if something found and served then add it's clone to the cache
		r.cache.AddItem(method, path, ctx.Clone())
		//}
	}

	r.station.pool.Put(ctx)
}
