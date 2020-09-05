package main

import (
	"github.com/kataras/iris/v12"
)

// How to run:
//
// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
// $ go-bindata -prefix "../embedding-files-into-app/assets/" -fs ../embedding-files-into-app/assets/...
// $ go run .
// Time to complete the compression and caching of [2/3] files: 31.9998ms
// Total size reduced from 156.6 kB to:
// br      (22.9 kB) [85.37%]
// snappy  (41.7 kB) [73.37%]
// gzip    (27.9 kB) [82.16%]
// deflate (27.9 kB) [82.19%]

var dirOptions = iris.DirOptions{
	IndexName: "index.html",
	// The `Compress` field is ignored
	// when the file is cached (when Cache.Enable is true),
	// because the cache file has a map of pre-compressed contents for each encoding
	// that is served based on client's accept-encoding.
	Compress: true, // true or false does not matter here.
	Cache: iris.DirCacheOptions{
		Enable:         true,
		CompressIgnore: iris.MatchImagesAssets,
		// Here, define the encodings that the cached files should be pre-compressed
		// and served based on client's needs.
		Encodings:       []string{"gzip", "deflate", "br", "snappy"},
		CompressMinSize: 50, // files smaller than this size will NOT be compressed.
		Verbose:         1,
	},
}

func newApp() *iris.Application {
	app := iris.New()
	app.HandleDir("/static", AssetFile(), dirOptions)
	return app
}

func main() {
	app := newApp()

	// http://localhost:8080/static/css/bootstrap.min.css
	// http://localhost:8080/static/js/jquery-2.1.1.js
	// http://localhost:8080/static/favicon.ico
	app.Listen(":8080")
}
