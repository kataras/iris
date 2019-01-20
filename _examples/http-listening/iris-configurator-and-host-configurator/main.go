package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.ConfigureHost(func(host *iris.Supervisor) { // <- HERE: IMPORTANT
		// You can control the flow or defer something using some of the host's methods:
		// host.RegisterOnError
		// host.RegisterOnServe
		host.RegisterOnShutdown(func() {
			app.Logger().Infof("Application shutdown on signal")
		})
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hello</h1>\n")
	})

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))

	/* There are more easy ways to notify for global shutdown using the `iris.RegisterOnInterrupt` for default signal interrupt events.
	You can even go it even further by looking at the: "graceful-shutdown" example.
	*/
}
