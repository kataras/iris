package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	// Read about its markup syntax at: https://github.com/yosssi/ace
	tmpl := iris.Ace("./views", ".ace")
	// tmpl.Layout("layouts/main.ace") -> global layout for all pages.

	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		ctx.View("index", iris.Map{
			"Title": "Title of The Page",
		})
	})

	app.Get("/layout", func(ctx iris.Context) {
		ctx.ViewLayout("layouts/main") // layout for that response.
		ctx.View("index", iris.Map{    // file extension is optional.
			"Title": "Title of the main Page",
		})
	})

	// otherGroup := app.Party("/other").Layout("layouts/other.ace") -> layout for that party.
	// otherGroup.Get("/", func(ctx iris.Context) { ctx.View("index.ace", [...]) })

	app.Listen(":8080")
}
