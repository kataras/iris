package main

import (
	"github.com/kataras/iris/v12"
)

type mypage struct {
	Title   string
	Message string
}

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./templates", ".html").Layout("layout.html"))
	// TIP: append .Reload(true) to reload the templates on each request.

	app.Get("/", func(ctx iris.Context) {
		ctx.CompressWriter(true)
		ctx.ViewData("", mypage{"My Page title", "Hello world!"})
		if err := ctx.View("mypage.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
		// Note that: you can pass "layout" : "otherLayout.html" to bypass the config's Layout property
		// or view.NoLayout to disable layout on this render action.
		// third is an optional parameter
	})

	// http://localhost:8080
	app.Listen(":8080")
}
