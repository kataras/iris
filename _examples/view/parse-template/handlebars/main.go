package main

import "github.com/kataras/iris/v12"

func main() {
	e := iris.Handlebars(nil, ".html") // You can still use a file system though.
	e.ParseTemplate("program.html", `<h1>{{greet Name}}</h1>`, iris.Map{
		"greet": func(name string) string {
			return "Hello, " + name + "!"
		},
	})
	e.Reload(true)

	app := iris.New()
	app.RegisterView(e)
	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.View("program.html", iris.Map{
		"Name": "Gerasimos",
	})
}
