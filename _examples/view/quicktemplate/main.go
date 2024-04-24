package main

import (
	"github.com/kataras/iris/v12/_examples/view/quicktemplate/controllers"

	"github.com/kataras/iris/v12"
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
	app.Listen(":8080")
}
