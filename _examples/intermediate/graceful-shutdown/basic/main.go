package main

import (
	stdContext "context"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
)

// Before continue, please read the below notes:
//
// Current version of Iris is auto-graceful on control+C/command+C
// or kill command sent or whenever app.Shutdown called.
//
// In order to add a custom interrupt handler(ctrl+c/cmd+c) or
// shutdown manually you have to "schedule a host supervisor's task" or
// use the core/host package manually or use a pure http.Server as we already seen at "custom-server" example.
//
// At this example, we will disable the interrupt handler and set our own interrupt handler.
func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout

	app.Get("/", func(ctx context.Context) {
		ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
	})

	// tasks are always running in their go-routine by-default.
	//
	// register custom interrupt handler, fires when ctrl+C/cmd+C pressed or kill command sent.
	app.Scheduler.Schedule(host.OnInterrupt(func(proc host.TaskProcess) {
		println("Shutdown the server gracefully...")

		timeout := 5 * time.Second // give the server 5 seconds to wait for idle connections.
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		proc.Host().Shutdown(ctx) // Shutdown the underline supervisor's server
	}))

	// Start the server and disable the default interrupt handler in order to use our scheduled interrupt task.
	app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)
}

// Note:
// You can just use an http.Handler with your own signal notify channel and do that as you did with the net/http
// package. I will not show this way, but you can find many examples on the internet.
