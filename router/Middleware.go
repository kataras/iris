package router

import (
	"net/http"
)

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
		Next: &Middleware{},
	}
}
