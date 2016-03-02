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

type prefixRoute struct {
	prefix string
	routes []*Route
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	MiddlewareSupporter
	//no routes map[string]map[string][]*Route // key = path prefix, value a map which key = method and the vaulue an array of the routes starts with that prefix and method
	//routes map[string][]*Route // key = path prefix, value an array of the routes starts with that prefix
	routes []prefixRoute
	//mu            sync.RWMutex
	httpErrors *HTTPErrors //the only need of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func newRouter() *Router {
	return &Router{routes: make([]prefixRoute, 0)}
}

// HandleFunc registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the name of the route + the pattern
//
// HandleFunc is exported for the future, not being used outside of the iris package yet, some of other functions also.
func (r *Router) HandleFunc(registedPath string, handler Handler) *Route {
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

		//r.routes = append(r.routes, route)
		ok := false

		for index, _prefRoute := range r.routes {
			if _prefRoute.prefix == route.pathPrefix {
				//edw ginete h zhmia pleon,prepei kapws na to kanw
				r.routes[index].routes = append(_prefRoute.routes, route)
				//println("add a route to the prefix " + _prefRoute.prefix + " registedPath: " + registedPath)
				ok = true
			}
		}

		if !ok {
			registedPR := prefixRoute{prefix: route.pathPrefix, routes: make([]*Route, 0)}
			//println("register new prefixRoute with prefix: " + route.pathPrefix + " and registedPath : " + registedPath)
			registedPR.routes = append(registedPR.routes, route)
			r.routes = append(r.routes, registedPR)
		}

		/*
			var registedRoutePrefix *prefixRoute
			for _, prefixRoute := range r.routes {
				if prefixRoute.prefix == route.pathPrefix {
					registedRoutePrefix = &prefixRoute
				}
			}

			if registedRoutePrefix == nil {
				registedRoutePrefix = &prefixRoute{prefix: route.pathPrefix, routes: make([]*Route, 0)}


			}
			registedRoutePrefix.routes = append(registedRoutePrefix.routes, route)
			r.routes = append(r.routes, *registedRoutePrefix)
		*/
		/*

			if r.routes[route.pathPrefix] == nil {
				r.routes[route.pathPrefix] = make([]*Route, 0)
			}
			r.routes[route.pathPrefix] = append(r.routes[route.pathPrefix], route)*/

	}
	route.httpErrors = r.httpErrors
	return route
}

//HandleFunc handle without methods, if not method given before the Listen then the http methods will be []{"GET"}
func (s *Server) HandleFunc(path string, handler Handler) *Route {
	return s.router.HandleFunc(path, handler)
}

// Handle in the route registers a normal http.Handler
func (r *Router) Handle(registedPath string, httpHandler http.Handler) *Route {
	return r.HandleFunc(registedPath, HandlerFunc(httpHandler))
}

// handleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (r *Router) handleAnnotated(irisHandler Annotated) (*Route, error) {
	//r.mu.Lock()
	//defer r.mu.Unlock()
	var route *Route
	var methods []string
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

			if strings.Index(tagName, ",") != -1 {
				//has multi methods seperate by commas

				if !strings.Contains(avalaibleMethodsStr, tagName) {
					//wrong methods passed
					errMessage = errMessage + "\niris.HandleAnnotated: Wrong methods passed to Handler -> " + tagName
					continue
				}

				methods = strings.Split(tagName, ",")
				break
			} else {
				//it is single 'GET','POST' .... method
				methods = []string{tagName}
				break

			}

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
			route = r.HandleFunc(path, convertToHandler(handleFunc.Interface()))
			route.Methods(methods...)
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
func (s *Server) handleAnnotated(irisHandler Annotated) (*Route, error) {
	return s.router.handleAnnotated(irisHandler)
}

///////////////////
//global middleware
///////////////////

// Use registers a a custom handler, with next, as a global middleware
func (r *Router) Use(handler MiddlewareHandler) *Router {
	r.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(r.routes) > 0 {
		for _, routes := range r.routes {
			for _, route := range routes.routes {
				route.Use(handler)
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

// find returns the correct/matched route, if any, for  the request
// if error route != nil , then the errorcode will be 200 OK
func (r *Router) find(req *http.Request) (*Route, int) {

	for _, prefRoute := range r.routes {
		if strings.HasPrefix(req.URL.Path, prefRoute.prefix) {

			wrongMethod := false

			for _, route := range prefRoute.routes {

				if route.match(req.URL.Path) {
					if route.containsMethod(req.Method) == false {
						//if route has found but with wrong method, we must continue it because maybe the next route has the correct method, but
						wrongMethod = true
						continue //the for _, route
					}
					return route, http.StatusOK

				}

			}
			if wrongMethod {
				return nil, http.StatusMethodNotAllowed
			}
		}
	}

	//here if no method found
	return nil, http.StatusNotFound
}
