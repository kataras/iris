package iris

import (
	"net/http"
	"reflect"
	"regexp"
	"sync"
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {

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

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler HTTPHandler, methods ...string) *Route {
	if methods == nil {
		methods = make([]string, 0)
	}
	Route := &Route{handler: handler, path: registedPath, methods: methods, isReady: false}
	makePathPattern(Route) //moved to RegexHelper.go

	if Route.handler != nil {
		typeFn := reflect.TypeOf(Route.handler)
		if typeFn.NumIn() == 0 {
			//no parameters passed to the route, then panic.
			panic("iris: Route handler: Provide parameters to the handler, otherwise the route cannot be served")
		}
		///Maybe at the future change it to a static type check no just a string because developer may use other Context from other package... I dont know lawl
		if hasContextAndRenderer(typeFn) {
			Route.handlerAcceptsBothContextRenderer = true
		} else if typeFn.NumIn() == 2 { //has two parameters but they are not context and render
			Route.handlerAcceptsBothResponseRequest = true
		} else if hasContextParam(typeFn) { //has only one parameter which is *Context
			Route.handlerAcceptsOnlyContext = true
		} else if hasRendererParam(typeFn) { //has one parameter, it's not *Context, then maybe it's Renderer
			Route.handlerAcceptsOnlyRenderer = true
		} else {
			//panic wrong parameters passed
			panic("iris: Route handler: Wrong parameters passed to the handler, pelase refer to the docs")
		}
	}

	return Route
}

// containsMethod determinates if this route contains a http method
func (r *Route) containsMethod(method string) bool {
	for _, m := range r.methods {
		if m == method {
			return true
		}
	}
	return false
}

// Methods adds methods to its registed http methods
func (r *Route) Methods(methods ...string) *Route {
	r.methods = append(r.methods, methods...)
	return r
}

// Match determinates if this route match with a url
func (r *Route) match(urlPath string) bool {
	return r.path == MatchEverything || r.Pattern.MatchString(urlPath)
}

// Template creates (if not exists) and returns the template cache for this route
func (r *Route) Template() *TemplateCache {
	if r.templates == nil {
		r.templates = NewTemplateCache()
	}
	return r.templates
}

// run runs the route, this means response to the client's Request.
// Here is the place for checking for parameters passed to the Handler with ...interface{}
func (r *Route) run(res http.ResponseWriter, req *http.Request) {
	//var some []reflect.Value

	if r.handlerAcceptsBothContextRenderer {
		ctx := NewContext(res, req)
		renderer := NewRenderer(res)
		if r.templates != nil {
			renderer.templateCache = r.templates
		}

		r.handler.(func(context *Context, renderer *Renderer))(ctx, renderer)
	} else if r.handlerAcceptsBothResponseRequest {
		r.handler.(func(res http.ResponseWriter, req *http.Request))(res, req)
	} else if r.handlerAcceptsOnlyContext {
		ctx := NewContext(res, req)
		r.handler.(func(context *Context))(ctx)
	} else if r.handlerAcceptsOnlyRenderer {
		renderer := NewRenderer(res)
		if r.templates != nil {
			renderer.templateCache = r.templates
		}
		r.handler.(func(context *Renderer))(renderer)
	}

}

// prepare prepares the route's handler , places it to the last middleware , handler acts like a middleware too.
// Runs once before the first ServeHTTP
func (r *Route) prepare() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			//r.Handler(res, req) :->
			r.run(res, req)
			next(res, req)
		})

		r.Use(convertedMiddleware)
	}

	//here if no methods are defined at all, then use GET by default.
	if r.methods == nil {
		r.methods = []string{HTTPMethods.GET}
	}

	r.isReady = true
}

// ServeHTTP serves this route and it's middleware
func (r *Route) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if r.isReady == false && r.handler != nil {
		r.prepare()
	}
	r.middleware.ServeHTTP(res, req)
}
