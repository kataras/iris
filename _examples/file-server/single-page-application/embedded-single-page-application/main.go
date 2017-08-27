package main

import (
	"github.com/kataras/iris"
)

// $ go get -u github.com/jteeuwen/go-bindata/...
// $ go-bindata ./public/...
// $ go build
// $ ./embedded-single-page-application

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()
	app.RegisterView(iris.HTML("./public", ".html").Binary(Asset, AssetNames))

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("Page", page)
		ctx.View("index.html")
	})

	assetHandler := app.StaticEmbeddedHandler("./public", Asset, AssetNames)
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

// Note that app.Use/UseGlobal/Done will be executed
// only to the registered routes like our index (app.Get("/", ..)).
// The file server is clean, but you can still add middleware to that by wrapping its "assetHandler".
//
// With this method, unlike StaticWeb("/" , "./public") which is not working by-design anymore,
// all custom http errors and all routes are working fine with a file server that is registered
// to the root path of the server.
