package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	// This will serve the ./static/favicons/iris_favicon_48_48.ico to: localhost:8080/favicon.ico
	app.Favicon("./static/favicons/iris_favicon_48_48.ico")

	// app.Favicon("./static/favicons/iris_favicon_48_48.ico", "/favicon_48_48.ico")
	// This will serve the ./static/favicons/iris_favicon_48_48.ico to: localhost:8080/favicon_48_48.ico

	app.Get("/", func(ctx context.Context) {
		ctx.HTML(`<a href="/favicon.ico"> press here to see the favicon.ico</a>.
		 At some browsers like chrome, it should be visible at the top-left side of the browser's window,
		 because some browsers make requests to the /favicon.ico automatically,
		  so Iris serves your favicon in that path too (you can change it).`)
	}) // if favicon doesn't show to you, try to clear your browser's cache.

	app.Run(iris.Addr(":8080"))
}
