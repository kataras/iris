package main

import (
	"github.com/kataras/iris"
)

func newApp() *iris.Application {
	app := iris.New()

	app.Favicon("./assets/favicon.ico")

	// first parameter is the request path
	// second is the system directory
	//
	// app.HandleDir("/css", "./assets/css")
	// app.HandleDir("/js", "./assets/js")

	app.HandleDir("/static", "./assets", iris.DirOptions{
		// Defaults to "/index.html", if request path is ending with **/*/$IndexName
		// then it redirects to **/*(/) which another handler is handling it,
		// that another handler, called index handler, is auto-registered by the framework
		// if end developer does not managed to handle it by hand.
		IndexName: "/index.html",
		// When files should served under compression.
		Gzip: false,
		// List the files inside the current requested directory if `IndexName` not found.
		ShowList: false,
		// If `ShowList` is true then this function will be used instead of the default one to show the list of files of a current requested directory(dir).
		// DirList: func(ctx iris.Context, dirName string, dir http.File) error { ... }
		//
		// Optional validator that loops through each requested resource.
		// AssetValidator:  func(ctx iris.Context, name string) bool { ... }
	})

	// You can also register any index handler manually, order of registration does not matter:
	// app.Get("/static", [...custom middleware...], func(ctx iris.Context) {
	//  [...custom code...]
	// 	ctx.ServeFile("./assets/index.html", false)
	// })

	// http://localhost:8080/static
	// http://localhost:8080/static/css/main.css
	// http://localhost:8080/static/js/jquery-2.1.1.js
	// http://localhost:8080/static/favicon.ico
	return app
}

func main() {
	app := newApp()
	app.Run(iris.Addr(":8080"))
}
