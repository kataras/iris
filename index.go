package gapi

import (
	"github.com/kataras/gapi/router"
	"github.com/kataras/gapi/server"
)

func NewRouter() *router.HttpRouterBuilder {
	return router.NewHttpRouterBuilder(nil)
}

func NewServer() *server.HttpServer {
	return server.NewHttpServer()
}

func New() (theServer *server.HttpServer, theRouter *router.HttpRouterBuilder) {
	theServer = NewServer()
	theRouter = NewRouter()
	theServer.Router(theRouter)
	return theServer, theRouter
}
