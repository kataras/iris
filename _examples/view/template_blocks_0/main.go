package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	// Read about its syntax at: https://github.com/kataras/blocks
	app.RegisterView(iris.Blocks("./views", ".html").Reload(true))

	app.Get("/", index)
	app.Get("/500", internalServerError)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title": "Page Title",
	}

	ctx.ViewLayout("main")
	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func internalServerError(ctx iris.Context) {
	ctx.StatusCode(iris.StatusInternalServerError)

	data := iris.Map{
		"Code":    iris.StatusInternalServerError,
		"Message": "Internal Server Error",
	}

	ctx.ViewLayout("error")
	if err := ctx.View("500", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
