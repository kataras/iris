package signalhandler

// NewProcess and OnSignal expose newProcess and onSignal to the external test
// package, which runs a real application through the path of a signal without
// sending one: see run_test.go.
var (
	NewProcess = newProcess
	OnSignal   = onSignal
)
