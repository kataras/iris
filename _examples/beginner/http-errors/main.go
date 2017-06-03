package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.OnErrorCode(iris.StatusInternalServerError, func(ctx context.Context) {
		ctx.HTML("Message: <b>" + ctx.Values().GetString("message") + "</b>")
	})

	app.Get("/", func(ctx context.Context) {
		ctx.HTML(`Click <a href="/my500">here</a> to fire the 500 status code`)
	})

	app.Get("/my500", func(ctx context.Context) {
		ctx.Values().Set("message", "this is the error message")
		ctx.StatusCode(500)
	})

	app.Get("/u/{firstname:alphabetical}", func(ctx context.Context) {
		ctx.Writef("Hello %s", ctx.Values().GetString("firstname"))
	})

	app.Run(iris.Addr(":8080"))
}
