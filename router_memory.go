package iris

import (
	"net/http"
	"strings"
	"time"
)

type MemoryRouter struct {
	*Router
	cache IRouterCache
}

func NewMemoryRouter(maxitems int, resetDuration time.Duration) *MemoryRouter {
	r := &MemoryRouter{}
	r.Router = NewRouter() // extends all methods from the standar router
	r.cache = NewMemoryRouterCache()
	r.cache.SetMaxItems(maxitems) //no max items just clear every 5 minutes
	ticker := NewTicker()
	ticker.OnTick(r.cache.OnTick) // registers the cache to the ticker
	ticker.Start(resetDuration)   //starts the ticker now

	return r
}

func (r *MemoryRouter) HandleFunc(registedPath string, handler Handler, method string) *Route {

	return r.Router.HandleFunc(registedPath, handler, method)
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *MemoryRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//defer r.mu.Unlock() defer is slow.
	if v := r.cache.GetItem(req.Method, req.URL.Path); v != nil {
		v.ServeHTTP(res, req)
		return
	}

	var _node *node
	var _route *Route
	var _nodes = r.nodes[req.Method]
	if _nodes != nil {
		for i := 0; i < len(_nodes); i++ {
			_node = _nodes[i]

			if strings.HasPrefix(req.URL.Path, _node.prefix) {
				for j := 0; j < len(_node.routes); j++ {
					_route = _node.routes[j]
					if !_route.Verify(req.URL.Path) {
						continue
					}

					_route.ServeHTTP(res, req)
					r.cache.AddItem(req.Method, req.URL.Path, _route)
					return

				}
			}
		}
	}
	//not found
	//println(req.URL.Path)

	r.httpErrors.NotFound(res)
}
