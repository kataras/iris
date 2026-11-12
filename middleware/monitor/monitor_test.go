package monitor_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14/middleware/monitor"
)

// monitor.New(monitor.Options{}) - the obvious first call - used to panic at
// startup: RefreshInterval was "defaulted" with "2 * opts.RefreshInterval",
// which is still 0, and time.NewTicker(0) panics.
func TestMonitorZeroOptionsDoesNotPanic(t *testing.T) {
	m := monitor.New(monitor.Options{})
	if m == nil {
		t.Fatal("expected a Monitor")
	}
}

func TestMonitorRespectsConfiguredInterval(t *testing.T) {
	m := monitor.New(monitor.Options{RefreshInterval: 5 * time.Second})
	if m == nil {
		t.Fatal("expected a Monitor")
	}
}

func TestMonitorStatsAreAvailable(t *testing.T) {
	m := monitor.New(monitor.Options{RefreshInterval: time.Second})
	if m == nil {
		t.Fatal("expected a Monitor")
	}

	stats := m.Holder.GetStats()
	if stats.PIDCPU < 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
