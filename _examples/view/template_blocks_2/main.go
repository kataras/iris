package main

import "github.com/kataras/iris/v12"

// Based on https://github.com/kataras/iris/issues/2214.
func main() {
	app := initApp()
	app.Listen(":8080")
}

func initApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	tmpl := iris.Blocks("./src/public/html", ".html")
	tmpl.Layout("main")
	app.RegisterView(tmpl)

	app.Get("/list", func(ctx iris.Context) {
		ctx.View("files/list")
	})

	app.Get("/menu", func(ctx iris.Context) {
		ctx.View("menu/menu")
	})

	app.Get("/list2", func(ctx iris.Context) {
		ctx.ViewLayout("secondary")
		ctx.View("files/list")
	})

	app.Get("/menu2", func(ctx iris.Context) {
		ctx.ViewLayout("secondary")
		ctx.View("menu/menu")
	})

	return app
}
