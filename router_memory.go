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

func NewMemoryRouter() *MemoryRouter {
	r := &MemoryRouter{}
	r.Router = NewRouter() // extends all methods from the standar router
	r.cache = NewMemoryRouterCache()
	r.cache.SetMaxItems(0) //no max items just clear every 5 minutes
	ticker := NewTicker()
	ticker.OnTick(r.cache.OnTick) // registers the cache to the ticker
	ticker.Start(5 * time.Minute) //starts the ticker now

	return r
}

func (r *MemoryRouter) HandleFunc(registedPath string, handler Handler, method string) *Route {

	return r.Router.HandleFunc(registedPath, handler, method)
}

func (r *MemoryRouter) Find(req *http.Request) *Route {
	//defer r.mu.Unlock() defer is slow.
	if v := r.cache.GetItem(req.Method, req.URL.Path); v != nil {
		return v
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
					if _route.Match(req.URL.Path) {
						r.cache.AddItem(req.Method, req.URL.Path, _route)
						return _route

					}

				}
			}
		}
	}

	//println(req.URL.Path)

	return nil
}
