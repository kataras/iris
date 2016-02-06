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

type HttpRoute struct {
	//Middleware
	MiddlewareSupporter

	methods   []string
	Path      string
	Handler   Handler
	Pattern   *regexp.Regexp
	ParamKeys []string

	isReady bool
}

func NewHttpRoute(registedPath string, handler Handler, methods ...string) *HttpRoute {
	if methods == nil {
		methods = make([]string, 0)
	}
	httpRoute := &HttpRoute{Handler: handler, Path: registedPath, methods: methods, isReady: false}
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

func (this *HttpRoute) ContainsMethod(method string) bool {
	for _, m := range this.methods {
		if m == method {
			return true
		}
	}
	return false
}

func (this *HttpRoute) Methods(methods ...string) *HttpRoute {
	this.methods = append(this.methods, methods...)
	return this
}

func (route *HttpRoute) Match(urlPath string) bool {
	return route.Path == MATCH_EVERYTHING || route.Pattern.MatchString(urlPath)
}

//Runs once before the first ServeHTTP
func (this *HttpRoute) Prepare() {
	if this.Handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			this.Handler(res, req)
			next(res, req)
		})

		this.Use(convertedMiddleware)
	}
	
	//here if no methods are defined at all, then use GET by default.
	if this.methods == nil {
		this.methods = []string{HttpMethods.GET}
	}

	this.isReady = true
}

//

func (this *HttpRoute) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if this.isReady == false && this.Handler != nil {
		this.Prepare()
	}
	this.middleware.ServeHTTP(res, req)
}
