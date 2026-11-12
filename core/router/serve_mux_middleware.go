package router

import (
	"github.com/kataras/iris/v14/context"
)

type (
	// serveMuxWrapper is a ServeMux middleware.
	serveMuxMiddleware struct {
		ServeMux

		route *Route
	}
)

var _ ServeMux = (*serveMuxMiddleware)(nil)

// newServeMuxMiddleware returns a new ServeMux which can work as a middleware for the provided ServeMux.
func newServeMuxMiddleware(mux ServeMux, preRoutingRoute *Route) ServeMux {
	if preRoutingRoute == nil || preRoutingRoute.HandlerList.Count() == 0 {
		return mux
	}

	preRoutingRoute.HandlerList.Append(mux.HandleRequest)
	preRoutingRoute.BuildHandlers()

	return &serveMuxMiddleware{
		ServeMux: mux,
		route:    preRoutingRoute,
	}
}

func (h *serveMuxMiddleware) HandleRequest(ctx *context.Context) {
	ctx.SetCurrentRoute(h.route)
	// Note: error handlers will be the same, routes came from the same router,
	// no need to update them.
	ctx.HandlerIndex(0)
	ctx.Do(h.route.Handlers)
	// Execute the handlers, they may proceed to the next handler, the latest one is the router itself.
	// ctx.Do(h.handlers) // TODO: here MAYBE BUT MAYBE NOT NEED TEST AFTER FIXING COM ERRORS: ctx.Next() and defer operations may not work as HandleRequest re-sets the handlers, maybe we have to append there? see switch_hosts_test.go#131.
	// if ctx.HandlerIndex(-1) >= len(h.handlers) {

	// }
	/* No, better idea: just add the handlers to the context itself, and execute them before the ServeMux.HandleRequest.
	// Execute the handlers, they may proceed to the next handler.
	ctx.Do(h.handlers)
	// Check if the execution was stopped by any of the above handlers, and return here if so.
	if ctx.IsStopped() {
		return
	}
	// Reset the execution flow, so the route's handlers will be executed in right order.
	ctx.HandlerIndex(0)
	ctx.SetHandlers(nil)

	// Continue to the serve mux itself.
	h.ServeMux.HandleRequest(ctx)
	*/
}
