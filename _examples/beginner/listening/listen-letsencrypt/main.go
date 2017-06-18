// Package main provide one-line integration with letsencrypt.org
package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.Writef("Hello from SECURE SERVER!")
	})

	app.Get("/test2", func(ctx context.Context) {
		ctx.Writef("Welcome to secure server from /test2!")
	})

	app.Get("/redirect", func(ctx context.Context) {
		ctx.Redirect("/test2")
	})

	// If http to https auto-redirect is one of your needs
	// please look the code inside iris_deprecateed.go.ListenLETSENCRYPT to do it manually.

	// NOTE: This may not work on local addresses like this,
	// use it on a real domain, because
	// it uses the 	"golang.org/x/crypto/acme/autocert" package.
	app.Run(iris.AutoTLS("localhost:443"))
}
