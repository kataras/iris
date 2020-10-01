package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.RegisterView(iris.Blocks("./views", ".html"))
	// Note, in Blocks engine, layouts
	// are used by their base names, the
	// blocks.LayoutDir(layoutDir) defaults to "./layouts".

	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title":      "Page Title",
		"FooterText": "Footer contents",
		"Message":    "Main contents",
	}

	ctx.ViewLayout("main")
	ctx.View("index", data)
}
