// Conditional middleware: a debug bundle that runs only when the request asks
// for it.
//
// iris.NewConditionalHandler(filter, handlers...) packs a filter and one or
// more middleware into a single registerable handler. The filter is an
// iris.Filter - func(iris.Context) bool. When it returns true the handlers are
// spliced into the current request's chain and run in order; when it returns
// false the chain simply continues without them. The wrapped handlers must
// behave like middleware, i.e. call ctx.Next().
//
// Here a debug bundle (request echo + timing log) is gated on an "X-Debug: 1"
// request header, so its cost is paid only when someone asks for it.
//
// For coarse conditions you rarely need this: attach the middleware to the
// child router or subdomain whose requests it concerns and let the routing
// structure do the filtering. Reach for a conditional handler when the
// decision is genuinely per-request.
//
// Learn: gate one or more middleware on a per-request iris.Filter with
// iris.NewConditionalHandler.
// Before: ../writing-middleware
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/items                           # 200, no X-Debug-* headers
//	curl -i -H "X-Debug: 1" http://localhost:8080/items           # 200, X-Debug-* headers
//	curl -i -X POST -H "X-Debug: 1" http://localhost:8080/items   # 201, X-Debug-Method: POST
//
// Next: ../error-middleware
package main

import (
	"time"

	"github.com/kataras/iris/v14"
)

// debugRequested is an iris.Filter: the request decides, not the route.
// Naming it makes it reusable and testable on its own.
var debugRequested iris.Filter = func(ctx iris.Context) bool {
	return ctx.Header("X-Debug") == "1"
}

func main() {
	app := iris.New()

	// One registered handler, two middleware behind the filter.
	app.Use(iris.NewConditionalHandler(debugRequested,
		echoRequestInfo, timeRequest))

	app.Get("/items", func(ctx iris.Context) {
		ctx.JSON([]string{"pen", "notebook"})
	})
	app.Post("/items", func(ctx iris.Context) {
		ctx.Created(iris.Map{"created": true}, nil)
	})

	app.Listen(":8080")
}

// echoRequestInfo mirrors back what the server saw. Response headers are set
// before Next, while the body has not been written yet.
func echoRequestInfo(ctx iris.Context) {
	ctx.SetHeader("X-Debug-Method", ctx.Method())
	ctx.SetHeader("X-Debug-Path", ctx.Path())
	ctx.SetHeader("X-Debug-Remote", ctx.RemoteAddr())

	ctx.Next()
}

// timeRequest measures the chain below it. Anything after Next can only log
// or record - the response is already sent by then.
func timeRequest(ctx iris.Context) {
	start := time.Now()

	ctx.Next()

	ctx.Application().Logger().Infof("debug: %s %s took %s",
		ctx.Method(), ctx.Path(), time.Since(start))
}
