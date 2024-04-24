package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	/*
	 * Setup static files
	 */

	app.HandleDir("/assets", iris.Dir("./public/assets"))
	app.HandleDir("/upload_resources", iris.Dir("./public/upload_resources"))

	dashboard := app.Party("dashboard.")
	{
		dashboard.Get("/", func(ctx iris.Context) {
			ctx.Writef("HEY FROM dashboard")
		})
	}
	system := app.Party("system.")
	{
		system.Get("/", func(ctx iris.Context) {
			ctx.Writef("HEY FROM system")
		})
	}

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("HEY FROM frontend /")
	})
	// http://domain.local:80
	// http://dashboard.local
	// http://system.local
	// Make sure you prepend the "http" in your browser
	// because .local is a virtual domain we think to show case you
	// that you can declare any syntactical correct name as a subdomain in iris.
	app.Listen("domain.local:80") // for beginners: look ../hosts file
}
