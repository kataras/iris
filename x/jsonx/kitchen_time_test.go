package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONKitchenTime(t *testing.T) {
	tests := []struct {
		rawData string
	}{
		{
			rawData: `{"start": "8:33 AM", "end": "3:04 PM", "nothing": null, "empty": ""}`,
		},
		{
			rawData: `{"start": "08:33 AM", "end": "03:04 PM", "nothing": null, "empty": ""}`,
		},
		{
			rawData: `{"start": "08:33:00.000000 AM", "end": "03:04 PM", "nothing": null, "empty": ""}`,
		},
	}

	for _, tt := range tests {
		v := struct {
			Start   KitchenTime `json:"start"`
			End     KitchenTime `json:"end"`
			Nothing KitchenTime `json:"nothing"`
			Empty   KitchenTime `json:"empty"`
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

		if expected, got := time.Date(0, time.January, 1, 8, 33, 0, 0, loc), v.Start.Value(); expected != got {
			t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
		}

		if expected, got := time.Date(0, time.January, 1, 15, 4, 0, 0, loc), v.End.Value(); expected != got {
			t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
		}
	}
}
