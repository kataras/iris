package main

import (
	"github.com/kataras/iris/v12"
)

// $ go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// $ go-bindata -nomemcopy -fs ./public/...
// $ go run .

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

	// We didn't add a `-prefix "public"` argument on go-bindata command
	// because the view's `Assset` and `AssetNames` require fullpath.
	// Make use of the `PrefixDir` to serve assets on cases like that;
	// when bindata.go file contains files that are
	// not necessary public assets to be served.
	app.HandleDir("/", iris.PrefixDir("public", AssetFile()))

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/app.js
	// http://localhost:8080/css/main.css
	// http://localhost:8080/app2
	app.Listen(":8080")
}
