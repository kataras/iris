package router

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	REGEX_BRACKETS_CONTENT = "{(.*?)}"
	MATCH_EVERYTHING       = "*"
)

type HTTPRoute struct {
	//Middleware
	MiddlewareSupporter

	methods   []string
	Path      string
	Handler   Handler
	Pattern   *regexp.Regexp
	ParamKeys []string

	isReady bool
}

func NewHTTPRoute(registedPath string, handler Handler, methods ...string) *HTTPRoute {
	if methods == nil {
		methods = make([]string, 0)
	}
	httpRoute := &HTTPRoute{Handler: handler, Path: registedPath, methods: methods, isReady: false}
	if registedPath != MATCH_EVERYTHING {
		pattern := regexp.MustCompile(REGEX_BRACKETS_CONTENT)                  //fint all {key}
		var regexpRoute = pattern.ReplaceAllString(registedPath, "\\w+") + "$" //replace that {key} with /w+ and on the finish $
		regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1)             //escape / character for regex
		routePattern := regexp.MustCompile(regexpRoute)
		httpRoute.Pattern = routePattern
		httpRoute.ParamKeys = pattern.FindAllString(registedPath, -1)
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
	return route.Path == MATCH_EVERYTHING || route.Pattern.MatchString(urlPath)
}

//Runs once before the first ServeHTTP
func (this *HTTPRoute) Prepare() {
	if this.Handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			this.Handler(res, req)
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
	if this.isReady == false && this.Handler != nil {
		this.Prepare()
	}
	this.middleware.ServeHTTP(res, req)
}
