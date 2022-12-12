package main

import (
	"github.com/kataras/iris/v12"
)

// $ go install github.com/go-bindata/go-bindata/v3/go-bindata@latest
// $ go-bindata -prefix "data" -fs ./data/...
// $ go run .

var page = struct {
	Title string
}{"Welcome"}

func newApp() *iris.Application {
	app := iris.New()

	app.RegisterView(iris.HTML(AssetFile(), ".html").RootDir("views"))

	// Using the iris.PrefixDir you can select
	// which directories to use under a particular file system,
	// e.g. for views the ./public:
	// publicFS := iris.PrefixDir("./public", AssetFile())
	publicFS := iris.PrefixDir("./public", AssetFile())
	app.HandleDir("/", publicFS)

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("Page", page)
		if err := ctx.View("index.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
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
