package main

import (
	"github.com/kataras/iris"
)

// same as embedded-single-page-application but without go-bindata, the files are "physical" stored in the
// current system directory.

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()
	app.RegisterView(iris.HTML("./public", ".html"))

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("Page", page)
		ctx.View("index.html")
	})

	// or just serve index.html as it is:
	// app.Get("/", func(ctx iris.Context) {
	// 	ctx.ServeFile("index.html", false)
	// })

	assetHandler := app.StaticHandler("./public", false, false)
	// as an alternative of SPA you can take a look at the /routing/dynamic-path/root-wildcard
	// example too
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
