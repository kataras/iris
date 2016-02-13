package gapi

import (
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

const (
	REGEX_BRACKETS_CONTENT    = "{(.*?)}" //{(.*?)}
	REGEX_PARENTHESIS_CONTENT = "((.*?))"
	MATCH_EVERYTHING          = "*"
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
	makePathPattern(httpRoute)

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

func makePathPattern(httpRoute *HTTPRoute) {
	registedPath := httpRoute.path
	if registedPath != MATCH_EVERYTHING {
		regexpRoute := registedPath
		pattern := regexp.MustCompile(REGEX_BRACKETS_CONTENT) //fint all {key}
		keys := pattern.FindAllString(registedPath, -1)
		for indexKey, key := range keys {
			backupKey := key // the full {name(regex)} we will need it for the replace.
			key = key[1 : len(key)-1]
			keys[indexKey] = key
			startParenthesisIndex := strings.Index(key, "(")
			finishParenthesisIndex := strings.LastIndex(key, ")") // checks only the first (), if more than one (regex) exists for one key then the application will be fail and I dont care :)
			//I did LastIndex because the custom regex maybe has ()parenthesis too.
			if startParenthesisIndex > 0 && finishParenthesisIndex > startParenthesisIndex {
				keyPattern := key[startParenthesisIndex+1 : finishParenthesisIndex]
				key = key[0:startParenthesisIndex] //remove the (regex) from key and  the {, }

				keys[indexKey] = key
				if isSupportedType(keyPattern) {
					//if it is (string) or (int) inside contents
					keyPattern = toPattern(keyPattern)
				}
				regexpRoute = strings.Replace(registedPath, backupKey, keyPattern, -1)
				//println("regex found for "+key)
			} else {

				//if no regex found in this key then add the w+
				regexpRoute = strings.Replace(regexpRoute, backupKey, "\\w+", -1)

			}
		}

		//regexpRoute = pattern.ReplaceAllString(registedPath, "\\w+") + "$" //replace that {key} with /w+ and on the finish $
		regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1) + "$" ///escape / character for regex and finish it with $, if route/{name} and req url is route/{name}/somethingelse then it will not be matched
		routePattern := regexp.MustCompile(regexpRoute)
		httpRoute.Pattern = routePattern

		httpRoute.ParamKeys = keys
	}
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
