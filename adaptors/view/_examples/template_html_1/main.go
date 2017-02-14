package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

type mypage struct {
	Title   string
	Message string
}

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())

	tmpl := view.HTML("./templates", ".html")
	tmpl.Layout("layout.html")

	app.Adapt(tmpl)

	app.Get("/", func(ctx *iris.Context) {
		ctx.Render("mypage.html", mypage{"My Page title", "Hello world!"}, iris.Map{"gzip": true})
		// Note that: you can pass "layout" : "otherLayout.html" to bypass the config's Layout property
		// or iris.NoLayout to disable layout on this render action.
		// third is an optional parameter
	})

	app.Listen(":8080")
}
