package root

import "github.com/kataras/iris/v12"

func init() {
	app := iris.New()
	app.SetName("root app")

	app.Get("/", index)
}

func index(ctx iris.Context) {
	ctx.HTML("Main Root Index (App Name: <b>%s</b> | Host: <b>%s</b>)",
		ctx.Application().String(), ctx.Host())
}
