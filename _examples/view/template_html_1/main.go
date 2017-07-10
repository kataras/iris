package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type mypage struct {
	Title   string
	Message string
}

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./templates", ".html").Layout("layout.html"))
	// TIP: append .Reload(true) to reload the templates on each request.

	app.Get("/", func(ctx context.Context) {
		ctx.Gzip(true)
		ctx.ViewData("", mypage{"My Page title", "Hello world!"})
		ctx.View("mypage.html")
		// Note that: you can pass "layout" : "otherLayout.html" to bypass the config's Layout property
		// or view.NoLayout to disable layout on this render action.
		// third is an optional parameter
	})

	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}
