package iris

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

const (
	// CookieName is the name of the cookie which this frameworks sends to the temporary request in order to get the named parameters
	CookieName = "____iris____"
)

///TODO: continue IRouter
/*
type IRouter interface {
	NewRouter() *Router
	HandleFunc(registedPath string, handler Handler, method string) *Route
	HandleAnnotated(handler Annotated) (*Route, error)
	Find(req *http.Request) *Route
	Cache(mustEnable bool)
}*/

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	MiddlewareSupporter
	//no routes map[string]map[string][]*Route // key = path prefix, value a map which key = method and the vaulue an array of the routes starts with that prefix and method
	//routes map[string][]*Route // key = path prefix, value an array of the routes starts with that prefix
	nodes      tree
	cache      *MemoryRouterCache
	httpErrors *HTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
	// find is setted at the first HandleFunc(.startCache) of this route,
	// if Cache is true then the find  sets to _findCache, if not then find = _find
	// I did it for better perfomance, no check if Cache so many times inside .find
	///TODO: 1. remove it and replace the whole Router with two types of routers, one CachedRouter and the SimpleRouter or something like that
	///TODO: 2. but first create an interface named IRouter to hold the nessecary functions
	find func(req *http.Request) (_route *Route)
}

// NewRouter creates and returns an empty Router
func newRouter() *Router {
	r := &Router{nodes: make(tree, 0), cache: &MemoryRouterCache{}}
	r.find = r._findCache //by default it is to the normal find. this is changing to the startCache
	return r
}

// Cache receives a boolean value
// If false then disables the memory cache
// If true then enables the memory cache, which it's by default
//
// Returns the MemoryRouterCache pointer reference in order to optionally set it's options using .Cache(true).SetOptions(...)
func (r *Router) Cache(enable bool) *MemoryRouterCache {
	if enable {
		r.cache.Enable()
		r.find = r._findCache
	} else {
		r.cache.Disable()
		r.find = r._find
	}
	return r.cache
}

func Cache(enable bool) *MemoryRouterCache {
	return DefaultServer.router.Cache(enable)
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

		//start the cache when at least one route is registed to the router
		//already started? no problem it handles it.
		r.cache.TryStart()
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
func (r *Router) Use(handler MiddlewareHandler) *Router {
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
	return r
}

// Use registers a a custom handler, with next, as a global middleware
func (s *Server) Use(handler MiddlewareHandler) *Server {
	s.router.Use(handler)
	return s
}

// Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) *Server {

	DefaultServer.router.Use(handler)
	return DefaultServer
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func (s *Server) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	s.router.UseFunc(handlerFunc)
	return s
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	DefaultServer.router.UseFunc(handlerFunc)
	return DefaultServer
}

// UseHandler registers a simple http.Handler as global middleware
func (s *Server) UseHandler(handler http.Handler) *Server {
	s.router.UseHandler(handler)
	return s
}

// UseHandler registers a simple http.Handler as global middleware
func UseHandler(handler http.Handler) *Server {
	DefaultServer.router.UseHandler(handler)
	return DefaultServer
}

func (r *Router) _findCache(req *http.Request) *Route {
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

// find returns the correct/matched route, if any, for  the request
// if error route != nil , then the errorcode will be 200 OK
func (r *Router) _find(req *http.Request) *Route {
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
					if _route.Match(req.URL.Path) {
						return _route
					}

				}
			}
		}
	}

	//println(req.URL.Path)

	return nil
}
