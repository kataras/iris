package main

import (
	"html/template"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	tmpl := iris.Pug("./templates", ".pug")
	tmpl.Reload(true)                                            // reload templates on each request (development mode)
	tmpl.AddFunc("bold", func(s string) (template.HTML, error) { // add your template func here.
		return template.HTML("<b>" + s + "</b>"), nil
	})

	app.RegisterView(tmpl)

	app.Get("/", index)

	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}

func index(ctx iris.Context) {
	ctx.View("index.pug")
}
