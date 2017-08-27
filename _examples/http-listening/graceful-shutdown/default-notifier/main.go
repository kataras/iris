package main

import (
	stdContext "context"
	"time"

	"github.com/kataras/iris"
)

// Before continue:
//
// Gracefully Shutdown on control+C/command+C or when kill command sent is ENABLED BY-DEFAULT.
//
// In order to manually manage what to do when app is interrupted,
// We have to disable the default behavior with the option `WithoutInterruptHandler`
// and register a new interrupt handler (globally, across all possible hosts).
func main() {
	app := iris.New()

	iris.RegisterOnInterrupt(func() {
		timeout := 5 * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		// close all hosts
		app.Shutdown(ctx)
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
	})

	// http://localhost:8080
	app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)
}
