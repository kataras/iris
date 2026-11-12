package encoding

import (
	"slices"
	"testing"
	"time"
)

func TestNewMapStringSlice(t *testing.T) {
	m := map[string][]string{
		"key1": {"value1", "value2"},
		"key2": {"value3"},
		"key3": {"42"},
	}
	entries := NewStringSliceEntries(m, ",")
	if entries.GetSeparator() != "," {
		t.Errorf("expected separator to be ',', got %v", entries.GetSeparator())
	}

	if expected, got := 42, entries.GetInt("key3"); expected != got {
		t.Errorf("expected %v, got %v", expected, got)
	}

	if expected, got := "value3", entries.GetString("key2"); expected != got {
		t.Errorf("expected %v, got %v", expected, got)
	}

	if expected, got := "value1,value2", entries.GetString("key1"); expected != got {
		t.Errorf("expected %v, got %v", expected, got)
	}

	sliceEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		// order does not matter
		for _, v := range a {
			found := slices.Contains(b, v)
			if !found {
				return false
			}
		}

		return true
	}

	if expected, got := []string{"value1", "value2"}, entries.GetStringSlice("key1"); !sliceEqual(expected, got) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// GetSimpleDate parses the first non-empty key as a SimpleDate and reports whether
// any key was present; a missing key is not an error, an unparsable value is.
func TestStringSliceEntriesGetSimpleDate(t *testing.T) {
	entries := NewStringSliceEntries(map[string][]string{
		"date":  {"2024-03-05"},
		"from":  {" 2024-1-2 "},
		"blank": {"   "},
		"bad":   {"not-a-date"},
	}, ",")

	date, found, err := entries.GetSimpleDate()
	if err != nil || !found {
		t.Fatalf("expected the default 'date' key to be found and valid, got found=%v err=%v", found, err)
	}
	if got := time.Time(date).Format("2006-01-02"); got != "2024-03-05" {
		t.Fatalf("expected 2024-03-05, got %s", got)
	}

	// Keys are tried in order; a blank value does not count as present.
	date, found, err = entries.GetSimpleDate("missing", "blank", "from")
	if err != nil || !found {
		t.Fatalf("expected 'from' to be used, got found=%v err=%v", found, err)
	}
	if got := time.Time(date).Format("2006-01-02"); got != "2024-01-02" {
		t.Fatalf("expected 2024-01-02, got %s", got)
	}

	if _, found, err = entries.GetSimpleDate("missing"); found || err != nil {
		t.Fatalf("expected a missing key to report found=false with no error, got found=%v err=%v", found, err)
	}

	if _, found, err = entries.GetSimpleDate("bad"); !found || err == nil {
		t.Fatalf("expected an unparsable value to report found=true with an error, got found=%v err=%v", found, err)
	}

	if _, found, err = (StringSliceEntries{}).GetSimpleDate(); found || err != nil {
		t.Fatalf("expected empty entries to report found=false with no error, got found=%v err=%v", found, err)
	}
}
