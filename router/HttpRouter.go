package router

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	COOKIE_NAME            = "____gapi____"
	REGEX_BRACKETS_CONTENT = "{(.*?)}"
)

type Handler func(http.ResponseWriter, *http.Request)

type Parameters map[string]string

func (params Parameters) Get(key string) string {
	return params[key]
}

type HttpRoute struct {
	Method    string
	Path      string
	Handler   Handler
	Pattern   *regexp.Regexp
	ParamKeys []string
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
func (this *HttpRouter) Route(method string, registedPath string, handler Handler) *HttpRouter {
	if registedPath == "" {
		registedPath = "/"
	}
	if method == "" {
		method = HttpMethods.GET
	}
	if handler != nil {
		//this.routes[registedPath] = NewHttpRoute(method, registedPath, handler)
		this.routes = append(this.routes, NewHttpRoute(method, registedPath, handler))
	}
	return this
}

//Global to package router.
func NewHttpRoute(method string, registedPath string, handler Handler) *HttpRoute {
	httpRoute := &HttpRoute{Method: method, Handler: handler, Path: registedPath}

	pattern := regexp.MustCompile(REGEX_BRACKETS_CONTENT)                  //fint all {key}
	var regexpRoute = pattern.ReplaceAllString(registedPath, "\\w+") + "$" //replace that {key} with /w+ and on the finish $
	regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1)             //escape / character for regex
	routePattern := regexp.MustCompile(regexpRoute)

	httpRoute.Pattern = routePattern
	httpRoute.ParamKeys = pattern.FindAllString(registedPath, -1)

	return httpRoute
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
//the req.Method check goes to the HttpServer, in order to provide the correct http error  405 Method not allowed.
func (this *HttpRouter) Find(req *http.Request) *HttpRoute {
	reqUrlPath := req.URL.Path
	for _, route := range this.routes {
		if route.Match(reqUrlPath) {
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
			return route
		}
	}
	
	return nil
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
