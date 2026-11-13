// Graceful shutdown: what runs when the process is interrupted, in which order, and how long the server waits for in-flight requests.
//
// Iris installs the interrupt handler itself. Unless ServerOptions
// DisableInterruptHandler is set, SIGINT and SIGTERM (Ctrl+C, docker stop, a
// Kubernetes pod termination) stop the listener and drain in-flight requests
// for up to ShutdownOnInterruptTimeout, 15 seconds by default, set below
// through iris.Timeout(). Your own cleanup is a separate list, run by
// app.Shutdown(ctx); iris.RegisterOnInterrupt is what connects the two.
//
// Learn: make Ctrl+C drain the requests already running, and hand your own closers to the same signal.
// Before: ../serving-and-tls
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl localhost:8080/jobs/slow &   # 200 {"status":"done"} after 5 seconds, even if you press Ctrl+C while it runs
//
// Next: ../observability
package main

import (
	stdContext "context"
	"log"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := newApp()

	// The server's built-in handler drains connections but knows nothing about
	// your closers: those belong to app.Shutdown, which first stops the server
	// (Run prepends that step) and then runs the Deferrables and the Close
	// methods Services collected. Registering it here is what makes Ctrl+C run
	// the whole list. This closer is registered before the server's own, so it
	// goes first, and its context bounds the drain.
	iris.RegisterOnInterrupt(func() error {
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 10*time.Second)
		defer cancel()

		log.Println("jobs: interrupt received, draining")
		return app.Shutdown(ctx)
	})

	// Listen uses the default timeouts. Run spells them out: 20 seconds for the
	// server's own drain on interrupt, instead of the 15-second default. A
	// failed start is logged and exits the process.
	//
	// Run returns once app.Shutdown has run its whole list, not when the server
	// has drained, and main ends with it. Interrupt it while /jobs/slow is
	// running: "jobs: flushing caches", "jobs: goodbye" and "worker: stopping"
	// are all printed before the process exits. Nothing has to keep main alive
	// after Run.
	app.Run(iris.ServerOptions{
		Addr:      ":8080",
		Network:   "tcp",
		KeepAlive: 3 * time.Minute,
		Timeout:   iris.Timeout().WithShutdownOnInterruptTimeout(20 * time.Second),
	})
}

// Worker is a background job with a Close method. Passing it to Services both
// injects it and registers Close as a shutdown closer, with no separate
// Deferrables entry needed.
type Worker struct {
	done chan struct{}
}

func NewWorker() *Worker {
	w := &Worker{done: make(chan struct{})}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.done:
				return
			case <-ticker.C:
				log.Println("worker: tick")
			}
		}
	}()

	return w
}

func (w *Worker) Close() error {
	log.Println("worker: stopping")
	close(w.done)
	return nil
}

// JobsAPI is the only controller: it exists so there is something to drain.
type JobsAPI struct {
	worker *Worker
}

func NewJobsAPI(worker *Worker) *JobsAPI { return &JobsAPI{worker: worker} }

func (api *JobsAPI) Routing(r iris.Router) {
	r.Get("/slow", api.slow)
}

// slow takes five seconds. Interrupt the server while it runs: the response
// still arrives, because the drain waits for handlers that already started.
func (api *JobsAPI) slow(ctx iris.Context) {
	select {
	case <-time.After(5 * time.Second):
		ctx.JSON(iris.Map{"status": "done"})
	case <-ctx.Request().Context().Done():
		log.Println("jobs: client went away")
	}
}

func newApp() *iris.Application {
	return iris.NewBuilder(iris.Options{Name: "jobs", LogLevel: "info"}).
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions()).
		// Deferrables takes func(), func() error, io.Closer and the
		// context-aware variants. They run in registration order, after the
		// server has stopped serving.
		Deferrables(
			func() { log.Println("jobs: flushing caches") },
			func() error { log.Println("jobs: goodbye"); return nil },
		).
		Services(NewWorker()). // Close picked up automatically.
		RouterMiddlewares().
		Middlewares().
		API("/jobs", NewJobsAPI).
		Build()
}
