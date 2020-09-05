package main

import (
	"regexp"

	"github.com/kataras/iris/v12"
)

// How to run:
// $ go get -u github.com/go-bindata/go-bindata/...
// # OR go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// $ go-bindata -nomemcopy -fs -prefix "../http2push/assets" ../http2push/assets/...
// $ go run .

var opts = iris.DirOptions{
	IndexName: "index.html",
	PushTargetsRegexp: map[string]*regexp.Regexp{
		"/":              iris.MatchCommonAssets,
		"/app2/app2app3": iris.MatchCommonAssets,
	},
	ShowList: true,
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

func main() {
	app := iris.New()
	app.HandleDir("/public", AssetFile(), opts)

	// https://127.0.0.1/public
	// https://127.0.0.1/public/app2
	// https://127.0.0.1/public/app2/app2app3
	// https://127.0.0.1/public/app2/app2app3/dirs
	app.Run(iris.TLS(":443", "../http2push/mycert.crt", "../http2push/mykey.key"))
}
