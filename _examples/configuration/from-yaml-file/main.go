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

	// Good when you have two configurations, one for development and a different one for production use.
	app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.YAML("./configs/iris.yml")))

	// or before run:
	// app.Configure(iris.WithConfiguration(iris.YAML("./configs/iris.yml")))
	// app.Run(iris.Addr(":8080"))
}
