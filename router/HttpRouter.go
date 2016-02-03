package router

import (
	"net/http"
	"strings"
	//"regexp"
)

const COOKIE_NAME = "____gapi____"

type Handler func(http.ResponseWriter, *http.Request)

type Parameters map[string]string

func (params Parameters) Get(key string) string {
	return params[key]
}

type HttpRoute struct {
	Method  string
	Handler Handler
	Pattern string
}

type HttpRouter struct {
	routes map[string]*HttpRoute
}

func NewHttpRouter() *HttpRouter {
	return &HttpRouter{routes: make(map[string]*HttpRoute)}
}

func (this *HttpRouter) Unroute(urlPath string) *HttpRouter {
	delete(this.routes, urlPath)
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
		_path, _pattern := this.parseRoutePath(registedPath)
		//println("path: ",_path ," pattern: " , _pattern)
		this.routes[_path] = &HttpRoute{Method: method, Handler: handler, Pattern: _pattern}
	}
	return this
}

/*
/home/thisIsStatic , if no match then:
/home/{name} set the var name to this
/home/{name:regexp  patter} set the var name to this if the regexp is matches with the requested url
*/

//called only one time per route registration.
//returns the full pattern /home/{paramKey}/{paramKey2}...
//for now works only with /home/{paramKey}
//returns the path and the pattern, ex : /home/{name}  path = /home pattern = {name}
func (this *HttpRouter) parseRoutePath(registedPath string) (string, string) {
	var pattern string
	var thePath string = registedPath
	startPatternIndex := strings.Index(registedPath, "{")
	finishPatternIndex := strings.Index(registedPath, "}")

	//must have slash/ before the {
	if startPatternIndex > 1 && finishPatternIndex > startPatternIndex {
		thePath = registedPath[:startPatternIndex-1] //get the /path not the /path/{name}
		if len(thePath) > 0 {
			pattern = registedPath[startPatternIndex : finishPatternIndex+1]
		}

		//println("startPatternIndex ", startPatternIndex, " finishPatternIndex ", finishPatternIndex, " theSlash is: ", theSlash, " and the pattern: ",pattern)
	}

	return thePath, pattern
}

//Unexported - no regexp yet.
func (this *HttpRouter) match(urlPath string, route HttpRoute) bool {
	return false
}

func (this *HttpRouter) getRoute(s string) *HttpRoute {
	return this.routes[s]
}

func (this *HttpRouter) Find(req *http.Request) *HttpRoute {
	//reqUrlPath = /home/dsadsa
	reqUrlPath := req.URL.Path
	route := this.getRoute(reqUrlPath)

	if route != nil && route.Pattern != "" {
		//means registed route: /home/{name} , reqUri : /home , but if no /name then the routing is not valid .
		return nil
	}

	if route == nil {
		//check the /home/...

		var path = reqUrlPath[:strings.LastIndex(reqUrlPath, "/")]
		route = this.getRoute(path) //we get the first part of the path , and the route now must be ok

		if route != nil && route.Pattern != "" { //route.Pattern != "" maybe not nessecary here, but for safety....

			paramValue := reqUrlPath[len(path)+1:] // +1 because of: /test -> test

			if paramValue == "" { //in case of /home/
				return nil
			}

			paramKey := route.Pattern[1 : len(route.Pattern)-1]
			//var paramMap = make(map[string]string)
			//paramMap[paramKey] = paramValue
			param := paramKey + "=" + paramValue
			_cookie := &http.Cookie{Name: COOKIE_NAME, Value: param}

			req.AddCookie(_cookie)
			//println(paramKey," = ", paramValue) //name = test

		}
	}
	return route
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
