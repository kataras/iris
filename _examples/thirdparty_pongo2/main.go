package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware"
)

func main() {
	iris.Use(middleware.Pongo2())

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Set("template", "./_examples/thirdparty_pongo2/index.html")
		ctx.Set("data", map[string]interface{}{"is_admin": true})
	})

	iris.Listen(":8080")
}
