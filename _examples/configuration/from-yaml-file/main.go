package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<b>Hello!</b>")
	})
	// [...]

	// Good when you have two configurations, one for development and a different one for production use.
	// If iris.YAML's input string argument is "~" then it loads the configuration from the home directory
	// and can be shared between many iris instances.
	app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.YAML("./configs/iris.yml")))

	// or before run:
	// app.Configure(iris.WithConfiguration(iris.YAML("./configs/iris.yml")))
	// app.Run(iris.Addr(":8080"))
}
