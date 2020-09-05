package main

import (
	"regexp"

	"github.com/kataras/iris/v12"
)

// How to run:
// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
// $ go-bindata -nomemcopy -fs -prefix "../http2push/assets" ../http2push/assets/...
// # OR if the ./assets directory was inside this example foder:
// # go-bindata -nomemcopy -refix "assets" ./assets/...
//
// $ go run .
// Physical files are not used, you can delete the "assets" folder and run the example.

var opts = iris.DirOptions{
	IndexName: "index.html",
	PushTargetsRegexp: map[string]*regexp.Regexp{
		"/":              iris.MatchCommonAssets,
		"/app2/app2app3": iris.MatchCommonAssets,
	},
	Compress: false,
	ShowList: true,
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
