package main

import (
	stdContext "context"
	"net/http"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
)

// The difference from its parent main.go is that
// with a custom host we're able to call the host's shutdown
// and be notified about its .Shutdown call.
// Almost the same as before.
func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout

	app.Get("/", func(ctx context.Context) {
		ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
	})

	// build the framework when all routing-relative actions are declared.
	if err := app.Build(); err != nil {
		panic(err)
	}

	// create our custom host supervisor by adapting
	// a custom http server
	srv := host.New(&http.Server{Addr: ":8080", Handler: app})

	// tasks are always running in their go-routine by-default.
	//
	// register custom interrupt handler, fires when ctrl+C/cmd+C pressed or kill command sent, as we did before.
	srv.Schedule(host.OnInterrupt(func(proc host.TaskProcess) {
		println("Shutdown the server gracefully...")

		timeout := 5 * time.Second // give the server 5 seconds to wait for idle connections.
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		proc.Host().DeferFlow() // defer the exit, in order to catch the event
		srv.Shutdown(ctx)       // Shutdown the supervisor, only this way the proc.Host().Done will be notified
		// the proc.Host().Shutdown, closes the underline server but no the supervisor.
		// This behavior was choosen because is inneed for custom maintanance services
		// (i.e restart server every week without loosing connections) which you can easly adapt with Iris.
	}))

	// schedule a task to be notify when the server was closed by our interrutp handler,
	// optionally ofcourse.
	srv.ScheduleFunc(func(proc host.TaskProcess) {
		select {
		case <-proc.Host().Done(): // when .Host.Shutdown(ctx) called.
			println("Server was closed.")
			proc.Host().RestoreFlow() // Restore the flow in order to exit(continue after the srv.ListenAndServe)
		}
	})

	// Start our custom host
	println("Server is running at :8080")
	srv.ListenAndServe()
	// Go to the console and press ctrl+c(for windows and linux) or cmd+c for osx.
	// The output should be:
	// Server is running at:8080
	// Shutdown the server gracefully...
	// Server was closed.
}

// Note:
// You can just use an http.Handler with your own signal notify channel and do that as you did with the net/http
// package. I will not show this way, but you can find many examples on the internet.
