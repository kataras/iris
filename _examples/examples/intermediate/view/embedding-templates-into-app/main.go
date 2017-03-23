package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())

	// $ go-bindata ./templates/...
	// $ go build
	// $ ./template_binary
	// templates are not used, you can delete the folder and run the example
	app.Adapt(view.HTML("./templates", ".html").Binary(Asset, AssetNames))
	app.Get("/hi", hi)
	app.Listen(":8080")
}

func hi(ctx *iris.Context) {
	ctx.MustRender("hi.html", struct{ Name string }{Name: "iris"})
}
