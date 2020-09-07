// Package main shows how to parse a template through custom byte slice content.
package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	// To not load any templates from files or embedded data,
	// pass nil or empty string on the first argument:
	// view := iris.HTML(nil, ".html")

	view := iris.HTML("./views", ".html")
	view.ParseTemplate("program.html", []byte(`<h1>{{greet .Name}}</h1>`), iris.Map{
		"greet": func(name string) string {
			return "Hello, " + name + "!"
		},
	})

	app.RegisterView(view)
	app.Get("/", index)
	app.Get("/layout", layout)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.View("program.html", iris.Map{
		"Name": "Gerasimos",
	})
}

func layout(ctx iris.Context) {
	ctx.ViewLayout("layouts/main.html")
	index(ctx)
}
