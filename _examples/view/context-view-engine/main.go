package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	// Register a root view engine, as usual,
	// will be used to render files through Context.View method
	// when no Party or Handler-specific view engine is available.
	app.RegisterView(iris.Blocks("./views/public", ".html"))

	// http://localhost:8080
	app.Get("/", index)

	// Register a view engine per group of routes.
	adminGroup := app.Party("/admin")
	adminGroup.RegisterView(iris.Blocks("./views/admin", ".html"))

	// http://localhost:8080/admin
	adminGroup.Get("/", admin)

	// Register a view engine on-fly for the current chain of handlers.
	views := iris.Blocks("./views/on-fly", ".html")
	if err := views.Load(); err != nil {
		app.Logger().Fatal(err)
	}

	// http://localhost:8080/on-fly
	app.Get("/on-fly", setViews(views), onFly)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title": "Public Index Title",
	}

	ctx.ViewLayout("main")
	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func admin(ctx iris.Context) {
	data := iris.Map{
		"Title": "Admin Panel",
	}

	ctx.ViewLayout("main")
	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func setViews(views iris.ViewEngine) iris.Handler {
	return func(ctx iris.Context) {
		ctx.ViewEngine(views)
		ctx.Next()
	}
}

func onFly(ctx iris.Context) {
	data := iris.Map{
		"Message": "View engine changed through 'setViews' custom middleware.",
	}

	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
