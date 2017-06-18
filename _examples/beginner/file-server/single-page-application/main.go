package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/view"
)

// same as embedding-single-page-application but without go-bindata, the files are "physical" stored in the
// current system directory.

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()
	app.AttachView(view.HTML("./public", ".html"))

	app.Get("/", func(ctx context.Context) {
		ctx.ViewData("Page", page)
		ctx.View("index.html")
	})

	// or just serve index.html as it is:
	// app.Get("/", func(ctx context.Context) {
	// 	ctx.ServeFile("index.html", false)
	// })

	assetHandler := app.StaticHandler("./public", false, false)
	app.SPA(assetHandler)

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/index.html
	// http://localhost:8080/app.js
	// http://localhost:8080/css/main.css
	app.Run(iris.Addr(":8080"))
}
