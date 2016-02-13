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
	mu                    sync.RWMutex
	methods               []string
	path                  string
	handler               HTTPHandler
	Pattern               *regexp.Regexp
	ParamKeys             []string
	handlerAcceptsContext bool
	isReady               bool
	templates              *TemplateCache
}

func NewHTTPRoute(registedPath string, handler HTTPHandler, methods ...string) *HTTPRoute {
	if methods == nil {
		methods = make([]string, 0)
	}
	httpRoute := &HTTPRoute{handler: handler, path: registedPath, methods: methods, isReady: false}
	makePathPattern(httpRoute)

	if httpRoute.handler != nil {
		typeFn := reflect.TypeOf(httpRoute.handler)
		countParams := typeFn.NumIn()

		///Maybe at the future change it to a static type check no just a string because developer may use other Context from other package... I dont know lawl
		if countParams == 1 && strings.Contains(typeFn.In(0).String(), "Context") {
			httpRoute.handlerAcceptsContext = true
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

// 1. function has Parameters type check if the request has parameters then pass them or pass empty map.
// 2. If no res, req (classic http handler) then we check if

//This has to run in every ServeHTTP but we are writing on one place on Prepare() function inside the convertedMiddleware which is running at every request.
//NO  this will run one time, it will create some fields ex: isClassicHttp=true if res,req are the only params
// isWaitingForParameters = true if gapi.Parameters passsed as an argument.
//or no? Compared to a disk seek or network transfer, the cost of reflection will be negligible ... I have to think about it.
//na pernw ta types kai ta parameters mia fora px typeparamSomething =1 //1 position of .In(i)
func (this *HTTPRoute) run(res http.ResponseWriter, req *http.Request) {
	//var some []reflect.Value

	if this.handlerAcceptsContext {
		ctx := NewContext(res, req)
		if this.templates != nil {
			ctx.templateCache = this.templates
		}
		this.handler.(func(context *Context))(ctx)
	} else {
		this.handler.(func(res http.ResponseWriter, req *http.Request))(res, req)
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
