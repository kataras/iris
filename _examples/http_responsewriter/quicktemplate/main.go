package main

import (
	"github.com/kataras/iris/_examples/http_responsewriter/quicktemplate/controllers"

	"github.com/kataras/iris"
)

func newApp() *iris.Application {
	app := iris.New()
	app.Get("/", controllers.Index)
	app.Get("/{name}", controllers.Hello)

	return app
}

func main() {
	app := newApp()
	// http://localhost:8080
	// http://localhost:8080/yourname
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
