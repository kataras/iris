package main

import (
	"github.com/kataras/iris/v12"
)

// Follow the steps below:
// $ go get -u github.com/kataras/bindata/cmd/bindata
//
// $ bindata -prefix "../http2push/" ../http2push/assets/...
// # OR if the ./assets directory was inside this example foder:
// # bindata ./assets/...
//
// $ go run .
// Physical files are not used, you can delete the "assets" folder and run the example.

var opts = iris.DirOptions{
	IndexName: "/index.html",
	PushTargets: map[string][]string{
		"/": { // Relative path without route prefix.
			"favicon.ico",
			"js/main.js",
			"css/main.css",
			// ^ Relative to the index, if need absolute ones start with a slash ('/').
		},
	},
	Compress:   false, // SHOULD be set to false, files already compressed.
	ShowList:   true,
	Asset:      GzipAsset,
	AssetInfo:  GzipAssetInfo,
	AssetNames: GzipAssetNames,
	// Required for pre-compressed files:
	AssetValidator: func(ctx iris.Context, _ string) bool {
		// ctx.Header("Vary", "Content-Encoding")
		ctx.Header("Content-Encoding", "gzip")
		return true
	},
}

func main() {
	app := iris.New()
	app.HandleDir("/public", "./assets", opts)

	app.Run(iris.TLS(":443", "../http2push/mycert.crt", "../http2push/mykey.key"))
}
