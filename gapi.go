package gapi

//This file just exposes the server and it's router & middlewares
import (
	"net/http"

	"github.com/kataras/gapi/router"
	"github.com/kataras/gapi/server"
)

func NewRouter() *router.HttpRouter {
	return router.NewHttpRouter()
}

func NewServer() *server.HttpServer {
	return server.NewHttpServer()
}

type Gapi struct {
	server *server.HttpServer
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

func (this *Gapi) Listen(fullHostOrPort interface{}) *server.HttpServer {
	this.server.Listen(fullHostOrPort)
	return this.server
}

/* GLOBAL MIDDLEWARE(S) */

//func (this *Gapi) Use(_middlewares ...server.Middleware) *Gapi {
//	this.server.Use(_middlewares...)
//	return this
//}

func (this *Gapi) Use(handler router.MiddlewareHandler) *Gapi {

	return this
}

func (this *Gapi) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Gapi {

	return this
}

func (this *Gapi) UseHandler(handler http.Handler) *Gapi {

	return this
}

/* ROUTER */

func (this *Gapi) Get(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.GET, path, handler)
}

func (this *Gapi) Post(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.POST, path, handler)
}

func (this *Gapi) Put(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.PUT, path, handler)
}

func (this *Gapi) Delete(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.DELETE, path, handler)
}

func (this *Gapi) Connect(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.CONNECT, path, handler)
}

func (this *Gapi) Head(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.HEAD, path, handler)
}

func (this *Gapi) Options(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.OPTIONS, path, handler)
}

func (this *Gapi) Patch(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.PATCH, path, handler)
}

func (this *Gapi) Trace(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(router.HttpMethods.TRACE, path, handler)
}

/* Router's params */

func (this *Gapi) Params(req *http.Request) router.Parameters {
	return router.GetParameters(req)
}

func (this *Gapi) Param(req *http.Request, key string) string {
	params := this.Params(req)
	param := ""
	if params != nil {
		param = params[key]
	}
	return param
}
