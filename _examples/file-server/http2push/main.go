package main

import (
	"regexp"

	"github.com/kataras/iris/v12"
)

var opts = iris.DirOptions{
	IndexName: "/index.html",
	// Optionally register files (map's values) to be served
	// when a specific path (map's key WITHOUT prefix) is requested
	// is fired before client asks (HTTP/2 Push).
	// E.g. "/" (which serves the `IndexName` if not empty).
	//
	// Note: Requires running server under TLS,
	// that's why we use `iris.TLS` below.
	// PushTargets: map[string][]string{
	// 	"/": { // Relative path without prefix.
	// 		"favicon.ico",
	// 		"js/main.js",
	// 		"css/main.css",
	// 		// ^ Relative to the index, if need absolute ones start with a slash ('/').
	// 	},
	// },
	// OR use regexp:
	PushTargetsRegexp: map[string]*regexp.Regexp{
		// Match all js, css and ico files
		// from all files (recursively).
		// "/": regexp.MustCompile("((.*).js|(.*).css|(.*).ico)$"),
		// OR:
		"/":              iris.MatchCommonAssets,
		"/app2/app2app3": iris.MatchCommonAssets,
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
