package main

import (
	"github.com/kataras/iris/v12"
)

// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
// $ go-bindata -fs -prefix "public" ./public/...
// $ go run .

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()

	app.RegisterView(iris.HTML(AssetFile(), ".html"))

	app.HandleDir("/", AssetFile())

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("Page", page)
		ctx.View("index.html")
	})

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
