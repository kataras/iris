package main

import "github.com/kataras/iris/v12"

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	app.HandleMany(iris.MethodGet, "/ /api/{page:string suffix(.html)}", handler1)
	app.Get("/api/{name:string suffix(.zip)}", handler2)

	return app
}

func handler1(ctx iris.Context) {
	reply(ctx)
}

func handler2(ctx iris.Context) {
	reply(ctx)
}

func reply(ctx iris.Context) {
	ctx.JSON(iris.Map{
		"handler": ctx.HandlerName(),
		"params":  ctx.Params().Store,
	})
}
