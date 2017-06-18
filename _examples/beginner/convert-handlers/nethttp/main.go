package main

import (
	"net/http"

	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/handlerconv"
)

func main() {
	app := iris.New()
	irisMiddleware := handlerconv.FromStd(nativeTestMiddleware)
	app.Use(irisMiddleware)

	// Method GET: http://localhost:8080/
	app.Get("/", func(ctx context.Context) {
		ctx.HTML("Home")
	})

	// Method GET: http://localhost:8080/ok
	app.Get("/ok", func(ctx context.Context) {
		ctx.HTML("<b>Hello world!</b>")
	})

	// http://localhost:8080
	// http://localhost:8080/ok
	app.Run(iris.Addr(":8080"))
}

func nativeTestMiddleware(w http.ResponseWriter, r *http.Request) {
	println("Request path: " + r.URL.Path)
}
