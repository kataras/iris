package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from the SECURE server")
	})

	app.Get("/mypath", func(ctx iris.Context) {
		ctx.Writef("Hello from the SECURE server on path /mypath")
	})

	// Start the server (HTTPS) on port 443,
	// and a secondary of (HTTP) on port :80 which redirects requests to their HTTPS version.
	// This is a blocking func.
	app.Run(iris.TLS("127.0.0.1:443", "mycert.crt", "mykey.key"))

	// Note: to disable automatic "http://" to "https://" redirections pass the `iris.TLSNoRedirect`
	// host configurator to TLS function, example:
	//
	// app.Run(iris.TLS("127.0.0.1:443", "mycert.crt", "mykey.key", iris.TLSNoRedirect))
}
