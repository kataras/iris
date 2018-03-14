package main

import (
	"github.com/kataras/iris"
)

// NOTE: need different tool than the "embedding-files-into-app" example.
//
// Follow these steps first:
// $ go get -u github.com/kataras/bindata/cmd/bindata
// $ bindata ./assets/...
// $ go build
// $ ./embedding-gziped-files-into-app
// "physical" files are not used, you can delete the "assets" folder and run the example.

func newApp() *iris.Application {
	app := iris.New()

	// Note the `GzipAsset` and `GzipAssetNames` are different from `go-bindata`'s `Asset` and `AssetNames,
	// that means that you can use both `go-bindata` and `bindata` tools,
	// the `go-bindata` can be used for the view engine's `Binary` method
	// and the `bindata` with the `StaticEmbeddedGzip` (x8 times faster than the StaticEmbeded with `go-bindata`).
	app.StaticEmbeddedGzip("/static", "./assets", GzipAsset, GzipAssetNames)

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080/static/css/bootstrap.min.css
	// http://localhost:8080/static/js/jquery-2.1.1.js
	// http://localhost:8080/static/favicon.ico
	app.Run(iris.Addr(":8080"))
}
