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

/* MIDDLEWARE(S) */

func (this *Gapi) Use(_middlewares ...server.Middleware) *Gapi {
	this.server.Use(_middlewares...)
	return this
}

/* ROUTER */

func (this *Gapi) Get(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.GET, path, handler)
	return this
}

func (this *Gapi) Post(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.POST, path, handler)
	return this
}

func (this *Gapi) Put(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.PUT, path, handler)
	return this
}

func (this *Gapi) Delete(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.DELETE, path, handler)
	return this
}

func (this *Gapi) Connect(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.CONNECT, path, handler)
	return this
}

func (this *Gapi) Head(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.HEAD, path, handler)
	return this
}

func (this *Gapi) Options(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.OPTIONS, path, handler)
	return this
}

func (this *Gapi) Patch(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.PATCH, path, handler)
	return this
}

func (this *Gapi) Trace(path string, handler router.Handler) *Gapi {
	this.server.Router.Route(router.HttpMethods.TRACE, path, handler)
	return this
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
