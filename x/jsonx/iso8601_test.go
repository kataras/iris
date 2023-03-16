package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestISO8601(t *testing.T) {
	data := `{"start": "2021-08-20T10:05:01", "end": "2021-12-01T17:05:06", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
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

	if expected, got := time.Date(2021, time.August, 20, 10, 5, 1, 0, loc), v.Start.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2021, time.December, 1, 17, 5, 6, 0, loc), v.End.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}
}

func TestISO8601WithZoneUTCOffset(t *testing.T) {
	data := `{"start": "2022-08-10T03:21:00.000000+03:00", "end": "2022-08-10T09:49:00.000000+03:00", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// t.Logf("Start: %s, location: %s\n", v.Start.String(), v.Start.ToTime().Location().String())

	if !v.Nothing.IsZero() {
		t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
	}

	if !v.Empty.IsZero() {
		t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
	}

	loc := time.FixedZone("EEST", 10800)

	if expected, got := time.Date(2022, time.August, 10, 3, 21, 0, 0, loc).String(), v.Start.ToTime().String(); expected != got {
		t.Fatalf("expected 'start' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 9, 49, 0, 0, loc).String(), v.End.ToTime().String(); expected != got {
		t.Fatalf("expected 'end' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 3, 21, 0, 0, loc), v.Start.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 9, 49, 0, 0, loc), v.End.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'end' to be: %v but got: %v", expected, got)
	}
}

func TestISO8601WithZoneUTCOffsetWithoutMilliseconds(t *testing.T) {
	data := `{"start": "2023-02-04T09:48:14+00:00", "end": "2023-02-05T00:03:16+00:00", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// t.Logf("Start: %s, location: %s\n", v.Start.String(), v.Start.ToTime().Location().String())

	if !v.Nothing.IsZero() {
		t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
	}

	if !v.Empty.IsZero() {
		t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
	}

	loc := time.FixedZone("UTC", 0)

	if expected, got := time.Date(2023, time.February, 04, 9, 48, 14, 0, loc).String(), v.Start.ToTime().String(); expected != got {
		t.Fatalf("expected 'start' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 05, 0, 3, 16, 0, loc).String(), v.End.ToTime().String(); expected != got {
		t.Fatalf("expected 'end' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 04, 9, 48, 14, 0, loc), v.Start.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 05, 0, 3, 16, 0, loc), v.End.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'end' to be: %v but got: %v", expected, got)
	}
}
