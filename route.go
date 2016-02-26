package iris

import (
	"net/http"
	"regexp"
	"sync"
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {

	//Middleware
	MiddlewareSupporter
	mu            sync.RWMutex
	methods       []string
	path          string
	handler       Handler
	Pattern       *regexp.Regexp
	ParamKeys     []string
	isReady       bool
	templates     *TemplateCache //this is passed to the Renderer
	errorHandlers ErrorHandlers  //the only need of this is to pass into the Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context

}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	Route := &Route{handler: handler, path: registedPath, isReady: false}
	makePathPattern(Route) //moved to RegexHelper.go

	//26-02-2016 handler can be a struct too which have a run(*route,response,request) method
	/*if Route.handler != nil {
		typeFn := reflect.TypeOf(Route.handler)
		if typeFn.NumIn() == 0 {
			//no parameters passed to the route, then panic.
			panic("iris: Route handler: Provide parameters to the handler, otherwise the route cannot be served")
		}
	}*/

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
	if r.methods == nil {
		r.methods = make([]string, 0)
	}
	r.methods = append(r.methods, methods...)
	return r
}

// Method SETS a method to its registed http methods, overrides the previous methods registed (if any)
func (r *Route) Method(method string) *Route {
	r.methods = []string{HTTPMethods.GET}
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

	/*	if r.handlerAcceptsBothContextRenderer {
			ctx := newContext(res, req, r.errorHandlers)
			renderer := newRenderer(res)
			if r.templates != nil {
				renderer.templateCache = r.templates
			}

			r.handler.(func(context *Context, renderer *Renderer))(ctx, renderer)
		} else if r.handlerAcceptsBothResponseRequest {
			r.handler.(func(res http.ResponseWriter, req *http.Request))(res, req)
		} else if r.handlerAcceptsOnlyContext {
			ctx := newContext(res, req, r.errorHandlers)
			r.handler.(func(context *Context))(ctx)
		} else if r.handlerAcceptsOnlyRenderer {
			renderer := newRenderer(res)
			if r.templates != nil {
				renderer.templateCache = r.templates
			}
			r.handler.(func(context *Renderer))(renderer)
		}
	*/
}

// prepare prepares the route's handler , places it to the last middleware , handler acts like a middleware too.
// Runs once before the first ServeHTTP
func (r *Route) prepare() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			//r.Handler(res, req) :->
			//r.run(res, req)
			r.handler.run(r, res, req)
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
