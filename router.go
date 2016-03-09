package iris

import (
	"errors"
	"net/http"
	"reflect"
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

// node is just a collection of routes group by path's prefix, used inside Router.
type node struct {
	prefix string
	routes []*Route
}

type tree map[string][]*node

func (tr tree) addRoute(method string, route *Route) {
	_nodes := tr[method]

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
			ok = true
			break
		}
	}
	if !ok {
		_node = &node{prefix: route.pathPrefix, routes: make([]*Route, 0)}
		_node.routes = append(_node.routes, route)
		//_node.makePriority(route)
		tr[method] = append(tr[method], _node)
	}
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
	//wrongMethod := false
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
					return

				}
			}
		}
	}
	//not found
	r.httpErrors.NotFound(res)

}
