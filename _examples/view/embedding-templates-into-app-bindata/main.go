package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	// $ go install github.com/go-bindata/go-bindata/v3/go-bindata@latest
	// $ go-bindata -fs -prefix "templates" ./templates/...
	// $ go run .
	// html files are not used, you can delete the folder and run the example.
	tmpl := iris.HTML(AssetFile(), ".html")
	tmpl.Layout("layouts/layout.html")
	tmpl.AddFunc("greet", func(s string) string {
		return "Greetings " + s + "!"
	})

	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		if err := ctx.View("page1.html"); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef(err.Error())
		}
	})

	// remove the layout for a specific route
	app.Get("/nolayout", func(ctx iris.Context) {
		ctx.ViewLayout(iris.NoLayout)
		if err := ctx.View("page1.html"); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef(err.Error())
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
