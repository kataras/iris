package gapi

import (
	"net/http"
)

type MiddlewareSupporter struct {
	middleware         Middleware
	middlewareHandlers []MiddlewareHandler //at the HTTPRoute the route handler is the last empty-next 'MiddlewareHandler'.
}

func (this *MiddlewareSupporter) Use(handler MiddlewareHandler) {
	if this.middlewareHandlers == nil {
		this.middlewareHandlers = make([]MiddlewareHandler, 0)
	}

	this.middlewareHandlers = append(this.middlewareHandlers, handler)
	this.middleware = makeMiddlewareFor(this.middlewareHandlers)
}

func (this *MiddlewareSupporter) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	this.Use(MiddlewareHandlerFunc(handlerFunc))
}

func (this *MiddlewareSupporter) UseHandler(handler http.Handler) {
	convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(res, req)
		//run the next automatically after this handler finished
		next(res, req)

	})

	this.Use(convertedMiddleware)
}

//
type MiddlewareHandler interface {
	ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)
}

type MiddlewareHandlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)

func (this MiddlewareHandlerFunc) ServeHTTP(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	this(res, req, next)
}

type Middleware struct {
	Handler MiddlewareHandler
	Next    *Middleware
}

func (this Middleware) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//println("what is nil [check this.Handler is nil]?", this.Handler == nil)
	///TODO: provlima edw einai nil to handler kai to next, kati pezei, dokimazw na valw pointer sto this *MiddlewareUser, nai telika auto eftege.
	this.Handler.ServeHTTP(res, req, this.Next.ServeHTTP)
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

func emptyMiddleware() Middleware {
	return Middleware{
		Handler: MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {}),
		Next:    &Middleware{},
	}
}
