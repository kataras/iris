package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// this works as expected now,
	// will handle all GET requests
	// except:
	// /                     -> because of app.Get("/", ...)
	// /other/anything/here  -> because of app.Get("/other/{paramother:path}", ...)
	// /other2/anything/here -> because of app.Get("/other2/{paramothersecond:path}", ...)
	// /other2/static2        -> because of app.Get("/other2/static", ...)
	//
	// It isn't conflicts with the rest of the routes, without routing performance cost!
	//
	// i.e /something/here/that/cannot/be/found/by/other/registered/routes/order/not/matters
	app.Get("/{p:path}", h)
	// app.Get("/static/{p:path}", staticWildcardH)

	// this will handle only GET /
	app.Get("/", staticPath)

	// this will handle all GET requests starting with "/other/"
	//
	// i.e /other/more/than/one/path/parts
	app.Get("/other/{paramother:path}", other)

	// this will handle all GET requests starting with "/other2/"
	// except /other2/static (because of the next static route)
	//
	// i.e /other2/more/than/one/path/parts
	app.Get("/other2/{paramothersecond:path}", other2)

	// this will handle only GET "/other2/static"
	app.Get("/other2/static2", staticPathOther2)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func h(ctx iris.Context) {
	param := ctx.Params().Get("p")
	ctx.WriteString(param)
}

func staticWildcardH(ctx iris.Context) {
	param := ctx.Params().Get("p")
	ctx.WriteString("from staticWildcardH: param=" + param)
}

func other(ctx iris.Context) {
	param := ctx.Params().Get("paramother")
	ctx.Writef("from other: %s", param)
}

func other2(ctx iris.Context) {
	param := ctx.Params().Get("paramothersecond")
	ctx.Writef("from other2: %s", param)
}

func staticPath(ctx iris.Context) {
	ctx.Writef("from the static path(/): %s", ctx.Path())
}

func staticPathOther2(ctx iris.Context) {
	ctx.Writef("from the static path(/other2/static2): %s", ctx.Path())
}
