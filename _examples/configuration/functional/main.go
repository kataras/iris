package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<b>Hello!</b>")
	})
	// [...]

	// Good when you want to change some of the configuration's field.
	// Prefix: "With", code editors will help you navigate through all
	// configuration options without even a glitch to the documentation.

	app.Listen(":8080", iris.WithoutStartupLog, iris.WithCharset("utf-8"))

	// or before run:
	// app.Configure(iris.WithoutStartupLog, iris.WithCharset("utf-8"))
	// app.Listen(":8080")
}
