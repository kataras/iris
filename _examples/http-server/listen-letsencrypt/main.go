// Package main provide one-line integration with letsencrypt.org
package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from SECURE SERVER!")
	})

	app.Get("/test2", func(ctx iris.Context) {
		ctx.Writef("Welcome to secure server from /test2!")
	})

	app.Get("/redirect", func(ctx iris.Context) {
		ctx.Redirect("/test2")
	})

	// NOTE: This will not work on domains like this,
	// use real whitelisted domain(or domains split by whitespaces)
	// and a non-public e-mail instead or edit your hosts file.
	app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com"))

	// Note: to disable automatic "http://" to "https://" redirections pass the `iris.TLSNoRedirect`
	// host configurator to TLS or AutoTLS functions, e.g:
	//
	// app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com", iris.TLSNoRedirect))
}
