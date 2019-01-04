package main

import (
	"context"
	"time"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
	})

	app.Logger().Info("Wait 10 seconds and check your terminal again")
	// simulate a shutdown action here...
	go func() {
		<-time.After(10 * time.Second)
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// close all hosts, this will notify the callback we had register
		// inside the `configureHost` func.
		app.Shutdown(ctx)
	}()

	// app.ConfigureHost(configureHost) -> or pass "configureHost" as `app.Addr` argument, same result.

	// start the server as usual, the only difference is that
	// we're adding a second (optional) function
	// to configure the just-created host supervisor.
	//
	// http://localhost:8080
	// wait 10 seconds and check your terminal.
	app.Run(iris.Addr(":8080", configureHost), iris.WithoutServerError(iris.ErrServerClosed))

	/*
		Or for simple cases you can just use the:
		iris.RegisterOnInterrupt for global catch of the CTRL/CMD+C and OS events.
		Look at the "graceful-shutdown" example for more.
	*/
}

func configureHost(su *iris.Supervisor) {
	// here we have full access to the host that will be created
	// inside the `app.Run` function or `NewHost`.
	//
	// we're registering a shutdown "event" callback here:
	su.RegisterOnShutdown(func() {
		println("server is closed")
	})
	// su.RegisterOnError
	// su.RegisterOnServe
}
