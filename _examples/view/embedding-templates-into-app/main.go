package main

import (
	"embed"

	"github.com/kataras/iris/v12"
)

//go:embed embedded/*
var embeddedFS embed.FS

func main() {
	app := iris.New()

	tmpl := iris.HTML(embeddedFS, ".html").RootDir("embedded/templates")

	tmpl.Layout("layouts/layout.html")
	tmpl.AddFunc("greet", func(s string) string {
		return "Greetings " + s + "!"
	})

	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		if err := ctx.View("page1.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	// remove the layout for a specific route
	app.Get("/nolayout", func(ctx iris.Context) {
		ctx.ViewLayout(iris.NoLayout)
		if err := ctx.View("page1.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	// set a layout for a party, .Layout should be BEFORE any Get or other Handle party's method
	my := app.Party("/my").Layout("layouts/mylayout.html")
	{ // both of these will use the layouts/mylayout.html as their layout.
		my.Get("/", func(ctx iris.Context) {
			if err := ctx.View("page1.html"); err != nil {
				ctx.HTML("<h3>%s</h3>", err.Error())
				return
			}
		})
		my.Get("/other", func(ctx iris.Context) {
			if err := ctx.View("page1.html"); err != nil {
				ctx.HTML("<h3>%s</h3>", err.Error())
				return
			}
		})
	}

	// http://localhost:8080
	// http://localhost:8080/nolayout
	// http://localhost:8080/my
	// http://localhost:8080/my/other
	app.Listen(":8080")
}
