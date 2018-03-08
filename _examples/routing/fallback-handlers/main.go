package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()

	// add a fallback handler to process requests that would not be declared in the router.
	app.Fallback(fallbackHandler)

	// this works as expected now,
	// will handle *all* expect DELETE requests, even if there is no routes.
	app.Get("/action/{p}", h)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func h(ctx iris.Context) {
	ctx.Writef("[%s] %s : Parameter = `%s`", ctx.Method(), ctx.Path(), ctx.Params().Get("p"))
}

func fallbackHandler(ctx iris.Context) {
	if ctx.Method() == iris.MethodDelete {
		ctx.Next()
		return
	}

	ctx.Writef("[%s] %s : From fallback handler", ctx.Method(), ctx.Path())
}
