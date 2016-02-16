package gapi

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const (
	COOKIE_NAME = "____gapi____"
)

type Router struct {
	MiddlewareSupporter
	//routes map[string]*Route, I dont need this anymore because I will have to iterate to all of them to check the regex pattern vs request url..
	routes []*Route
	mu     sync.RWMutex
}

func NewRouter() *Router {
	return &Router{routes: make([]*Route, 0)}
}

//registedPath is the name of the route + the pattern
func (this *Router) Handle(registedPath string, handler HTTPHandler, methods ...string) *Route {
	this.mu.Lock()
	defer this.mu.Unlock()
	var route *Route
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil || registedPath == MATCH_EVERYTHING {

		//validate the handler to be a func

		if reflect.TypeOf(handler).Kind() != reflect.Func {
			panic("gapi | Router.go:50 -- Handler HAS TO BE A func")
		}

		//I will do it inside the Prepare, because maybe developer don't wants the GET if methods not defined yet.
		//		if methods == nil {
		//			methods = []string{HttpMethods.GET}
		//		}

		route = NewRoute(registedPath, handler, methods...)

		if len(this.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = this.middlewareHandlers
		}

		this.routes = append(this.routes, route)
	}
	return route
}
func (this *Router) RegisterHandler(gapiHandler Handler) (*Route, error) {
	var route *Route
	var methods []string
	var path string
	var handleFunc reflect.Value
	var template string
	var templateIsGLob bool = false
	var err error = errors.New("")
	val := reflect.ValueOf(gapiHandler).Elem()

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
					err = errors.New(err.Error() + "\ngapi.RegisterHandler: Error on getting template: " + templateUnqerr.Error())
					continue
				}

				template = temlateTagValue
			}

			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

			if unqerr != nil {
				err = errors.New(err.Error() + "\ngapi.RegisterHandler: Error on getting path: " + unqerr.Error())
				continue
			}

			path = tagValue

			if strings.Index(tagName, ",") != -1 {
				//has multi methods seperate by commas

				if !strings.Contains(avalaibleMethodsStr, tagName) {
					//wrong methods passed
					err = errors.New(err.Error() + "\ngapi.RegisterHandler: Wrong methods passed to Handler -> " + tagName)
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
		//route = this.server.Router.Route(path, gapiHandler.Handle, methods...)

		//now check/get the Handle method from the gapiHandler 'obj'.
		handleFunc = reflect.ValueOf(gapiHandler).MethodByName("Handle")

		if !handleFunc.IsValid() {
			err = errors.New("Missing Handle function inside gapi.Handler")
		}

		if err == nil {
			route = this.Handle(path, handleFunc.Interface(), methods...)
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

func (this *Router) getRouteByRegistedPath(registedPath string) *Route {

	for _, route := range this.routes {
		if route.path == registedPath {
			return route
		}
	}
	return nil

}

/* GLOBAL MIDDLEWARE */

func (this *Router) Use(handler MiddlewareHandler) *Router {
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
func (this *Router) Find(req *http.Request) (*Route, int) {
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
				panic("This error has no excuse, line 99 gapi/router/Router.go")
				continue
			}*/
			var cookieFullValue string
			for _, key := range route.ParamKeys {

				for splitIndex, pathPart := range routePathSplited {
					//	pathPart = pathPart. //here must be replace :name(dsadsa) to name in order to comprae it with the key
					hasRegex := strings.Contains(pathPart, PARAMETER_PATTERN_START) // polu proxeira...

					if (hasRegex && strings.Contains(pathPart, PARAMETER_START+key+PARAMETER_PATTERN_START)) || (!hasRegex && strings.Contains(pathPart, PARAMETER_START+key)) {
						param := key + "=" + reqPathSplited[splitIndex]
						cookieFullValue += "," + param
					}
				}
			}
			if cookieFullValue != "" {
				_cookie := &http.Cookie{Name: COOKIE_NAME, Value: cookieFullValue[1:]} //remove the first comma
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
