package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(httprouter.New())
	// This will serve the ./static/favicons/iris_favicon_32_32.ico to: localhost:8080/favicon.ico
	app.Favicon("./static/favicons/iris_favicon_32_32.ico")

	// app.Favicon("./static/favicons/iris_favicon_32_32.ico", "/favicon_32_32.ico")
	// This will serve the ./static/favicons/iris_favicon_32_32.ico to: localhost:8080/favicon_32_32.ico

	app.Get("/", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, `You should see the favicon now at the side of your browser,
      if not, please refresh or clear the browser's cache.`)
	})

	app.Listen(":8080")
}
