package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.OnErrorCode(iris.StatusInternalServerError, func(ctx iris.Context) {
		ctx.HTML("Message: <b>" + ctx.Values().GetString("message") + "</b>")
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML(`Click <a href="/my500">here</a> to fire the 500 status code`)
	})

	app.Get("/my500", func(ctx iris.Context) {
		ctx.Values().Set("message", "this is the error message")
		ctx.StatusCode(500)
	})

	app.Get("/u/{firstname:alphabetical}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("firstname"))
	})

	app.Run(iris.Addr(":8080"))
}
