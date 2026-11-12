// Package monitor reports live process and system statistics (CPU, memory,
// goroutines, uptime) as JSON and as a small HTML dashboard.
//
// Its entry point is New(Options), which returns a *Monitor rather than a
// Handler because it exposes two of them - `Stats` for the JSON payload and
// `View` for the dashboard page - and owns the background sampler that both
// read; call `Stop` when it is no longer needed.
package monitor

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/kataras/iris/v14/context"

	"github.com/shirou/gopsutil/v4/process"
)

func init() {
	context.SetHandlerName("iris/middleware/monitor.*", "iris.monitor")
}

// Options holds the optional fields for the Monitor structure.
type Options struct {
	// Optional process id, defaults to the current one.
	PID int32 `json:"pid" yaml:"PID"`

	RefreshInterval     time.Duration `json:"refresh_interval" yaml:"RefreshInterval"`
	ViewRefreshInterval time.Duration `json:"view_refresh_interval" yaml:"ViewRefreshInterval"`
	// If more than zero enables line animation. Defaults to zero.
	ViewAnimationInterval time.Duration `json:"view_animation_interval" yaml:"ViewAnimationInterval"`
	// The title of the monitor HTML document.
	ViewTitle string `json:"view_title" yaml:"ViewTitle"`
}

// Monitor tracks and renders the server's process and operating system statistics.
//
// Look its `Stats` and `View` methods.
// Initialize with the `New` package-level function.
type Monitor struct {
	opts   Options
	Holder *StatsHolder

	viewBody []byte
}

// DefaultRefreshInterval is the stats refresh interval used when
// Options.RefreshInterval is not set.
const DefaultRefreshInterval = 2 * time.Second

// New returns a new Monitor.
// Metrics stored through expvar standard package:
// - pid_cpu
// - pid_ram
// - pid_conns
// - os_cpu
// - os_ram
// - os_total_ram
// - os_load_avg
// - os_conns
//
// Check https://github.com/iris-contrib/middleware/tree/master/expmetric
// which can be integrated with datadog or other platforms.
//
// SECURITY WARNING: the stats endpoint MUST NOT be public.
//
// It reports the process id, resident memory, host CPU and RAM, load average
// and connection counts - a precise picture of the machine, useful to anyone
// deciding whether and when to attack it. Guard it exactly like pprof:
//
//	auth := basicauth.Default(map[string]string{"admin": os.Getenv("MONITOR_PASSWORD")})
//	app.Get("/monitor", auth, m.View)
//	app.Get("/monitor/stats", auth, m.Stats)
//
// Note also that importing this package registers its metrics into the default
// expvar map at init, so they are also visible through any mounted
// "/debug/vars" handler.
func New(opts Options) *Monitor {
	if opts.PID == 0 {
		opts.PID = int32(os.Getpid())
	}

	if opts.RefreshInterval <= 0 {
		// Note(@kataras): "2 * opts.RefreshInterval" is 0 when the interval is 0,
		// so monitor.New(monitor.Options{}) - the obvious first call - reached
		// time.NewTicker(0) and panicked at startup.
		opts.RefreshInterval = DefaultRefreshInterval
	}

	if opts.ViewRefreshInterval <= 0 {
		opts.ViewRefreshInterval = opts.RefreshInterval
	}

	viewRefreshIntervalBytes := fmt.Appendf(nil, "%d", opts.ViewRefreshInterval.Milliseconds())
	viewBody := bytes.Replace(defaultViewBody, viewRefreshIntervalTmplVar, viewRefreshIntervalBytes, 1)
	viewAnimationIntervalBytes := fmt.Appendf(nil, "%d", opts.ViewAnimationInterval.Milliseconds())
	viewBody = bytes.Replace(viewBody, viewAnimationIntervalTmplVar, viewAnimationIntervalBytes, 2)
	viewTitleBytes := []byte(opts.ViewTitle)
	viewBody = bytes.Replace(viewBody, viewTitleTmplVar, viewTitleBytes, 2)
	proc, err := process.NewProcess(opts.PID)
	if err != nil {
		panic(err)
	}

	sh := startNewStatsHolder(proc, opts.RefreshInterval)
	m := &Monitor{
		opts:     opts,
		Holder:   sh,
		viewBody: viewBody,
	}

	return m
}

// Stop terminates the retrieve stats loop for
// the process and the operating system statistics.
// No other monitor instance should be initialized after the first Stop call.
func (m *Monitor) Stop() {
	m.Holder.Stop()
}

// Stats sends the stats as json.
func (m *Monitor) Stats(ctx *context.Context) {
	ctx.JSON(m.Holder.GetStats())
}

// View renders a default view for the stats.
func (m *Monitor) View(ctx *context.Context) {
	ctx.SetContentType("text/html")
	ctx.Write(m.viewBody)
}
