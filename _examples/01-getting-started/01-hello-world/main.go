// Step 1 of 13: the smallest Iris program.
//
// This is the first step of a tutorial that grows one service, the Bookshelf
// (shelves hold books; a locked shelf freezes its books), from a single route
// to a tested, production-shaped API. Every step is a complete program that
// adds one idea to the previous one. Diff two neighbours to see what changed.
//
// iris.New() builds an application with sensible defaults, app.Get registers
// a route, and app.Listen serves it with graceful shutdown already wired to
// SIGINT/SIGTERM (Ctrl+C).
//
// Learn: start an Iris server, register a route and return JSON.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i localhost:8080/        # 200 {"message":"Hello, Bookshelf!","framework":"github.com/kataras/iris/v14"}
//
// Next: step 2 adds the first real routes and a typed path parameter (../02-routes-and-params).
package main

import (
	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// Every handler receives an iris.Context: the single object used to read
	// the request and write the response. ctx.JSON serializes any Go value
	// and sets the Content-Type header. iris.Map is a shorthand for
	// map[string]any.
	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"message":   "Hello, Bookshelf!",
			"framework": "github.com/kataras/iris/v14",
		})
	})

	// Listen blocks until the server stops. Ctrl+C drains the in-flight
	// requests and returns normally; a failed start is logged and exits the
	// process with status 1, so there is no error to check here.
	app.Listen(":8080")
}
