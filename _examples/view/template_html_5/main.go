package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./views", ".html").Layout("layout.html"))
	// TIP: append .Reload(true) to reload the templates on each request.

	app.Get("/home", func(ctx iris.Context) {
		ctx.ViewData("title", "Home page");
		ctx.View("home.html")
		// Note that: you can pass "layout" : "otherLayout.html" to bypass the config's Layout property
		// or view.NoLayout to disable layout on this render action.
		// third is an optional parameter
	})

	app.Get("/about", func(ctx iris.Context) {
		ctx.View("about.html")
	})

	app.Get("/user/index", func(ctx iris.Context) {
		ctx.View("user/index.html")
	})

	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}
