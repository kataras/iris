package main

import (
	"net/http"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	irisMiddleware := iris.FromStd(nativeTestMiddleware)
	app.Use(irisMiddleware)

	// Method GET: http://localhost:8080/
	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("Home")
	})

	// Method GET: http://localhost:8080/ok
	app.Get("/ok", func(ctx iris.Context) {
		ctx.HTML("<b>Hello world!</b>")
	})

	// http://localhost:8080
	// http://localhost:8080/ok
	app.Listen(":8080")
}

func nativeTestMiddleware(w http.ResponseWriter, r *http.Request) {
	println("Request path: " + r.URL.Path)
}
