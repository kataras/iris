package httpcost

import (
	"testing"
)

// A handler that writes no body leaves the response writer reporting
// context.NoWritten (-1). Converting that straight to uint64 wraps around
// to 18446744073709551615 and inflates the computed cost.
func TestStopClampsNegativeNetworkUsage(t *testing.T) {
	computer := NewComputer(DefaultOptions)

	if metric := computer.Start().Stop(-1); metric.NetworkUsed != 0 {
		t.Fatalf("expected no network usage for an unwritten body, got %d",
			metric.NetworkUsed)
	}

	if metric := computer.Start().Stop(512); metric.NetworkUsed != 512 {
		t.Fatalf("expected 512 bytes of network usage, got %d", metric.NetworkUsed)
	}
}
