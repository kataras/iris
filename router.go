package iris

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const (
	// CookieName is the name of the cookie which this frameworks sends to the temporary request in order to get the named parameters
	CookieName = "____iris____"
)

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	MiddlewareSupporter
	//routes map[string]*Route, I dont need this anymore because I will have to iterate to all of them to check the regex pattern vs request url..
	routes []*Route
	mu     sync.RWMutex
}

// NewRouter creates and returns an empty Router
func newRouter() *Router {
	return &Router{routes: make([]*Route, 0)}
}

// Handle registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the name of the route + the pattern
//
// Handle is exported for the future, not being used outside of the iris package yet, some of other functions also.
func (r *Router) Handle(registedPath string, handler HTTPHandler, methods ...string) *Route {
	r.mu.Lock()
	defer r.mu.Unlock()
	var route *Route
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil || registedPath == MatchEverything {

		//validate the handler to be a func

		if reflect.TypeOf(handler).Kind() != reflect.Func {
			panic("iris | Router.go:50 -- Handler HAS TO BE A func")
		}

		//I will do it inside the Prepare, because maybe developer don't wants the GET if methods not defined yet.
		//		if methods == nil {
		//			methods = []string{HttpMethods.GET}
		//		}

		route = newRoute(registedPath, handler, methods...)

		if len(r.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = r.middlewareHandlers
		}

		r.routes = append(r.routes, route)
	}
	return route
}

// RegisterHandler registers a route handler using a Struct
// implements Handle() function and has iris.Handler anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (r *Router) RegisterHandler(irisHandler Handler) (*Route, error) {
	var route *Route
	var methods []string
	var path string
	var handleFunc reflect.Value
	var template string
	var templateIsGLob = false
	var err = errors.New("")
	val := reflect.ValueOf(irisHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Name == "Handler" {
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
					err = errors.New(err.Error() + "\niris.RegisterHandler: Error on getting template: " + templateUnqerr.Error())
					continue
				}

				template = temlateTagValue
			}

			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

			if unqerr != nil {
				err = errors.New(err.Error() + "\niris.RegisterHandler: Error on getting path: " + unqerr.Error())
				continue
			}

			path = tagValue

			if strings.Index(tagName, ",") != -1 {
				//has multi methods seperate by commas

				if !strings.Contains(avalaibleMethodsStr, tagName) {
					//wrong methods passed
					err = errors.New(err.Error() + "\niris.RegisterHandler: Wrong methods passed to Handler -> " + tagName)
					continue
				}

				methods = strings.Split(tagName, ",")
				err = nil
				break
			} else {
				//it is single 'GET','POST' .... method
				methods = []string{tagName}
				err = nil
				break

			}

		}

	}

	if err == nil {
		//route = r.server.Router.Route(path, irisHandler.Handle, methods...)

		//now check/get the Handle method from the irisHandler 'obj'.
		handleFunc = reflect.ValueOf(irisHandler).MethodByName("Handle")

		if !handleFunc.IsValid() {
			err = errors.New("Missing Handle function inside iris.Handler")
		}

		if err == nil {
			route = r.Handle(path, handleFunc.Interface(), methods...)
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

	return route, err
}

func (r *Router) getRouteByRegistedPath(registedPath string) *Route {

	for _, route := range r.routes {
		if route.path == registedPath {
			return route
		}
	}
	return nil

}

///////////////////
//global middleware
///////////////////

// Use registers a a custom handler, with next, as a global middleware
func (r *Router) Use(handler MiddlewareHandler) *Router {
	r.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(r.routes) > 0 {
		for _, route := range r.routes {
			route.Use(handler)
		}
	}
	return r
}

//find returns the correct/matched route, if any, for  the request passed as parameter
func (r *Router) find(req *http.Request) (*Route, int) {
	reqURLPath := req.URL.Path
	wrongMethod := false
	for _, route := range r.routes {
		if route.match(reqURLPath) {
			if route.containsMethod(req.Method) == false {
				wrongMethod = true
				continue
			}

			reqPathSplited := strings.Split(reqURLPath, "/")
			routePathSplited := strings.Split(route.path, "/")
			/*if len(reqPathSplited) != len(reqPathSplited) {
				panic("This error has no excuse, line 99 iris/router/Router.go")
				continue
			}*/
			var cookieFullValue string
			for _, key := range route.ParamKeys {

				for splitIndex, pathPart := range routePathSplited {
					//	pathPart = pathPart. //here must be replace :name(dsadsa) to name in order to comprae it with the key
					hasRegex := strings.Contains(pathPart, ParameterPatternStart) // polu proxeira...

					if (hasRegex && strings.Contains(pathPart, ParameterStart+key+ParameterPatternStart)) || (!hasRegex && strings.Contains(pathPart, ParameterStart+key)) {
						param := key + "=" + reqPathSplited[splitIndex]
						cookieFullValue += "," + param
					}
				}
			}
			if cookieFullValue != "" {
				_cookie := &http.Cookie{Name: CookieName, Value: cookieFullValue[1:]} //remove the first comma
				req.AddCookie(_cookie)
			}

			//break
			return route, 0
		}
	}

	//if route has found but with wrong method, we must continue it because maybe the next route has the correct method, but
	//here if no method found
	if wrongMethod {
		return nil, 405
	}
	//not found
	return nil, 404
}
