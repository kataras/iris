// Observability endpoints: the live process dashboard, the Go profiler, the expvar gauges the dashboard publishes, and per-request cost measurement.
//
// Every one of these reveals internals (host metrics, source file names,
// execution traces), so they are mounted behind basic authentication, never
// publicly.
//
// Learn: mount the monitor dashboard, pprof and expvar behind basic auth, and read what each request cost in CPU, memory and network.
// Before: ../graceful-shutdown
//
// Run:
//
//	ADMIN_PASSWORD=change-me go run main.go
//
// Try:
//
//	open http://localhost:8080/admin/monitor                                   # the dashboard, sign in as admin:change-me
//	curl -i localhost:8080/admin/monitor                                       # 401, no credentials
//	curl -u admin:change-me localhost:8080/admin/debug/vars                    # 200, expvar JSON with the monitor's gauges
//	curl -u admin:change-me localhost:8080/admin/debug/pprof/heap -o heap.out  # 200, a heap profile for go tool pprof
//	curl localhost:8080/reports                                                # 200 {"report":"..."}, cost logged by the server
//
// Next: ../../16-advanced/std-interop
package main

import (
	"expvar"
	"log"
	"os"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/middleware/httpcost"
	"github.com/kataras/iris/v14/middleware/monitor"
	"github.com/kataras/iris/v14/middleware/pprof"
)

func main() {
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Fatal("ADMIN_PASSWORD environment variable is required")
	}

	app := iris.New(iris.Options{Name: "observability", LogLevel: "info"})

	// Priced routes. The prices are per unit; setting one to zero disables that
	// dimension and at least one must stay enabled. CPU and memory are sampled
	// process-wide, so under concurrency each request is attributed a share of
	// everything that ran at the same time: consistent approximations, not an
	// invoice. The pair is registered on this group rather than on the whole
	// application: you meter the API, not your own dashboards.
	reports := app.Router("/reports", costReporter, httpcost.New(httpcost.Options{
		CostPerSecond: 0.01,  // per wall-clock second
		CostPerMB:     0.001, // per MB of RAM and network
		CostPerCPU:    0.02,  // per CPU-second
	}))
	reports.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"report": "..."})
	})

	// The monitor samples process and OS statistics on a fixed interval.
	// RefreshInterval has no usable default: without a positive value the
	// collector never ticks.
	m := monitor.New(monitor.Options{
		RefreshInterval:     2 * time.Second, // stats collection
		ViewRefreshInterval: 2 * time.Second, // dashboard refresh
		ViewTitle:           "Observability Example",
	})
	// Stop the collector on shutdown: Deferrables(m.Stop) in a builder app.
	iris.RegisterOnInterrupt(m.Stop)

	admin := app.Router("/admin", basicauth.Default(map[string]string{
		"admin": adminPassword,
	}))

	// The dashboard page fetches its numbers with a POST to its own URL, so
	// both handlers mount on the same path.
	admin.Get("/monitor", m.View)   // HTML dashboard
	admin.Post("/monitor", m.Stats) // JSON stats, used by the view

	// The profiler as a routing controller: GET /admin/debug/pprof lists the
	// available profiles (heap, goroutine, allocs, block, mutex, profile,
	// trace). They feed the Go toolchain directly:
	//
	//	go tool pprof http://admin:change-me@localhost:8080/admin/debug/pprof/heap
	admin.Router("/debug/pprof", pprof.New())

	// The monitor publishes its gauges (pid_cpu, os_ram, ...) through the
	// standard expvar package; iris.FromStd mounts expvar's own handler.
	admin.Any("/debug/vars", iris.FromStd(expvar.Handler()))

	app.Listen(":8080")
}

// costReporter reads what the cost middleware measured. The Metric lands on
// the context only after the handler chain has returned, so it can only be
// read by something that wraps the chain: this runs first, calls Next, and
// picks the metric up on the way back. An access log field setter
// (accesslog.AddFields) is the other natural place.
func costReporter(ctx iris.Context) {
	ctx.Next()

	if metric, ok := httpcost.Get(ctx); ok {
		ctx.Application().Logger().Infof("%s %s cost=%.6f cpu=%.3fs mem=%dB net=%dB",
			ctx.Method(), ctx.Path(),
			metric.Cost, metric.CpuTime, metric.MemoryUsed, metric.NetworkUsed)
	}
}
