package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
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
		// Query appends the url query to the Path.
		Query: true,
		// Shows information about the executed route.
		TraceRoute: true,
		// Columns: true,

		// if !empty then its contents derives from `ctx.Values().Get("logger_message")
		// will be added to the logs.
		MessageContextKeys: []string{"logger_message"},

		// if !empty then its contents derives from `ctx.GetHeader("User-Agent")
		MessageHeaderKeys: []string{"User-Agent"},
	})

	// Runs first on every request: parties & subdomains, route match or not and all http errors.
	app.UseRouter(customLogger)

	// Runs first on each matched route of this Party and its children on every request.
	app.Use(routesMiddleware)

	app.Get("/", indexMiddleware, index)
	app.Get("/list", listMiddleware, list)

	app.Get("/1", hello)
	app.Get("/2", hello)

	app.OnAnyErrorCode(customLogger, func(ctx iris.Context) {
		// this should be added to the logs, at the end because of the `logger.Config#MessageContextKey`
		ctx.Values().Set("logger_message",
			"a dynamic message passed to the logs")
		ctx.Writef("My Custom error page")
	})

	// http://localhost:8080
	// http://localhost:8080/list
	// http://localhost:8080/list?stop=true
	// http://localhost:8080/1
	// http://localhost:8080/2
	// http://lcoalhost:8080/notfoundhere
	// see the output on the console.
	app.Listen(":8080")
}

func routesMiddleware(ctx iris.Context) {
	ctx.Writef("Executing Route: %s\n", ctx.GetCurrentRoute().MainHandlerName())
	ctx.Next()
}

func indexMiddleware(ctx iris.Context) {
	ctx.WriteString("Index Middleware\n")
	ctx.Next()
}

func index(ctx iris.Context) {
	ctx.WriteString("Index Handler")
}

func listMiddleware(ctx iris.Context) {
	ctx.WriteString("List Middleware\n")

	if simulateStop, _ := ctx.URLParamBool("stop"); !simulateStop {
		ctx.Next()
	}
}

func list(ctx iris.Context) {
	ctx.WriteString("List Handler")
}

func hello(ctx iris.Context) {
	ctx.Writef("Hello from %s", ctx.Path())
}
