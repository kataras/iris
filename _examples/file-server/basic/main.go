package main

import (
	"github.com/kataras/iris/v12"
)

func newApp() *iris.Application {
	app := iris.New()

	app.Favicon("./assets/favicon.ico")

	// first parameter is the request path
	// second is the system directory
	//
	// app.HandleDir("/css", iris.Dir("./assets/css"))
	// app.HandleDir("/js",  iris.Dir("./assets/js"))

	v1 := app.Party("/v1")
	v1.HandleDir("/static", iris.Dir("./assets"), iris.DirOptions{
		// Defaults to "/index.html", if request path is ending with **/*/$IndexName
		// then it redirects to **/*(/) which another handler is handling it,
		// that another handler, called index handler, is auto-registered by the framework
		// if end developer does not managed to handle it by hand.
		IndexName: "/index.html",
		// When files should served under compression.
		Compress: false,
		// List the files inside the current requested directory if `IndexName` not found.
		ShowList: false,
		// When ShowList is true you can configure if you want to show or hide hidden files.
		ShowHidden: false,
		Cache: iris.DirCacheOptions{
			// enable in-memory cache and pre-compress the files.
			Enable: true,
			// ignore image types (and pdf).
			CompressIgnore: iris.MatchImagesAssets,
			// do not compress files smaller than size.
			CompressMinSize: 300,
			// available encodings that will be negotiated with client's needs.
			Encodings: []string{"gzip", "br" /* you can also add: deflate, snappy */},
		},
		DirList: iris.DirListRich(),
		// If `ShowList` is true then this function will be used instead of the default
		// one to show the list of files of a current requested directory(dir).
		// DirList: func(ctx iris.Context, dirName string, dir http.File) error { ... }
		//
		// Optional validator that loops through each requested resource.
		// AssetValidator:  func(ctx iris.Context, name string) bool { ... }
	})

	// You can also register any index handler manually, order of registration does not matter:
	// v1.Get("/static", [...custom middleware...], func(ctx iris.Context) {
	//  [...custom code...]
	// 	ctx.ServeFile("./assets/index.html")
	// })

	// http://localhost:8080/v1/static
	// http://localhost:8080/v1/static/css/main.css
	// http://localhost:8080/v1/static/js/jquery-2.1.1.js
	// http://localhost:8080/v1/static/favicon.ico
	return app
}

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	app.Listen(":8080")
}
