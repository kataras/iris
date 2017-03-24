package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	// subdomains works with all available routers, like other features too.
	app.Adapt(httprouter.New())

	/*
	 * Setup static files
	 */

	app.StaticWeb("/assets", "./public/assets")
	app.StaticWeb("/upload_resources", "./public/upload_resources")

	dashboard := app.Party("dashboard.")
	{
		dashboard.Get("/", func(ctx *iris.Context) {
			ctx.Writef("HEY FROM dashboard")
		})
	}
	system := app.Party("system.")
	{
		system.Get("/", func(ctx *iris.Context) {
			ctx.Writef("HEY FROM system")
		})
	}

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("HEY FROM frontend /")
	})
	/* test this on firefox, because the domain is not real (because of .local), on firefox this will fail, but you can test it with other domain */
	app.Listen("domain.local:80") // for beginners: look ../hosts file
}
