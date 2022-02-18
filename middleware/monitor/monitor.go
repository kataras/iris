package monitor

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/shirou/gopsutil/v3/process"
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
func New(opts Options) *Monitor {
	if opts.PID == 0 {
		opts.PID = int32(os.Getpid())
	}

	if opts.RefreshInterval <= 0 {
		opts.RefreshInterval = 2 * opts.RefreshInterval
	}

	if opts.ViewRefreshInterval <= 0 {
		opts.ViewRefreshInterval = opts.RefreshInterval
	}

	viewRefreshIntervalBytes := []byte(fmt.Sprintf("%d", opts.ViewRefreshInterval.Milliseconds()))
	viewBody := bytes.Replace(defaultViewBody, viewRefreshIntervalTmplVar, viewRefreshIntervalBytes, 1)
	viewAnimationIntervalBytes := []byte(fmt.Sprintf("%d", opts.ViewAnimationInterval.Milliseconds()))
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
	ctx.ContentType("text/html")
	ctx.Write(m.viewBody)
}
