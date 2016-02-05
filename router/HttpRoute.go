package router

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	REGEX_BRACKETS_CONTENT = "{(.*?)}"
)

type HttpRoute struct {
	Method    string
	Path      string
	Handler   Handler
	Pattern   *regexp.Regexp
	ParamKeys []string

	//Middleware
	middleware         Middleware
	middlewareHandlers []MiddlewareHandler

	isReady bool
}

func NewHttpRoute(method string, registedPath string, handler Handler) *HttpRoute {

	httpRoute := &HttpRoute{Method: method, Handler: handler, Path: registedPath, middlewareHandlers: make([]MiddlewareHandler, 0), isReady: false}

	pattern := regexp.MustCompile(REGEX_BRACKETS_CONTENT)                  //fint all {key}
	var regexpRoute = pattern.ReplaceAllString(registedPath, "\\w+") + "$" //replace that {key} with /w+ and on the finish $
	regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1)             //escape / character for regex
	routePattern := regexp.MustCompile(regexpRoute)

	httpRoute.Pattern = routePattern
	httpRoute.ParamKeys = pattern.FindAllString(registedPath, -1)

	return httpRoute
}

//Middleware
func (this *HttpRoute) Use(handler MiddlewareHandler) *HttpRoute {
	this.middlewareHandlers = append(this.middlewareHandlers, handler)
	this.middleware = makeMiddlewareFor(this.middlewareHandlers)

	return this
}

func (this *HttpRoute) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *HttpRoute {
	this.Use(MiddlewareHandlerFunc(handlerFunc))
	return this
}

//Use normal buildin http.Handler
func (this *HttpRoute) UseHandler(handler http.Handler) *HttpRoute {
	convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(res, req)
		//run the next automatically after this handler finished
		next(res, req)
	})

	this.Use(convertedMiddleware)

	return this
}

func (this *HttpRoute) Prepare() {
	if this.Handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			this.Handler(res, req)
			next(res, req)
		})

		this.Use(convertedMiddleware)
	}

	this.isReady = true
}

//

func (this *HttpRoute) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if this.isReady == false {
		this.Prepare()
	}
	this.middleware.ServeHTTP(res, req)
}
