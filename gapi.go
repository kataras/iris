package gapi

//This file just exposes the server and it's router & middlewares
import (
	"net/http"
)

func NewRouter() *HTTPRouter {
	return NewHTTPRouter()
}

func NewServer() *HTTPServer {
	return NewHTTPServer()
}

type Gapi struct {
	server *HTTPServer
}

func New() *Gapi {
	theServer := NewServer()
	theServer.SetRouter(NewRouter())
	return &Gapi{server: theServer}
}

/* ServeHTTP, use as middleware only in already http server. */
func (this *Gapi) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	this.server.ServeHTTP(res, req)
}

/* STANDALONE SERVER */

func (this *Gapi) Listen(fullHostOrPort interface{}) *HTTPServer {
	this.server.Listen(fullHostOrPort)
	return this.server
}

/* GLOBAL MIDDLEWARE(S) */

func (this *Gapi) Use(handler MiddlewareHandler) *Gapi {
	this.server.Router.Use(handler)
	return this
}

func (this *Gapi) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Gapi {
	this.server.Router.UseFunc(handlerFunc)
	return this
}

func (this *Gapi) UseHandler(handler http.Handler) *Gapi {
	this.server.Router.UseHandler(handler)
	return this
}

/* ROUTER */
func (this *Gapi) Route(path string, handler HTTPHandler) *HTTPRoute {
	
	return this.server.Router.Route(path, handler)
}

func (this *Gapi) Get(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.GET)
}

func (this *Gapi) Post(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.POST)
}

func (this *Gapi) Put(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.PUT)
}

func (this *Gapi) Delete(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.DELETE)
}

func (this *Gapi) Connect(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.CONNECT)
}

func (this *Gapi) Head(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.HEAD)
}

func (this *Gapi) Options(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.OPTIONS)
}

func (this *Gapi) Patch(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.PATCH)
}

func (this *Gapi) Trace(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.TRACE)
}