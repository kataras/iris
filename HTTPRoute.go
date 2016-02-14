package gapi

import (
	"net/http"
	"reflect"
	"regexp"
	"sync"
)

type HTTPRoute struct {

	//Middleware
	MiddlewareSupporter
	mu        sync.RWMutex
	methods   []string
	path      string
	handler   HTTPHandler
	Pattern   *regexp.Regexp
	ParamKeys []string
	isReady   bool
	templates *TemplateCache //this is passed to the Renderer
	//
	handlerAcceptsOnlyContext         bool
	handlerAcceptsOnlyRenderer        bool
	handlerAcceptsBothContextRenderer bool
	handlerAcceptsBothResponseRequest bool
	//

}

func NewHTTPRoute(registedPath string, handler HTTPHandler, methods ...string) *HTTPRoute {
	if methods == nil {
		methods = make([]string, 0)
	}
	httpRoute := &HTTPRoute{handler: handler, path: registedPath, methods: methods, isReady: false}
	makePathPattern(httpRoute) //moved to RegexHelper.go

	if httpRoute.handler != nil {
		typeFn := reflect.TypeOf(httpRoute.handler)
		if typeFn.NumIn() == 0 {
			//no parameters passed to the route, then panic.
			panic("gapi: HTTPRoute handler: Provide parameters to the handler, otherwise the route cannot be served")
		}
		///Maybe at the future change it to a static type check no just a string because developer may use other Context from other package... I dont know lawl
		if hasContextAndRenderer(typeFn) {
			httpRoute.handlerAcceptsBothContextRenderer = true
		} else if typeFn.NumIn() == 2 { //has two parameters but they are not context and render
			httpRoute.handlerAcceptsBothResponseRequest = true
		} else if hasContextParam(typeFn) { //has only one parameter which is *Context
			httpRoute.handlerAcceptsOnlyContext = true
		} else if hasRendererParam(typeFn) { //has one parameter, it's not *Context, then maybe it's Renderer
			httpRoute.handlerAcceptsOnlyRenderer = true
		} else {
			//panic wrong parameters passed
			panic("gapi: HTTPRoute handler: Wrong parameters passed to the handler, pelase refer to the docs")
		}
	}

	return httpRoute
}

func (this *HTTPRoute) ContainsMethod(method string) bool {
	for _, m := range this.methods {
		if m == method {
			return true
		}
	}
	return false
}

func (this *HTTPRoute) Methods(methods ...string) *HTTPRoute {
	this.methods = append(this.methods, methods...)
	return this
}

func (route *HTTPRoute) Match(urlPath string) bool {
	return route.path == MATCH_EVERYTHING || route.Pattern.MatchString(urlPath)
}

func (route *HTTPRoute) Template() *TemplateCache {
	if route.templates == nil {
		route.templates = NewTemplateCache()
	}
	return route.templates
}

//Here to check for parameters passed to the Handler with ...interface{}
func (this *HTTPRoute) run(res http.ResponseWriter, req *http.Request) {
	//var some []reflect.Value

	if this.handlerAcceptsBothContextRenderer {
		ctx := NewContext(res, req)
		renderer := NewRenderer(res)
		if this.templates != nil {
			renderer.templateCache = this.templates
		}

		this.handler.(func(context *Context, renderer *Renderer))(ctx, renderer)
	} else if this.handlerAcceptsBothResponseRequest {
		this.handler.(func(res http.ResponseWriter, req *http.Request))(res, req)
	} else if this.handlerAcceptsOnlyContext {
		ctx := NewContext(res, req)
		this.handler.(func(context *Context))(ctx)
	} else if this.handlerAcceptsOnlyRenderer {
		renderer := NewRenderer(res)
		if this.templates != nil {
			renderer.templateCache = this.templates
		}
		this.handler.(func(context *Renderer))(renderer)
	}

}

//Runs once before the first ServeHTTP
func (this *HTTPRoute) Prepare() {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			//this.Handler(res, req) :->
			this.run(res, req)
			next(res, req)
		})

		this.Use(convertedMiddleware)
	}

	//here if no methods are defined at all, then use GET by default.
	if this.methods == nil {
		this.methods = []string{HTTPMethods.GET}
	}

	this.isReady = true
}

//

func (this *HTTPRoute) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if this.isReady == false && this.handler != nil {
		this.Prepare()
	}
	this.middleware.ServeHTTP(res, req)
}
