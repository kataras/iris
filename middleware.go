package iris

// IMiddlewareSupporter is the interface of the middleware 'manager'
type IMiddlewareSupporter interface {
	Use(handler MiddlewareHandler)
	UseFunc(handlerFunc func(ctx *Context, next Handler))
}

// MiddlewareSupporter is been 'injected-oop' in other struct,
// which usage is to support, manage and handle middleware
type MiddlewareSupporter struct {
	IMiddlewareSupporter
	middleware         Middleware
	middlewareHandlers []MiddlewareHandler //at the Route the route handler is the last empty-next 'MiddlewareHandler'.
}

// Use creates and adds a MiddlewareHandler (with next) to the middlewareHandlers collection
func (ms *MiddlewareSupporter) Use(handler MiddlewareHandler) {
	if ms.middlewareHandlers == nil {
		ms.middlewareHandlers = make([]MiddlewareHandler, 0)
	}

	ms.middlewareHandlers = append(ms.middlewareHandlers, handler)
	ms.middleware = makeMiddlewareFor(ms.middlewareHandlers)
}

// UseFunc creates and adds a function to the middlewareHandlers collection
func (ms *MiddlewareSupporter) UseFunc(handlerFunc func(ctx *Context, next Handler)) {
	ms.Use(MiddlewareHandlerFunc(handlerFunc))
}

// MiddlewareHandler is an interface which expects a ServeHTTP function with response,request and a next iris.HandlerFunc
type MiddlewareHandler interface {
	Serve(ctx *Context, next Handler)
}

// MiddlewareHandlerFunc is just the type of the function which is been expected on the MiddlewareHandler interface
type MiddlewareHandlerFunc func(ctx *Context, next Handler)

func (mh MiddlewareHandlerFunc) Serve(ctx *Context, next Handler) {
	mh(ctx, next)
}

// Middleware is the struct which holds a MiddlewareHandler and the next *Middleware of it
type Middleware struct {
	Handler MiddlewareHandler
	Next    *Middleware
}

// This is being called from a succeed request
func (m Middleware) Serve(ctx *Context) {
	m.Handler.Serve(ctx, m.Next)
}

func makeMiddlewareFor(handlers []MiddlewareHandler) Middleware {
	var next Middleware

	if len(handlers) == 0 {
		return emptyMiddleware()
	} else if len(handlers) > 1 {
		next = makeMiddlewareFor(handlers[1:])
	} else {
		next = emptyMiddleware()
	}

	return Middleware{handlers[0], &next}
}

// emptyMiddleware creates a Middleware which as an empty Next Middleware, is been used to define the Handler at the route
func emptyMiddleware() Middleware {
	return Middleware{
		Handler: MiddlewareHandlerFunc(func(ctx *Context, next Handler) {}),
		Next:    &Middleware{},
	}
}
