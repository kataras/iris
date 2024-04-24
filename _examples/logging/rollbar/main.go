package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/requestid"

	"github.com/rollbar/rollbar-go"
)

// * https://rollbar.com/signup
// * https://docs.rollbar.com/docs/go
func init() {
	token := os.Getenv("ROLLBAR_TOKEN") // replace that with your token.
	if token == "" {
		panic("ROLLBAR_TOKEN is missing")
	}

	// rb := rollbar.NewAsync(token, "production", "", hostname, "github.com/kataras/iris")
	// Or use the package-level instance:
	rollbar.SetToken(token)
	// defaults to "development"
	rollbar.SetEnvironment("production")
	// optional Git hash/branch/tag (required for GitHub integration)
	// rollbar.SetCodeVersion("v2")
	// optional override; defaults to hostname
	// rollbar.SetServerHost("web.1")
	// path of project (required for GitHub integration and non-project stacktrace collapsing)
	rollbar.SetServerRoot("github.com/kataras/iris")

}

func main() {
	app := iris.New()
	// A middleware which sets the ctx.GetID (or requestid.Get(ctx)).
	app.Use(requestid.New())
	// A recover middleware which sends the error trace to the rollbar.
	app.Use(func(ctx iris.Context) {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()

				file, line := ctx.HandlerFileLine() // the failed handler's source code position.

				//					cause				   other info
				rollbar.Critical(errors.New(fmt.Sprint(r)), iris.Map{
					"request_id":  ctx.GetID(),
					"request_ip":  ctx.RemoteAddr(),
					"request_uri": ctx.FullRequestURI(),
					"handler": iris.Map{
						"name": ctx.HandlerName(), // the handler which failed.
						"file": fmt.Sprintf("%s:%d", file, line),
					},
				})

				ctx.StopWithStatus(iris.StatusInternalServerError)
			}
		}()

		ctx.Next()
	})

	app.Get("/", index)
	app.Get("/panic", panicMe)

	// http://localhost:8080 should add an info message to the rollbar's "Items" dashboard.
	// http://localhost:8080/panic should add a critical message to the rollbar's "Items" dashboard,
	// with the corresponding information appending on its "Occurrences" tab item, e.g:
	// Timestamp (PDT)
	// * 2020-06-08 04:47 pm
	//
	// server.host
	// * DESKTOP-HOSTNAME
	//
	// trace_chain.0.exception.message
	// * a critical error message here
	//
	// custom.handler.file
	// * C:/mygopath/src/github.com/kataras/iris/_examples/logging/rollbar/main.go:76
	//
	// custom.handler.name
	// * main.panicMe
	//
	// custom.request_id
	// * cce61665-0c1b-4fb5-8547-06a3537e477c
	//
	// custom.request_ip
	// * ::1
	//
	// custom.request_uri
	// * http://localhost:8080/panic
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	rollbar.Info(fmt.Sprintf("Index page requested by %s", ctx.RemoteAddr()))

	ctx.HTML("<h1> Index Page </h1>")
}

func panicMe(ctx iris.Context) {
	panic("a critical error message here")
}
