// +build go1.9

package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.Default()

	// Method:   GET
	// Resource: http://localhost:8080/
	app.Handle("GET", "/", func(ctx iris.Context) {
		ctx.HTML("<b>Hello world!</b>")
	})

	// same as app.Handle("GET", "/ping", [...])
	// Method:   GET
	// Resource: http://localhost:8080/ping
	app.Get("/ping", func(ctx iris.Context) {
		ctx.WriteString("pong")
	})

	// Method:   GET
	// Resource: http://localhost:8080/hello
	app.Get("/hello", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"message": "Hello iris web framework."})
	})

	// http://localhost:8080
	// http://localhost:8080/ping
	// http://localhost:8080/hello
	app.Run(iris.Addr(":8080"))
}
