package main

import (
	"html/template"

	"github.com/kataras/iris/v12"
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
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	if err := ctx.View("index.pug"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
