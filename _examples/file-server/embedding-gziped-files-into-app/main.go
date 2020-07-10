package main

import (
	"github.com/kataras/iris/v12"
)

// NOTE: need different tool than the "embedding-files-into-app" example.
//
// Follow these steps first:
// $ go get -u github.com/kataras/bindata/cmd/bindata
// $ bindata ./assets/...
// $ go run .
// $ ./embedding-gziped-files-into-app
// "physical" files are not used, you can delete the "assets" folder and run the example.

func newApp() *iris.Application {
	app := iris.New()

	// Note the `GzipAsset` and `GzipAssetNames` are different from go-bindata's `Asset`,
	// do not set the `Compress` option to true, instead
	// use the `AssetValidator` option to manually set the content-encoding to "gzip".
	app.HandleDir("/static", "./assets", iris.DirOptions{
		Asset:      GzipAsset,
		AssetInfo:  GzipAssetInfo,
		AssetNames: GzipAssetNames,
		AssetValidator: func(ctx iris.Context, name string) bool {
			ctx.Header("Vary", "Accept-Encoding")
			ctx.Header("Content-Encoding", "gzip")
			return true
		},
	})
	return app
}

func main() {
	app := newApp()

	// http://localhost:8080/static/css/bootstrap.min.css
	// http://localhost:8080/static/js/jquery-2.1.1.js
	// http://localhost:8080/static/favicon.ico
	app.Listen(":8080")
}
