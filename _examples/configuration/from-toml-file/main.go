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

	// Good when you have two configurations, one for development and a different one for production use.
	app.Listen(":8080", iris.WithConfiguration(iris.TOML("./configs/iris.tml")))

	// or before run:
	// app.Configure(iris.WithConfiguration(iris.TOML("./configs/iris.tml")))
	// app.Listen(":8080")
}
