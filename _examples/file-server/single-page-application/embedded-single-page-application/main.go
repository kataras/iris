package main

import (
	"github.com/kataras/iris/v12"
)

// $ go get -u github.com/shuLhan/go-bindata/...
// $ go-bindata ./public/...
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

	app.HandleDir("/", "./public", iris.DirOptions{
		Asset:      Asset,
		AssetInfo:  AssetInfo,
		AssetNames: AssetNames,
	})

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/app.js
	// http://localhost:8080/css/main.css
	// http://localhost:8080/app2
	app.Run(iris.Addr(":8080"))
}
