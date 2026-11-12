package types

import (
	"testing"
	"time"
)

// Text-mode database drivers hand column values as []byte,
// Scan must accept them exactly as it accepts strings.

func TestDayTimeScanBytes(t *testing.T) {
	var v DayTime
	if err := v.Scan([]byte("08:33:00")); err != nil {
		t.Fatal(err)
	}

	if expected, got := "08:33:00", v.String(); expected != got {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}

func TestSimpleDateScanBytes(t *testing.T) {
	var v SimpleDate
	if err := v.Scan([]byte("2024-01-02")); err != nil {
		t.Fatal(err)
	}

	if expected, got := "2024-01-02", v.String(); expected != got {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}

func TestKitchenTimeScanBytes(t *testing.T) {
	var v KitchenTime
	if err := v.Scan([]byte("13:45:00")); err != nil {
		t.Fatal(err)
	}

	if expected, got := "1:45 PM", v.String(); expected != got {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}

func TestKitchenTimeScanInt64Milliseconds(t *testing.T) {
	// The int64 millisecond branch must keep working as before
	// (it is deliberately ordered before the []byte one).
	restore := time.Local
	time.Local = time.UTC // the int64 branch reads through time.Unix which is local-time based.
	defer func() { time.Local = restore }()

	var v KitchenTime
	if err := v.Scan(int64(13*3600+45*60) * 1000); err != nil {
		t.Fatal(err)
	}

	if expected, got := "1:45 PM", v.String(); expected != got {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}
