package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/middleware/logger"
)

func main() {
	app := iris.New()

	app.Adapt(iris.DevLogger()) // it just enables the print of the iris.DevMode logs. Enable it to view the middleware's messages.
	app.Adapt(httprouter.New())

	customLogger := logger.New(logger.Config{
		// Status displays status code
		Status: true,
		// IP displays request's remote address
		IP: true,
		// Method displays the http method
		Method: true,
		// Path displays the request path
		Path: true,
	})

	app.Use(customLogger)

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("hello")
	})

	app.Get("/1", func(ctx *iris.Context) {
		ctx.Writef("hello")
	})

	app.Get("/2", func(ctx *iris.Context) {
		ctx.Writef("hello")
	})

	// log http errors
	errorLogger := logger.New()

	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		errorLogger.Serve(ctx)
		ctx.Writef("My Custom 404 error page ")
	})

	// http://localhost:8080
	// http://localhost:8080/1
	// http://localhost:8080/2
	app.Listen(":8080")

}
