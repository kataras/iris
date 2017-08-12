package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.Favicon("./assets/favicon.ico")

	// enable gzip, optionally:
	// if used before the `StaticXXX` handlers then
	// the content byte range feature is gone.
	// recommend: turn off for large files especially
	// when server has low memory,
	// turn on for medium-sized files
	// or for large-sized files if they are zipped already,
	// i.e "zippedDir/file.gz"
	//
	// app.Use(iris.Gzip)

	// first parameter is the request path
	// second is the system directory
	//
	// app.StaticWeb("/css", "./assets/css")
	// app.StaticWeb("/js", "./assets/js")
	//
	app.StaticWeb("/static", "./assets")

	// http://localhost:8080/static/css/main.css
	// http://localhost:8080/static/js/jquery-2.1.1.js
	// http://localhost:8080/static/favicon.ico
	app.Run(iris.Addr(":8080"))

	// Note:
	// Routing doesn't allows something .StaticWeb("/", "./assets")
	//
	// To see how you can wrap the router in order to achieve
	// wildcard on root path, see "single-page-application".
}
