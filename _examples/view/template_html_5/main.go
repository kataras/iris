package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./views", ".html").Layout("layout.html"))
	// TIP: append .Reload(true) to reload the templates on each request.

	app.Get("/home", func(ctx iris.Context) {
		ctx.ViewData("title", "Home page")
		if err := ctx.View("home.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}

		// Note that: you can pass "layout" : "otherLayout.html" to bypass the config's Layout property
		// or view.NoLayout to disable layout on this render action.
		// third is an optional parameter
	})

	app.Get("/about", func(ctx iris.Context) {
		if err := ctx.View("about.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	app.Get("/user/index", func(ctx iris.Context) {
		if err := ctx.View("user/index.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	// http://localhost:8080
	app.Listen(":8080")
}
