package router

import (
	"net/http"
)

//The router is handled on BOTTOM of the default mux.

//type Handler func(http.Handler) http.Handler
type Handler func(http.ResponseWriter, *http.Request)

type HttpRoute struct {
	Method  string
	Handler Handler
}

type HttpRouter struct {
	//Routes map[string]http.Handler
	Routes map[string]*HttpRoute
}

func NewHttpRouter() *HttpRouter {

	return &HttpRouter{Routes: make(map[string]*HttpRoute)}
}

func (this *HttpRouter) Unroute(urlPath string) *HttpRouter {
	delete(this.Routes, urlPath)
	return this
}

func (this *HttpRouter) Route(method string, urlPath string, handler Handler) *HttpRouter {
	if urlPath == "" {
		urlPath = "/"
	}
	if method == "" {
		method = HttpMethods.GET
	}
	this.Routes[urlPath] = &HttpRoute{Method: method, Handler: handler}
	return this
}
