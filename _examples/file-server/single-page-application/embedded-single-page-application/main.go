package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/...
// $ go-bindata -fs ./data/...
// $ go run .

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()

	// Using the iris.PrefixDir you can select
	// which directories to use under a particular file system,
	// e.g. for views the ./data/views and for static files
	// the ./data/public.
	templatesFS := iris.PrefixDir("./data/views", AssetFile())
	app.RegisterView(iris.HTML(templatesFS, ".html"))

	publicFS := iris.PrefixDir("./data/public", AssetFile())
	app.HandleDir("/", publicFS)

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
