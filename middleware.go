package iris

import (
	"net/http"
)

type IMiddlewareSupporter interface {
	Use(handler MiddlewareHandler)
	UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc))
	UseHandler(handler http.Handler)
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
func (ms *MiddlewareSupporter) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	ms.Use(MiddlewareHandlerFunc(handlerFunc))
}

// UseHandler creates and adds a http.Handler to the middlewareHandlers collection
func (ms *MiddlewareSupporter) UseHandler(handler http.Handler) {
	convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(res, req)
		//run the next automatically after this handler finished
		next(res, req)

	})

	ms.Use(convertedMiddleware)
}

// MiddlewareHandler is an interface which expects a ServeHTTP function with response,request and a next http.HandlerFunc
type MiddlewareHandler interface {
	ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)
}

// MiddlewareHandlerFunc is just the type of the function which is been expected on the MiddlewareHandler interface
type MiddlewareHandlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)

// ServeHTTP here is just useful to the MiddlewareHandlerFunc acts like a normal net/http handler
func (mh MiddlewareHandlerFunc) ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	mh(res, req, next)
}

// Middleware is the struct which holds a MiddlewareHandler and the next *Middleware of it
type Middleware struct {
	Handler MiddlewareHandler
	Next    *Middleware
}

// ServeHTTP here is just useful to the Middleware acts like a normal net/http handler
func (m Middleware) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.Handler.ServeHTTP(res, req, m.Next.ServeHTTP)
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
		Handler: MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {}),
		Next:    &Middleware{},
	}
}

/* no difference at the heap size...
var emptyMiddleware = Middleware{
	Handler: MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {}),
	Next:    &Middleware{},
}*/
