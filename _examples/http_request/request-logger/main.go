package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/logger"
)

func main() {
	app := iris.New()

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

	h := func(ctx context.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	}
	app.Get("/", h)

	app.Get("/1", h)

	app.Get("/2", h)

	// http errors have their own handlers, therefore
	// registering a middleare should be done manually.
	/*
	 app.OnErrorCode(404 ,customLogger, func(ctx context.Context) {
	 	ctx.Writef("My Custom 404 error page ")
	 })
	*/
	// or catch all http errors:
	app.OnAnyErrorCode(customLogger, func(ctx context.Context) {
		ctx.Writef("My Custom error page")
	})

	// http://localhost:8080
	// http://localhost:8080/1
	// http://localhost:8080/2
	// http://lcoalhost:8080/notfoundhere
	// see the output on the console.
	app.Run(iris.Addr(":8080"))

}
