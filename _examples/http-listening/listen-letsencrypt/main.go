// Package main provide one-line integration with letsencrypt.org
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
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

	// NOTE: This will not work on domains like this,
	// use real whitelisted domain(or domains split by whitespaces)
	// and a non-public e-mail instead.
	app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com"))
}
