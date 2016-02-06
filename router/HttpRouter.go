package router

import (
	"net/http"
	"strings"
)

const (
	COOKIE_NAME = "____gapi____"
)

type Handler func(http.ResponseWriter, *http.Request)

type Parameters map[string]string

func (params Parameters) Get(key string) string {
	return params[key]
}

type HttpRouter struct {
	MiddlewareSupporter
	//routes map[string]*HttpRoute, I dont need this anymore because I will have to iterate to all of them to check the regex pattern vs request url..
	routes []*HttpRoute
}

func NewHttpRouter() *HttpRouter {
	return &HttpRouter{routes: make([]*HttpRoute, 0)}
}

func (this *HttpRouter) Unroute(urlPath string) *HttpRouter {
	//delete(this.routes, urlPath)
	///TODO
	return this
}

//registedPath is the name of the route + the pattern
func (this *HttpRouter) Route(registedPath string, handler Handler, methods ...string) *HttpRoute {
	var route *HttpRoute
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil || registedPath == MATCH_EVERYTHING {
		//I will do it inside the Prepare, because maybe developer don't wants the GET if methods not defined yet.
		//		if methods == nil {
		//			methods = []string{HttpMethods.GET}
		//		}

		route = NewHttpRoute(registedPath, handler, methods...)

		if len(this.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = this.middlewareHandlers
		}

		this.routes = append(this.routes, route)
	}
	return route
}

func (this *HttpRouter) getRouteByRegistedPath(registedPath string) *HttpRoute {

	for _, route := range this.routes {
		if route.Path == registedPath {
			return route
			break
		}
	}
	return nil

}

/* GLOBAL MIDDLEWARE */

func (this *HttpRouter) Use(handler MiddlewareHandler) *HttpRouter {
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
func (this *HttpRouter) Find(req *http.Request) (*HttpRoute, int) {
	reqUrlPath := req.URL.Path
	wrongMethod := false
	for _, route := range this.routes {
		if route.Match(reqUrlPath) {

			if route.ContainsMethod(req.Method) == false {
				wrongMethod = true
				continue
			}

			reqPathSplited := strings.Split(reqUrlPath, "/")
			routePathSplited := strings.Split(route.Path, "/")
			/*if len(reqPathSplited) != len(reqPathSplited) {
				panic("This error has no excuse, line 99 gapi/router/HttpRouter.go")
				continue
			}*/

			for _, key := range route.ParamKeys {

				for splitIndex, pathPart := range routePathSplited {
					if pathPart == key {
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

//Global to package router.
func GetParameters(req *http.Request) Parameters {
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
