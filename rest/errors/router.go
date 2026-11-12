package errors

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/funcer"
	"github.com/kataras/iris/v14/core/router"
)

// Router is a wrapper around the `router.Router`.
// It holds the underline core Router and provides an easy way to register routes that return an error,
// this error is automatically handled by the framework.
//
// All router.Router's methods that accept a context/iris.Handler are converted to accept a `Func`
// (func(*context.Context) error) instead.
type Router struct {
	router.Router // the underline core Router, embedded to expose all of its methods.
}

// NewRouter returns a new instance of the Router.
// It accepts a router.Router and returns a new Router.
// It's used to wrap the underline core router.Router with a Router that can register handlers that return an error.
//
// Read more at the `Router` type itself.
func NewRouter(r router.Router) *Router {
	return &Router{r}
}

var handlerPC = funcer.NewProgramCounter[Func](funcer.NewNameResolver())

// Handle registers a route to the server's router.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Handle method and returns its output argument.
func (r *Router) Handle(method, path string, funcs ...Func) *router.RouteRegistry {
	handlers := FuncHandlers(funcs...)
	routes := r.Router.Handle(method, path, handlers...)

	for _, route := range routes.List() {
		for i, fn := range route.MainHandlerList.List() { // same handlers and order with the given "funcs".
			originalInfo := handlerPC.Resolve(funcs[i])

			fn.File = originalInfo.File
			fn.FileRel = originalInfo.FileRel
			fn.Line = originalInfo.Line
			fn.Name = originalInfo.Name
			fn.Meta = originalInfo.Meta
		}
	}

	return routes
}

// // HandleNoRoute registers a handler for 404 status code, no route found.
// func (r *Router) HandleNoRoute(statusCode int, funcs ...Func) {
// 	handlers := Handlers(funcs...)
// 	r.Router.HandleNoRoute(statusCode, handlers...)
// }
// // HandleNoMethod registers a handler for the 405 http error status code.
// func (r *Router) HandleNoMethod(funcs ...Func) {
// 	handlers := Handlers(funcs...)
// 	r.Router.HandleNoMethod(handlers...)
// }

// Get registers a route for the Get http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Get method and returns its output argument.
func (r *Router) Get(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodGet, path, funcs...).First()
}

// Post registers a route for the Post http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Post method and returns its output argument.
func (r *Router) Post(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodPost, path, funcs...).First()
}

// Query registers a route for the QUERY http method (RFC 10008).
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Query method and returns its output argument.
func (r *Router) Query(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodQuery, path, funcs...).First()
}

// Put registers a route for the Put http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Put method and returns its output argument.
func (r *Router) Put(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodPut, path, funcs...).First()
}

// Delete registers a route for the Delete http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Delete method and returns its output argument.
func (r *Router) Delete(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodDelete, path, funcs...).First()
}

// Connect registers a route for the Connect http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Connect method and returns its output argument.
func (r *Router) Connect(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodConnect, path, funcs...).First()
}

// Head registers a route for the Head http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Head method and returns its output argument.
func (r *Router) Head(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodHead, path, funcs...).First()
}

// Options registers a route for the Options http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Options method and returns its output argument.
func (r *Router) Options(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodOptions, path, funcs...).First()
}

// Patch registers a route for the Patch http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Patch method and returns its output argument.
func (r *Router) Patch(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodPatch, path, funcs...).First()
}

// Trace registers a route for the Trace http method.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Trace method and returns its output argument.
func (r *Router) Trace(path string, funcs ...Func) *router.Route {
	return r.Handle(router.MethodTrace, path, funcs...).First()
}

// Any registers a route for all http methods.
// It converts the funcs variadic input to context.Handlers through `FuncHandlers`
// and then it calls the underline's Router.Any method and returns its output argument.
func (r *Router) Any(path string, funcs ...Func) *router.RouteRegistry {
	handlers := FuncHandlers(funcs...)
	return r.Router.Any(path, handlers...)
}

// Subdomain returns a new Router instance which is a subdomain of the current Router.
func (r *Router) Subdomain(subdomain string, middlewareFuncs ...Func) *Router {
	middleware := FuncHandlers(middlewareFuncs...)
	sub := r.Router.Subdomain(subdomain, middleware...)
	return NewRouter(sub)
}

// UseRouter registers middleware that runs before the routing itself:
// on every request under this router's path prefix, including unmatched
// routes (404) and method-not-allowed (405) requests. On a nil error the
// request continues to the next handler in the chain; a non-nil error is
// handled by HandleError and stops the request.
func (r *Router) UseRouter(funcs ...Func) {
	for _, fn := range funcs {
		if fn == nil {
			continue
		}

		r.Router.UseRouter(FuncMiddleware(fn))
	}
}

// Use registers a middleware to the router that runs after UseRouter but before main handlers.
func (r *Router) Use(funcs ...Func) {
	handlers := FuncHandlers(funcs...)
	r.Router.Use(handlers...)
}

// UseOnce same as `Use` but the same handler cannot be registered twice.
func (r *Router) UseOnce(funcs ...Func) {
	handlers := FuncHandlers(funcs...)
	r.Router.UseOnce(handlers...)
}

type (
	// ServerHandler is the interface that wraps the HandleRequest method.
	ServerHandler interface {
		HandleRequest(*context.Context) error
	}

	compatibleServerHandler struct {
		handler context.Handler // the converted handler.
	}
)

// HandleRequest is the method that completes the `router.ServerHandler` interface.
func (srv *compatibleServerHandler) HandleRequest(ctx *context.Context) {
	srv.handler(ctx)
}

// HandleServer registers a server handler to the router.
// It converts the server variadic input to a compatibleServerHandler and
// then it calls the underline's Router.HandleServer method.
func (r *Router) HandleServer(path string, server ServerHandler) {
	serverHandler := FuncHandler(server.HandleRequest)
	compatible := &compatibleServerHandler{handler: serverHandler}
	r.Router.HandleServer(path, compatible)
}
