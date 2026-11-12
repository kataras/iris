package httpcost

import (
	"runtime"
	"time"

	"github.com/kataras/iris/v14/rest/types"

	"github.com/shirou/gopsutil/v4/cpu"
)

// Compute calculates and returns the cost of the given job.
//
// It uses the DefaultOptions to initialize a new Computer, starts
// a running process measurement, executes the provided job function,
// and then stops the measurement. It returns the total cost as a float64.
func Compute(job func()) float64 {
	computer := NewComputer(DefaultOptions)
	proc := computer.Start()
	// defer func() {
	// 	if err := recover(); err != nil {
	// 	}
	// }()
	// No, if we have a panic don't calculate the cost, it is on us.
	job()
	completed := proc.Stop(0)
	return completed.Cost
}

// Options contains the cost configuration for time, memory, CPU, and network usage.
//
// CostPerSecond sets the cost per elapsed second. CostPerMB sets the cost per MB
// of used (RAM/Network) memory. CostPerCPU sets the cost per CPU-second.
//
// A zero field disables that cost; there are no per-field defaults. DefaultOptions
// carries the usual values (0.01, 0.001 and 0.02). At least one cost must be above
// zero, or New and NewComputer panic.
type Options struct {
	// The cost per second of time to execute the job.
	// Zero disables the time duration cost. DefaultOptions sets 0.01.
	CostPerSecond float64 `json:"cost_per_second" yaml:"CostPerSecond" toml:"CostPerSecond"`
	// The cost per MB of memory used (RAM (and Network)).
	// Zero disables the memory cost. DefaultOptions sets 0.001.
	CostPerMB float64 `json:"cost_per_mb" yaml:"CostPerMB" toml:"CostPerMB"`
	// The cost per second of CPU time.
	// Zero disables the CPU time cost. DefaultOptions sets 0.02.
	CostPerCPU float64 `json:"cost_per_cpu" yaml:"CostPerCPU" toml:"CostPerCPU"`
}

// calculateCost returns the total cost based on provided usage metrics:
//
// durationSec is the total duration in seconds,
// memUsed is the used memory in bytes,
// netBytes is the used network in bytes,
// cpuTime is the consumed CPU time in seconds.
func (c *Options) calculateCost(durationSec, memUsed, netBytes, cpuTime float64) float64 {
	var (
		costPerSecond = c.CostPerSecond
		costPerMB     = c.CostPerMB
		costPerCPU    = c.CostPerCPU
	)

	timeCost := costPerSecond * durationSec
	memCost := costPerMB * memUsed / (1024 * 1024)
	netCost := costPerMB * netBytes / (1024 * 1024)
	cpuCost := costPerCPU * cpuTime

	return timeCost + memCost + netCost + cpuCost
}

// DefaultOptions contains the default cost configuration for the package:
//
//   - CostPerSecond: 0.01
//   - CostPerMB: 0.001
//   - CostPerCPU: 0.02
var DefaultOptions = Options{
	CostPerSecond: 0.01,
	CostPerMB:     0.001,
	CostPerCPU:    0.02,
}

// Computer is responsible for calculating the cost of a job.
//
// It holds the Options and uses them to start and stop RunningProcess measurements.
type Computer struct {
	opts *Options
}

// NewComputer returns a new Computer instance, or panics
// if all Options are disabled (i.e. zero cost metrics).
//
// Usage:
//
//	computer := httpcost.NewComputer(httpcost.DefaultOptions)
func NewComputer(opts Options) *Computer {
	if opts.CostPerSecond <= 0 && opts.CostPerMB <= 0 && opts.CostPerCPU <= 0 {
		panic("httpcost: at least one cost must be enabled")
	}

	return &Computer{opts: &opts}
}

// RunningProcess represents an ongoing measurement of time, CPU, and memory usage.
type RunningProcess struct {
	Options *Options

	StartTime        time.Time // timestamp of the beginning.
	StartCpuUsage    float64   // seconds to perform.
	StartMemoryUsage uint64    // bytes.
}

// Start initializes a new RunningProcess and records the current time,
// CPU usage, and memory usage if the respective cost metrics are enabled.
func (c *Computer) Start() *RunningProcess {
	proc := new(RunningProcess)
	proc.Options = c.opts

	// Set the StartTime.
	if c.opts.CostPerSecond > 0 {
		proc.StartTime = time.Now()
	}

	// Set StartCPUTimes.
	if c.opts.CostPerCPU > 0 {
		if cpuTimes, _ := cpu.Times(false); len(cpuTimes) > 0 {
			proc.StartCpuUsage = cpuTimes[0].User + cpuTimes[0].System // seconds.
		}
	}

	// Set the StartMemAlloc.
	if c.opts.CostPerMB > 0 {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		proc.StartMemoryUsage = memStats.Alloc
	}

	return proc
}

// Metric holds the final measurement data and total cost.
type Metric struct {
	StartTime   types.ISO8601 `pg:"type=timestamp" json:"start_time"`
	Duration    time.Duration `pg:"type=interval" json:"duration"`
	MemoryUsed  uint64        `pg:"type=int8" json:"memory_used"`  // bytes.
	NetworkUsed uint64        `pg:"type=int8" json:"network_used"` // bytes.
	CpuTime     float64       `pg:"type=decimal" json:"cpu_time"`  // seconds.
	Cost        float64       `pg:"type=decimal" json:"cost"`
}

// Stop finishes the measurement and calculates the cost.
//
// netBytes is the total number of bytes written to the network.
func (proc *RunningProcess) Stop(netBytes int) *Metric {
	metric := new(Metric)
	metric.StartTime = types.ISO8601(proc.StartTime)
	if netBytes > 0 {
		// A response without a body reports context.NoWritten (-1);
		// converting that to uint64 would wrap around and inflate the cost.
		metric.NetworkUsed = uint64(netBytes)
	}

	if proc.Options.CostPerSecond > 0 {
		metric.Duration = time.Since(proc.StartTime)
	}

	if proc.Options.CostPerCPU > 0 {
		if afterCpuTimes, _ := cpu.Times(false); len(afterCpuTimes) > 0 {
			endCpuTimes := afterCpuTimes[0].User + afterCpuTimes[0].System

			metric.CpuTime = endCpuTimes - proc.StartCpuUsage
		}
	}

	if proc.Options.CostPerMB > 0 {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		metric.MemoryUsed = memStats.Alloc - proc.StartMemoryUsage
	}

	metric.Cost = proc.Options.calculateCost(
		metric.Duration.Seconds(),
		float64(metric.MemoryUsed),
		float64(metric.NetworkUsed),
		metric.CpuTime,
	)

	return metric
}
