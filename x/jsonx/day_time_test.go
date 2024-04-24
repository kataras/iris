package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDayTime(t *testing.T) {
	tests := []struct {
		rawData string
	}{
		{
			rawData: `{"start": "8:33:00", "end": "15:00:42", "nothing": null, "empty": ""}`,
		},
		{
			rawData: `{"start": "8:33:00", "end": "15:00:42", "nothing": null, "empty": ""}`,
		},
	}

	for _, tt := range tests {
		v := struct {
			Start   DayTime `json:"start"`
			End     DayTime `json:"end"`
			Nothing DayTime `json:"nothing"`
			Empty   DayTime `json:"empty"`
		}{}

		err := json.Unmarshal([]byte(tt.rawData), &v)
		if err != nil {
			t.Fatal(err)
		}

		if !v.Nothing.IsZero() {
			t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
		}

		if !v.Empty.IsZero() {
			t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
		}

		loc := time.UTC

		if expected, got := time.Date(0, time.January, 1, 8, 33, 0, 0, loc), v.Start.ToTime(); expected != got {
			t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
		}

		if expected, got := time.Date(0, time.January, 1, 15, 0, 42, 0, loc), v.End.ToTime(); expected != got {
			t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
		}
	}
}
