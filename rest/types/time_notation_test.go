package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseTimeNotationDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected TimeNotationDuration
	}{
		{"01:30:00", TimeNotationDuration(1*time.Hour + 30*time.Minute)},
		{"00:45:30", TimeNotationDuration(45*time.Minute + 30*time.Second)},
		{"00:00:10", TimeNotationDuration(10 * time.Second)},
	}

	for _, test := range tests {
		result, err := ParseTimeNotationDuration(test.input)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
		if result != test.expected {
			t.Fatalf("expected %v, but got %v", test.expected, result)
		}
	}
}

func TestTimeNotationDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		input    TimeNotationDuration
		expected string
	}{
		{TimeNotationDuration(1*time.Hour + 30*time.Minute), `"01:30:00"`},
		{TimeNotationDuration(45*time.Minute + 30*time.Second), `"00:45:30"`},
		{TimeNotationDuration(10 * time.Second), `"00:00:10"`},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.input)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
		if string(result) != test.expected {
			t.Fatalf("expected %v, but got %v", test.expected, string(result))
		}
	}
}

func TestTimeNotationDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected TimeNotationDuration
	}{
		{`"01:30:00"`, TimeNotationDuration(1*time.Hour + 30*time.Minute)},
		{`"00:45:30"`, TimeNotationDuration(45*time.Minute + 30*time.Second)},
		{`"00:00:10"`, TimeNotationDuration(10 * time.Second)},
	}

	for _, test := range tests {
		var result TimeNotationDuration
		err := json.Unmarshal([]byte(test.input), &result)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
		if result != test.expected {
			t.Fatalf("expected %v, but got %v", test.expected, result)
		}
	}
}

func TestTimeDurationToDuration(t *testing.T) {
	tests := []struct {
		input    TimeNotationDuration
		expected time.Duration
	}{
		{TimeNotationDuration(1*time.Hour + 30*time.Minute), 1*time.Hour + 30*time.Minute},
		{TimeNotationDuration(45*time.Minute + 30*time.Second), 45*time.Minute + 30*time.Second},
		{TimeNotationDuration(10 * time.Second), 10 * time.Second},
	}

	for _, test := range tests {
		result := test.input.ToDuration()
		if result != test.expected {
			t.Fatalf("expected %v, but got %v", test.expected, result)
		}
	}
}

func TestTimeNotationDurationSetDuration(t *testing.T) {
	var d TimeNotationDuration
	d.SetDuration(1*time.Hour + 30*time.Minute)
	expected := TimeNotationDuration(1*time.Hour + 30*time.Minute)
	if d != expected {
		t.Fatalf("expected %v, but got %v", expected, d)
	}
}
