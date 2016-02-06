package gapi

//This file just exposes the server and it's router & middlewares
import (
	"net/http"

	"github.com/kataras/gapi/router"
	"github.com/kataras/gapi/server"
)

var (
	HttpMethods = router.HttpMethods
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

func (this *Gapi) Use(handler router.MiddlewareHandler) *Gapi {
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
func (this *Gapi) Route(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler)
}

func (this *Gapi) Get(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.GET)
}

func (this *Gapi) Post(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.POST)
}

func (this *Gapi) Put(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.PUT)
}

func (this *Gapi) Delete(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.DELETE)
}

func (this *Gapi) Connect(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.CONNECT)
}

func (this *Gapi) Head(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.HEAD)
}

func (this *Gapi) Options(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.OPTIONS)
}

func (this *Gapi) Patch(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.PATCH)
}

func (this *Gapi) Trace(path string, handler router.Handler) *router.HttpRoute {
	return this.server.Router.Route(path, handler, router.HttpMethods.TRACE)
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
