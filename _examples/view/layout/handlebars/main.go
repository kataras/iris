package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	app.RegisterView(iris.Handlebars("./views", ".html"))

	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title":      "Page Title",
		"FooterText": "Footer contents",
		"Message":    "Main contents",
	}

	ctx.ViewLayout("layouts/main")
	ctx.View("index", data)
}
