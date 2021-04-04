package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Get("/", index)
	app.Get("/about", about).SetName("about_page")
	app.RemoveRoute("about_page")

	// http://localhost:8080
	// http://localhost:8080/about (Not Found)
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Hello, Gophers!")
}

func about(ctx iris.Context) {
	ctx.HTML("<h1>About Page</h1>")
}
