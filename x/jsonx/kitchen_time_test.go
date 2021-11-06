package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONKitckenTime(t *testing.T) {
	data := `{"start": "8:33 AM", "end": "3:04 PM", "nothing": null, "empty": ""}`
	v := struct {
		Start   KitckenTime `json:"start"`
		End     KitckenTime `json:"end"`
		Nothing KitckenTime `json:"nothing"`
		Empty   KitckenTime `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
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
