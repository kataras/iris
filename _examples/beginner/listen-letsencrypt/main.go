// Package main provide one-line integration with letsencrypt.org
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from SECURE SERVER!")
	})

	app.Get("/test2", func(ctx *iris.Context) {
		ctx.Writef("Welcome to secure server from /test2!")
	})

	app.Get("/redirect", func(ctx *iris.Context) {
		ctx.Redirect("/test2")
	})

	// This will provide you automatic certification & key from letsencrypt.org's servers
	// it also starts a second 'http://' server which will redirect all 'http://$PATH' requests to 'https://$PATH'

	// NOTE: may not work on local addresses like this,
	// use it on a real domain, because
	// it uses the 	"golang.org/x/crypto/acme/autocert" package.
	app.ListenLETSENCRYPT("localhost:443")
}
