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

	// Good when you share configuration between multiple iris instances.
	// This configuration file lives in your $HOME/iris.yml for unix hosts
	// or %HOMEDRIVE%+%HOMEPATH%/iris.yml for windows hosts, and you can modify it.
	app.Run(iris.Addr(":8080"), iris.WithGlobalConfiguration)
	// or before run:
	// app.Configure(iris.WithGlobalConfiguration)
	// app.Run(iris.Addr(":8080"))
}
