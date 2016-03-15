package iris

import (
	"strings"
)

// Route contains basic and temporary info about the route, it is nil after iris.Listen called
// contains all middleware and prepare them for execution
// Used to create a node at the Router's Build state
type Route struct {
	MiddlewareSupporter
	fullpath   string
	handler    Handler
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
