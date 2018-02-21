package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// this works as expected now,
	// will handle *all* expect DELETE requests, even if there is no routes
	app.Get("/action/{p}", h)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func h(ctx iris.Context) {
	ctx.Writef("[%s] %s : Parameter = `%s`", ctx.Method(), ctx.Path(), ctx.Params().Get("p"))
}

func fallbackHandler(ctx iris.Context) {
	if ctx.Method() == "DELETE" {
		ctx.Next()

		return
	}

	ctx.Writef("[%s] %s : From fallback handler", ctx.Method(), ctx.Path())
}
