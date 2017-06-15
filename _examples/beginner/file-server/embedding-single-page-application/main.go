package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/view"
)

// $ go get -u github.com/jteeuwen/go-bindata/...
// $ go-bindata ./public/...
// $ go build
// $ ./embedding-single-page-application

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()
	app.AttachView(view.HTML("./public", ".html").Binary(Asset, AssetNames))

	app.Get("/", func(ctx context.Context) {
		ctx.ViewData("Page", page)
		ctx.View("index.html")
	})

	assetHandler := app.StaticEmbeddedHandler("./public", Asset, AssetNames)
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
