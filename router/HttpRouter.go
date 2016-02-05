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
func (this *HttpRouter) Route(method string, registedPath string, handler Handler) *HttpRoute {
	var route *HttpRoute
	if registedPath == "" {
		registedPath = "/"
	}
	if method == "" {
		method = HttpMethods.GET
	}
	if handler != nil {
		route = NewHttpRoute(method, registedPath, handler)
		this.routes = append(this.routes, route)
	}
	return route
}

func (route *HttpRoute) Match(urlPath string) bool {
	return route.Pattern.MatchString(urlPath)
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

//Here returns the error code if no route found
func (this *HttpRouter) Find(req *http.Request) (*HttpRoute, int) {
	reqUrlPath := req.URL.Path
	wrongMethod := false
	for _, route := range this.routes {
		if route.Match(reqUrlPath) {

			if req.Method != route.Method {
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
