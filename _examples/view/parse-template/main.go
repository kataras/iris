// Package main shows how to parse a template through custom byte slice content.
// The following works with HTML, Pug and Ace template parsers.
// To learn how you can manually parse a template from a text for the rest
// template parsers navigate through the example's subdirectories.
package main

import "github.com/kataras/iris/v12"

func main() {
	// To not load any templates from files or embedded data,
	// pass nil or empty string on the first argument:
	// e := iris.HTML(nil, ".html")

	e := iris.HTML("./views", ".html")
	// e := iris.Pug("./views",".pug")
	// e := iris.Ace("./views",".ace")
	e.ParseTemplate("program.html", []byte(`<h1>{{greet .Name}}</h1>`), iris.Map{
		"greet": func(name string) string {
			return "Hello, " + name + "!"
		},
	})
	e.Reload(true)

	app := iris.New()
	app.RegisterView(e)

	app.Get("/", index)
	app.Get("/layout", layout)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	if err := ctx.View("program.html", iris.Map{
		"Name": "Gerasimos",
	}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func layout(ctx iris.Context) {
	ctx.ViewLayout("layouts/main.html")
	index(ctx)
}
