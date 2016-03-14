package iris

import (
	"strings"
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {
	//GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE bool //tried with []string, very slow, tried with map[string]bool gives 10k executions but +20k bytes, with this approact we have to code more but only 1k byte to space and make it 2.2 times faster than before!
	//Middleware
	MiddlewareSupporter
	isStatic      bool
	hasMiddleware bool
	fullpath      string // need only on parameters.Params(...)
	//fullparts   []string
	handler    Handler
	station    *Station
	PathPrefix string
}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	r := &Route{handler: handler, fullpath: registedPath}
	r.processPath()
	return r
}

func (r *Route) processPath() {
	endPrefixIndex := strings.IndexByte(r.fullpath, ParameterStartByte)

	if endPrefixIndex != -1 {
		r.PathPrefix = r.fullpath[:endPrefixIndex]

	} else {
		//check for *
		endPrefixIndex = strings.IndexByte(r.fullpath, MatchEverythingByte)
		if endPrefixIndex != -1 {
			r.PathPrefix = r.fullpath[:endPrefixIndex]
		} else {
			//check for the last slash
			endPrefixIndex = strings.LastIndexByte(r.fullpath, SlashByte)
			if endPrefixIndex != -1 {
				r.PathPrefix = r.fullpath[:endPrefixIndex]
			} else {
				//we don't have ending slash ? then it is the whole r.fullpath
				r.PathPrefix = r.fullpath
			}
		}
	}

	//1.check if pathprefix is empty ( it's empty when we have just '/' as fullpath) so make it '/'
	//2. check if it's not ending with '/', ( it is not ending with '/' when the next part is parameter or *)

	lastIndexOfSlash := strings.LastIndexByte(r.PathPrefix, SlashByte)
	if lastIndexOfSlash != len(r.PathPrefix)-1 || r.PathPrefix == "" {
		r.PathPrefix += "/"
	}
}

// prepare prepares the route's handler , places it to the last middleware , handler acts like a middleware too.
// Runs once at the BuildRouter state, which is part of the Build state at the station.
func (r *Route) prepare() {
	if r.middlewareHandlers != nil {
		r.hasMiddleware = true
	}
	convertedMiddleware := MiddlewareHandlerFunc(func(ctx *Context, next Handler) {
		r.handler.Serve(ctx)
		//except itself
		if r.middlewareHandlers != nil && len(r.middlewareHandlers) > 1 {
			next.Serve(ctx)
		}
	})

	r.Use(convertedMiddleware)

}

// Serve serves this route and it's middleware, anyway it acts like middleware executor
func (r *Route) Serve(ctx *Context) {
	r.middleware.Serve(ctx)
}
