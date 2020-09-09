package main

import "github.com/kataras/iris/v12"

func main() {
	e := iris.Amber(nil, ".amber") // You can still use a file system though.
	e.AddFunc("greet", func(name string) string {
		return "Hello, " + name + "!"
	})
	err := e.ParseTemplate("program.amber", []byte(`h1 #{ greet(Name) }`))
	if err != nil {
		panic(err)
	}
	e.Reload(true)

	app := iris.New()
	app.RegisterView(e)
	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.View("program.amber", iris.Map{
		"Name": "Gerasimos",
		// Or per template:
		// "greet": func(....)
	})
}
