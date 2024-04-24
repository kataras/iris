package monitor

import (
	"expvar"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Stats holds the process and operating system statistics values.
//
// Note that each statistic has its own expvar metric that you can use
// to render e.g. through statsd. Available values:
// * pid_cpu
// * pid_ram
// * pid_conns
// * os_cpu
// * os_ram
// * os_total_ram
// * os_load_avg
// * os_conns
type Stats struct {
	PIDCPU   float64 `json:"pid_cpu" yaml:"PIDCPU"`
	PIDRAM   uint64  `json:"pid_ram" yaml:"PIDRAM"`
	PIDConns int64   `json:"pid_conns" yaml:"PIDConns"`

	OSCPU      float64 `json:"os_cpu" yaml:"OSCPU"`
	OSRAM      uint64  `json:"os_ram" yaml:"OSRAM"`
	OSTotalRAM uint64  `json:"os_total_ram" yaml:"OSTotalRAM"`
	OSLoadAvg  float64 `json:"os_load_avg" yaml:"OSLoadAvg"`
	OSConns    int64   `json:"os_conns" yaml:"OSConns"`
}

// StatsHolder holds and refreshes the statistics.
type StatsHolder struct {
	proc *process.Process

	stats *Stats
	mu    sync.RWMutex

	started bool
	closeCh chan struct{}
	errCh   chan error
}

func startNewStatsHolder(proc *process.Process, refreshInterval time.Duration) *StatsHolder {
	sh := newStatsHolder(proc)
	sh.start(refreshInterval)
	return sh
}

func newStatsHolder(proc *process.Process) *StatsHolder {
	sh := &StatsHolder{
		proc:    proc,
		stats:   new(Stats),
		closeCh: make(chan struct{}),
		errCh:   make(chan error, 1),
	}

	return sh
}

// Err returns a read-only channel which may be filled with errors
// came from the refresh stats operation.
func (sh *StatsHolder) Err() <-chan error {
	return sh.errCh
}

// Stop terminates the routine retrieves the stats.
// Note that no other monitor can be initialized after Stop.
func (sh *StatsHolder) Stop() {
	if !sh.started {
		return
	}

	sh.closeCh <- struct{}{}
	sh.started = false
}

func (sh *StatsHolder) start(refreshInterval time.Duration) {
	if sh.started {
		return
	}
	sh.started = true

	once.Do(func() {
		go func() {
			ticker := time.NewTicker(refreshInterval)
			defer ticker.Stop()

			for {
				select {
				case <-sh.closeCh:
					// close(sh.errCh)
					return
				case <-ticker.C:
					err := refresh(sh.proc)
					if err != nil {
						// push the error to the channel and continue the execution,
						// the only way to stop it is through its "Stop" method.
						sh.errCh <- err
					}
				}
			}
		}()
	})
}

var (
	once = new(sync.Once)

	metricPidCPU   = expvar.NewFloat("pid_cpu")
	metricPidRAM   = newUint64("pid_ram")
	metricPidConns = expvar.NewInt("pid_conns")

	metricOsCPU      = expvar.NewFloat("os_cpu")
	metricOsRAM      = newUint64("os_ram")
	metricOsTotalRAM = newUint64("os_total_ram")
	metricOsLoadAvg  = expvar.NewFloat("os_load_avg")
	metricOsConns    = expvar.NewInt("os_conns")
)

// refresh updates the process and operating system statistics.
func refresh(proc *process.Process) error {
	// Collect the stats.
	//
	// Process.
	pidCPU, err := proc.CPUPercent()
	if err != nil {
		return err
	}

	pidRAM, err := proc.MemoryInfo()
	if err != nil {
		return err
	}

	pidConns, err := net.ConnectionsPid("tcp", proc.Pid)
	if err != nil {
		return err
	}

	// Operating System.
	osCPU, err := cpu.Percent(0, false)
	if err != nil {
		return err
	}

	osRAM, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	osLoadAvg, err := load.Avg()
	if err != nil {
		return err
	}

	osConns, err := net.Connections("tcp")
	if err != nil {
		return err
	}

	// Update the fields.
	//
	// Process.
	metricPidCPU.Set(pidCPU / 10)
	metricPidRAM.Set(pidRAM.RSS)
	metricPidConns.Set(int64(len(pidConns)))

	// Operating System.
	if len(osCPU) > 0 {
		metricOsCPU.Set(osCPU[0])
	}
	metricOsRAM.Set(osRAM.Used)
	metricOsTotalRAM.Set(osRAM.Total)
	metricOsLoadAvg.Set(osLoadAvg.Load1)
	metricOsConns.Set(int64(len(osConns)))

	return nil
}

// GetStats returns a copy of the latest stats available.
func (sh *StatsHolder) GetStats() Stats {
	sh.mu.Lock()
	statsCopy := Stats{
		PIDCPU:   metricPidCPU.Value(),
		PIDRAM:   metricPidRAM.Value(),
		PIDConns: metricPidConns.Value(),

		OSCPU:      metricOsCPU.Value(),
		OSRAM:      metricOsRAM.Value(),
		OSTotalRAM: metricOsTotalRAM.Value(),
		OSLoadAvg:  metricOsLoadAvg.Value(),
		OSConns:    metricOsConns.Value(),
	}
	sh.mu.Unlock()

	return statsCopy
}
