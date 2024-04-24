package main

import "github.com/kataras/iris/v12"

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	api := app.Party("/api")
	api.Use(myMiddleware)
	users := api.Party("/users")
	users.Get("/", usersIndex).RemoveHandler(myMiddleware)
	// OR for all routes under a Party (or Application):
	// users.RemoveHandler(...)

	return app
}

func myMiddleware(ctx iris.Context) {
	ctx.WriteString("Middleware\n")
}

func usersIndex(ctx iris.Context) {
	ctx.WriteString("OK")
}
