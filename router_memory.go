package iris

import (
	"net/http"
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

	var _branch *branch
	var _route *Route
	var _method = req.Method

search:
	{
		isHead := _method == "HEAD"
		_tree := r.trees[_method]
		if _tree != nil {
			for i := 0; i < len(_tree); i++ {
				_branch = _tree[i]
				if len(req.URL.Path) < len(_branch.prefix) {
					continue
				}
				hasPrefix := req.URL.Path[0:len(_branch.prefix)] == _branch.prefix
				//println("check url prefix: ", req.URL.Path[0:len(_branch.prefix)]+" with node's:  ", _branch.prefix)
				if hasPrefix {
					for j := 0; j < len(_branch.routes); j++ {
						_route = _branch.routes[j]
						if !_route.Verify(req.URL.Path) {
							continue

						}
						_route.ServeHTTP(res, req)
						r.cache.AddItem(_method, req.URL.Path, _route)
						return

					}

					//if prefix found on head but no route no route found, then search to the GET tree also
					if isHead {
						_method = HTTPMethods.GET
						goto search
					}
					r.httpErrors.NotFound(res)
					return

				}

			}
		} else if isHead { //if no any branches with routes found for the HEAD then try to search on GET tree
			_method = HTTPMethods.GET
			goto search
		}
	}
	//not found
	//println(req.URL.Path)

	r.httpErrors.NotFound(res)
}
