package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<b>Hello!</b>")
	})
	// [...]

	// Good when you want to change some of the configuration's field.
	// Prefix: "With", code editors will help you navigate through all
	// configuration options without even a glitch to the documentation.

	app.Run(iris.Addr(":8080"), iris.WithoutStartupLog, iris.WithCharset("UTF-8"))

	// or before run:
	// app.Configure(iris.WithoutStartupLog, iris.WithCharset("UTF-8"))
	// app.Run(iris.Addr(":8080"))
}
