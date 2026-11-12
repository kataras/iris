package types

import (
	"testing"
)

// Scanning a database TIME value must keep the minutes: a shadowed
// variable used to make every scanned value land on the exact hour.
func TestKitchenTimeScanKeepsMinutes(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		// String renders the Kitchen format, so the hour is not padded.
		{"13:45:00", "1:45 PM"},
		{"10:30:00", "10:30 AM"},
		{"00:05:00", "12:05 AM"},
		{"09:00:00", "9:00 AM"},
		{"23:59:00", "11:59 PM"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var kt KitchenTime
			if err := kt.Scan(tt.in); err != nil {
				t.Fatalf("scan %q: %v", tt.in, err)
			}

			if got := kt.String(); got != tt.expected {
				t.Fatalf("scan %q: expected %q but got %q", tt.in, tt.expected, got)
			}
		})
	}
}
