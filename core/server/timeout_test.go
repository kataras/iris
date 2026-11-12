package server

import (
	"testing"
	"time"
)

// TestCalculateTimeouts checks the calculateTimeouts function with various input values, including edge cases.
func TestCalculateTimeouts(t *testing.T) {
	tests := []struct {
		name                  string
		totalLifecycleTimeout time.Duration
		wantReadHeaderTimeout time.Duration
		wantReadTimeout       time.Duration
		wantWriteTimeout      time.Duration
		wantIdleTimeout       time.Duration
	}{
		{"ZeroTimeout", 0, 0, 0, 0, 0},
		{"NegativeTimeout", -10 * time.Second, 0, 0, 0, 0},
		{"MinimumThreshold", 10 * time.Second, 2500 * time.Millisecond, 5 * time.Second, 5 * time.Second, 5 * time.Second},
		{"EvenDivision", 30 * time.Second, 7500 * time.Millisecond, 15 * time.Second, 7500 * time.Millisecond, 11250 * time.Millisecond},
		{"LargeTimeout", 120 * time.Second, 30 * time.Second, 60 * time.Second, 30 * time.Second, 45 * time.Second},
		// Edge cases.
		// Tests the function when the total lifecycle timeout is just above the minimum threshold.
		{"JustAboveMinThreshold", 11 * time.Second, 2750 * time.Millisecond, 5500 * time.Millisecond, 5 * time.Second, 5 * time.Second},
		// Ensures that the idle timeout does not exceed the read timeout even when the calculated percentage would make it larger.
		{"MaxIdleTimeoutExceedsReadTimeout", 20 * time.Second, 5000 * time.Millisecond, 10 * time.Second, 5 * time.Second, 7500 * time.Millisecond},
		// Checks that the read header timeout is correctly set to half of the read timeout.
		{"ReadHeaderTimeoutHalfOfReadTimeout", 40 * time.Second, 10 * time.Second, 20 * time.Second, 10 * time.Second, 15 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReadHeaderTimeout, gotReadTimeout, gotWriteTimeout, gotIdleTimeout := CalculateTimeouts(tt.totalLifecycleTimeout)
			if gotReadHeaderTimeout != tt.wantReadHeaderTimeout {
				t.Errorf("calculateTimeouts() gotReadHeaderTimeout = %v, want %v", gotReadHeaderTimeout, tt.wantReadHeaderTimeout)
			}
			if gotReadTimeout != tt.wantReadTimeout {
				t.Errorf("calculateTimeouts() gotReadTimeout = %v, want %v", gotReadTimeout, tt.wantReadTimeout)
			}
			if gotWriteTimeout != tt.wantWriteTimeout {
				t.Errorf("calculateTimeouts() gotWriteTimeout = %v, want %v", gotWriteTimeout, tt.wantWriteTimeout)
			}
			if gotIdleTimeout != tt.wantIdleTimeout {
				t.Errorf("calculateTimeouts() gotIdleTimeout = %v, want %v", gotIdleTimeout, tt.wantIdleTimeout)
			}
		})
	}
}
