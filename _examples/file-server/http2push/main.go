package main

import (
	"github.com/kataras/iris/v12"
)

var opts = iris.DirOptions{
	IndexName: "/index.html",
	// Optionally register files (map's absolute values) to be served
	// when a specific path (map's key WITHOUT prefix) is requested
	// is fired before client asks (HTTP/2 Push).
	// E.g. "/" (which serves the `IndexName` if not empty).
	//
	// Note: Requires running server under TLS,
	// that's why we use ListenAndServeTLS below.
	PushTargets: map[string][]string{
		"/": {
			"/public/favicon.ico",
			"/public/js/main.js",
			"/public/css/main.css",
		},
	},
	Compress: true,
	ShowList: true,
}

func main() {
	app := iris.New()
	app.HandleDir("/public", "./assets", opts)

	// Open your browser's Network tools,
	// navigate to https://127.0.0.1/public.
	// you should see `Initiator` tab:  "Push / public".
	app.Run(iris.TLS(":443", "mycert.crt", "mykey.key"))
}
