package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware"
)

//This example doesn't contain any real pongo2 templates, just the basic code  for the middleware

func main() {
	iris.Use(middleware.Pongo2())

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Set("template", "index.html")
		ctx.Set("data", map[string]interface{}{"message": "Hello World!"})
	})

	iris.Listen(":8080")
}
