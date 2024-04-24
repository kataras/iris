package main

import (
	"embed"

	"github.com/kataras/iris/v12"
)

//go:embed assets/*
var fs embed.FS

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.HandleDir("/static", fs)

	/*
		Or if you need to cache them inside the memory (requires the assets folder
		to be located near the executable program):
		app.HandleDir("/static", iris.Dir("./assets"), iris.DirOptions{
			IndexName: "index.html",
			Cache: iris.DirCacheOptions{
				Enable:          true,
				Encodings:       []string{"gzip"},
				CompressIgnore:  iris.MatchImagesAssets,
				CompressMinSize: 30 * iris.B,
			},
		})
	*/
	return app
}

func main() {
	app := newApp()

	// http://localhost:8080/static/css/main.css
	// http://localhost:8080/static/js/main.js
	// http://localhost:8080/static/favicon.ico
	app.Listen(":8080")
}
