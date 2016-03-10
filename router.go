package iris

import (
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

///TODO: continue IRouter

type IRouter interface {
	Use(MiddlewareHandler)
	UseFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc))
	UseHandler(http.Handler)
	HandleFunc(string, Handler, string) *Route
	HandleAnnotated(Annotated) (*Route, error)
	GetErrors() *HTTPErrors
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type Routes []*Route

//implementing the sort.Interface for type 'Routes'

func (routes Routes) Len() int {
	return len(routes)
}

func (routes Routes) Less(r1, r2 int) bool {
	//sort by longest path parts no  longest fullpath, longest first.
	return len(routes[r1].pathParts) > len(routes[r2].pathParts)
}

func (routes Routes) Swap(r1, r2 int) {
	routes[r1], routes[r2] = routes[r2], routes[r1]
}

//end

// node is just a collection of routes group by path's prefix, used inside Router.
type node struct {
	prefix           string
	routes           Routes
	prioritySetTimes int // full is 100, this will be incremented and zero again every time a sort is done, sort is done when this number react to 100
	//that menas that every 100 requests the routes will sort itself via the route.priority
}

type prefixNodes []*node

//implementing the sort.Interface for type 'prefixNodes'

func (nodes prefixNodes) Len() int {
	return len(nodes)
}

func (nodes prefixNodes) Less(r1, r2 int) bool {
	//sort by longest path prefix, longest first.
	return len(nodes[r1].prefix) > len(nodes[r2].prefix)
}

func (nodes prefixNodes) Swap(r1, r2 int) {
	nodes[r1], nodes[r2] = nodes[r2], nodes[r1]
}

//end

type tree map[string]prefixNodes

func (tr tree) addRoute(method string, route *Route) {
	_nodes := tr[method]
	//route.pathPrefix = strings.TrimSpace(route.pathPrefix)

	if _nodes == nil {
		_nodes = make([]*node, 0)
	}
	ok := false
	var _node *node
	index := 0
	for index, _node = range _nodes {
		//check if route has parameters or * after the prefix, if yes then add a slash to the end
		routePref := route.pathPrefix

		if _node.prefix == routePref {
			tr[method][index].routes = append(_node.routes, route)
			//sort routes by the most larger path parts
			sort.Sort(tr[method][index].routes)
			ok = true
			break
		}
	}
	if !ok {
		_node = &node{prefix: route.pathPrefix, routes: make([]*Route, 0)}
		_node.routes = append(_node.routes, route)
		//sort routes by the most larger path parts
		sort.Sort(_node.routes)
		//_node.makePriority(route)
		tr[method] = append(tr[method], _node)
	}

	//sort nodes by the longest prefix
	sort.Sort(tr[method])
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	MiddlewareSupporter
	//no routes map[string]map[string][]*Route // key = path prefix, value a map which key = method and the vaulue an array of the routes starts with that prefix and method
	//routes map[string][]*Route // key = path prefix, value an array of the routes starts with that prefix
	nodes      tree
	cache      *IRouterCache
	httpErrors *HTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func NewRouter() *Router {
	return &Router{nodes: make(tree, 0), httpErrors: DefaultHTTPErrors()}
}

func (r *Router) SetErrors(httperr *HTTPErrors) {
	r.httpErrors = httperr
}

func (r *Router) GetErrors() *HTTPErrors {
	return r.httpErrors
}

// HandleFunc registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the name of the route + the pattern
//
// HandleFunc is exported for the future, not being used outside of the iris package yet, some of other functions also.
func (r *Router) HandleFunc(registedPath string, handler Handler, method string) *Route {
	//r.mu.Lock()
	//defer is 5 times slower only some nanosecconds difference but let's make it faster unlock it at the end of the function manually  or not?
	//defer r.mu.Unlock()
	//but wait... do we need locking here?

	var route *Route
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil {
		route = newRoute(registedPath, handler)

		if len(r.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = r.middlewareHandlers
		}

		r.nodes.addRoute(method, route)

	}
	route.httpErrors = r.httpErrors
	return route
}

//HandleFunc handle without methods, if not method given before the Listen then the http methods will be []{"GET"}
func (s *Server) HandleFunc(path string, handler Handler, method string) *Route {
	return s.router.HandleFunc(path, handler, method)
}

// Handle in the route registers a normal http.Handler
//func (r *Router) Handle(registedPath string, httpHandler http.Handler, method string) *Route {
//	return r.HandleFunc(registedPath, HandlerFunc(httpHandler), method)
//}

// handleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (r *Router) HandleAnnotated(irisHandler Annotated) (*Route, error) {
	//r.mu.Lock()
	//defer r.mu.Unlock()
	var route *Route
	var method string
	var path string
	var handleFunc reflect.Value
	var template string
	var templateIsGLob = false
	var errMessage = ""
	val := reflect.ValueOf(irisHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Anonymous && typeField.Name == "Annotated" {
			tags := strings.Split(strings.TrimSpace(string(typeField.Tag)), " ")
			//we can have two keys, one is the tag starts with the method (GET,POST: "/user/api/{userId(int)}")
			//and the other if exists is the OPTIONAL TEMPLATE/TEMPLATE-GLOB: "file.html"

			//check for Template first because on the method we break and return error if no method found , for now.
			if len(tags) > 1 {
				secondTag := tags[1]

				templateIdx := strings.Index(string(secondTag), ":")

				templateTagName := strings.ToUpper(string(secondTag[:templateIdx]))

				//check if it's regex pattern

				if templateTagName == "TEMPLATE-GLOB" {
					templateIsGLob = true
				}

				temlateTagValue, templateUnqerr := strconv.Unquote(string(secondTag[templateIdx+1:]))

				if templateUnqerr != nil {
					//err = errors.New(err.Error() + "\niris.RegisterHandler: Error on getting template: " + templateUnqerr.Error())
					errMessage = errMessage + "\niris.HandleAnnotated: Error on getting template: " + templateUnqerr.Error()

					continue
				}

				template = temlateTagValue
			}

			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

			if unqerr != nil {
				errMessage = errMessage + "\niris.HandleAnnotated: Error on getting path: " + unqerr.Error()
				continue
			}

			path = tagValue
			avalaibleMethodsStr := strings.Join(HTTPMethods.ANY, ",")
			//has multi methods seperate by commas

			if !strings.Contains(avalaibleMethodsStr, tagName) {
				//wrong methods passed
				errMessage = errMessage + "\niris.HandleAnnotated: Wrong methods passed to Handler -> " + tagName
				continue
			}
			//it is single 'GET','POST' .... method
			method = tagName

		} else {
			errMessage = "\nError on Iris on HandleAnnotated: Struct passed but it doesn't have an anonymous property of type iris.Annotated, please refer to docs\n"
		}

	}

	if errMessage == "" {
		//route = r.server.Router.Route(path, irisHandler.Handle, methods...)

		//now check/get the Handle method from the irisHandler 'obj'.
		handleFunc = reflect.ValueOf(irisHandler).MethodByName("Handle")
		if !handleFunc.IsValid() {
			errMessage = "Missing Handle function inside iris.Annotated"
		}

		if errMessage == "" {
			route = r.HandleFunc(path, convertToHandler(handleFunc.Interface()), method)
			//check if template string has stored by the tag ( look before this block )

			if template != "" {
				if templateIsGLob {
					route.Template().SetGlob(template)
				} else {
					route.Template().Add(template)
				}

			}
		}

	}

	var err error = nil
	if errMessage != "" {
		err = errors.New(errMessage)
	}

	return route, err
}

// HandleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (s *Server) HandleAnnotated(irisHandler Annotated) (*Route, error) {
	return s.router.HandleAnnotated(irisHandler)
}

///////////////////
//global middleware
///////////////////

// Use registers a a custom handler, with next, as a global middleware
func (r *Router) Use(handler MiddlewareHandler) {
	r.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(r.nodes) > 0 {
		for _, _nodes := range r.nodes {
			for _, v := range _nodes {
				for _, route := range v.routes {
					route.Use(handler)
				}

			}

		}
	}

}

// Use registers a a custom handler, with next, as a global middleware
func (s *Server) Use(handler MiddlewareHandler) {
	s.router.Use(handler)

}

// Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) {

	DefaultServer.router.Use(handler)
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func (s *Server) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	s.router.UseFunc(handlerFunc)

}

// UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	DefaultServer.router.UseFunc(handlerFunc)

}

// UseHandler registers a simple http.Handler as global middleware
func (s *Server) UseHandler(handler http.Handler) {
	s.router.UseHandler(handler)

}

// UseHandler registers a simple http.Handler as global middleware
func UseHandler(handler http.Handler) {
	DefaultServer.router.UseHandler(handler)

}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var _node *node
	var _route *Route
	var _nodes = r.nodes[req.Method]
	if _nodes != nil {
		for i := 0; i < len(_nodes); i++ {
			_node = _nodes[i]
			if len(req.URL.Path) < len(_node.prefix) {
				continue
			}
			hasPrefix := req.URL.Path[0:len(_node.prefix)] == _node.prefix
			if hasPrefix {
				for j := 0; j < len(_node.routes); j++ {
					_route = _node.routes[j]
					if !_route.Verify(req.URL.Path) {
						continue

					}
					_route.ServeHTTP(res, req)
					return

				}
				//if the prefix was found, so we are 100% at the correct node, because I make it to end with slashes
				//so no need to check other prefixes any more, just return 404 and exit from here.
				//println(req.URL.Path, " NOT found")
				r.httpErrors.NotFound(res)
				return

			}

		}
	}
	//no nodes or routes found
	//println(req.URL.Path, " NOT found")
	r.httpErrors.NotFound(res)

}
