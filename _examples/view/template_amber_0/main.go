package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	// Read about its markup syntax at: https://github.com/eknkc/amber
	tmpl := iris.Amber("./views", ".amber")

	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		ctx.View("index.amber", iris.Map{
			"Title": "Title of The Page",
		})
	})

	app.Listen(":8080")
}
