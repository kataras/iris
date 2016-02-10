package gapi

import (
	"net/http"
	"reflect"
	"strings"
	"sync"
)

const (
	COOKIE_NAME = "____gapi____"
)

//type HTTPHandler func(http.ResponseWriter, *http.Request)
type HTTPHandler interface{} //func(...interface{}) //[]reflect.Value

type Parameters map[string]string

func (params Parameters) Get(key string) string {
	return params[key]
}

type HTTPRouter struct {
	MiddlewareSupporter
	//routes map[string]*HttpRoute, I dont need this anymore because I will have to iterate to all of them to check the regex pattern vs request url..
	routes []*HTTPRoute
	mu     sync.RWMutex
}

func NewHTTPRouter() *HTTPRouter {
	return &HTTPRouter{routes: make([]*HTTPRoute, 0)}
}

func (this *HTTPRouter) Unroute(urlPath string) *HTTPRouter {
	//delete(this.routes, urlPath)
	///TODO
	return this
}

//registedPath is the name of the route + the pattern
func (this *HTTPRouter) Route(registedPath string, handler HTTPHandler, methods ...string) *HTTPRoute {
	this.mu.Lock()
	defer this.mu.Unlock()
	var route *HTTPRoute
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil || registedPath == MATCH_EVERYTHING {

		//validate the handler to be a func

		if reflect.TypeOf(handler).Kind() != reflect.Func {
			panic("gapi | HTTPRouter.go:50 -- Handler HAS TO BE A func")
		}

		//I will do it inside the Prepare, because maybe developer don't wants the GET if methods not defined yet.
		//		if methods == nil {
		//			methods = []string{HttpMethods.GET}
		//		}

		route = NewHTTPRoute(registedPath, handler, methods...)

		if len(this.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = this.middlewareHandlers
		}

		this.routes = append(this.routes, route)
	}
	return route
}

func (this *HTTPRouter) getRouteByRegistedPath(registedPath string) *HTTPRoute {

	for _, route := range this.routes {
		if route.path == registedPath {
			return route
			break
		}
	}
	return nil

}

/* GLOBAL MIDDLEWARE */

func (this *HTTPRouter) Use(handler MiddlewareHandler) *HTTPRouter {
	this.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(this.routes) > 0 {
		for _, route := range this.routes {
			route.Use(handler)
		}
	}
	return this
}

//

//Here returns the error code if no route found
func (this *HTTPRouter) Find(req *http.Request) (*HTTPRoute, int) {
	reqUrlPath := req.URL.Path
	wrongMethod := false
	for _, route := range this.routes {
		if route.Match(reqUrlPath) {
			if route.ContainsMethod(req.Method) == false {
				wrongMethod = true
				continue
			}

			reqPathSplited := strings.Split(reqUrlPath, "/")
			routePathSplited := strings.Split(route.path, "/")
			/*if len(reqPathSplited) != len(reqPathSplited) {
				panic("This error has no excuse, line 99 gapi/router/HttpRouter.go")
				continue
			}*/

			for _, key := range route.ParamKeys {

				for splitIndex, pathPart := range routePathSplited {
					//	pathPart = pathPart. //here must be replace {name(dsadsa)} to name in order to comprae it with the key
					hasRegex := strings.Contains(pathPart, "(") // polu proxeira...
					if (hasRegex && strings.Contains(pathPart, "{"+key+"(")) || (!hasRegex && strings.Contains(pathPart, "{"+key)) {
						param := key + "=" + reqPathSplited[splitIndex]
						_cookie := &http.Cookie{Name: COOKIE_NAME, Value: param}
						req.AddCookie(_cookie)

					}

				}
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

//Global to package
func Params(req *http.Request) Parameters {
	_cookie, _err := req.Cookie(COOKIE_NAME)
	if _err != nil {
		return nil
	}
	value := _cookie.Value
	params := make(Parameters)

	paramsStr := strings.Split(value, ",")
	for _, _fullVarStr := range paramsStr {
		vars := strings.Split(_fullVarStr, "=")
		if len(vars) != 2 { //check if key=val=somethingelse here ,is wrong, only key=value allowed, then just ignore this
			continue
		}
		params[vars[0]] = vars[1]
	}

	return params
}

func Param(req *http.Request, key string) string {
	params := Params(req)
	param := ""
	if params != nil {
		param = params[key]
	}
	return param
}
