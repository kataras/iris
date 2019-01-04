package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()

	tmpl := iris.Pug("./templates", ".pug")
	tmpl.Reload(true)                             // reload templates on each request (development mode)
	tmpl.AddFunc("greet", func(s string) string { // add your template func here.
		return "Greetings " + s + "!"
	})

	app.RegisterView(tmpl)

	app.Get("/", index)

	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}

func index(ctx iris.Context) {
	ctx.ViewData("pageTitle", "My Index Page")
	ctx.ViewData("youAreUsingJade", true)
	// Q: why need extension .pug?
	// A: Because you can register more than one view engine per Iris application.
	ctx.View("index.pug")

}
