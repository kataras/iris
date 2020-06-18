package main

import "github.com/kataras/iris/v12"

func main() {
	app := newApp()
	// Navigate through https://github.com/kataras/iris/issues/1448 for details.
	//
	// GET: http://localhost:8080
	// POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE : http://localhost:8080
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	// Skip and do NOT override existing regitered route, continue normally.
	// Applies to a Party and its children, in this case the whole application's routes.
	app.SetRegisterRule(iris.RouteSkip)

	/* Read also:
	// The default behavior, will override the getHandler to anyHandler on `app.Any` call.
	app.SetRegistRule(iris.RouteOverride)

	// Stops the execution and fires an error before server boot.
	app.SetRegisterRule(iris.RouteError)

	// If ctx.StopExecution or StopWithXXX then the next route will be executed
	// (see mvc/authenticated-controller example too).
	app.SetRegisterRule(iris.RouteOverlap)
	*/

	app.Get("/", getHandler)
	// app.Any does NOT override the previous GET route because of `iris.RouteSkip` rule.
	app.Any("/", anyHandler)

	return app
}

func getHandler(ctx iris.Context) {
	ctx.Writef("From GET: %s", ctx.GetCurrentRoute().MainHandlerName())
}

func anyHandler(ctx iris.Context) {
	ctx.Writef("From %s: %s", ctx.Method(), ctx.GetCurrentRoute().MainHandlerName())
}
