package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONSimpleDate(t *testing.T) {
	data := `{"start": "2021-08-20", "end": "2021-12-01", "nothing": null, "empty": ""}`
	v := struct {
		Start   SimpleDate `json:"start"`
		End     SimpleDate `json:"end"`
		Nothing SimpleDate `json:"nothing"`
		Empty   SimpleDate `json:"empty"`
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

	if expected, got := time.Date(2021, time.August, 20, 0, 0, 0, 0, loc), v.Start.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2021, time.December, 1, 0, 0, 0, 0, loc), v.End.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}
}

func TestSimpleDateAterBefore(t *testing.T) {
	d1, d2 := SimpleDateFromTime(time.Now()), SimpleDateFromTime(time.Now().AddDate(0, 0, 1))

	if d1.After(d2) {
		t.Fatalf("[after] expected d1 to be before d2")
	}

	if !d1.Before(d2) {
		t.Fatalf("[before] expected d1 to be before d2")
	}

	if d2.Before(d1) {
		t.Fatalf("[after] expected d2 to be after d1")
	}

	if !d2.After(d1) {
		t.Fatalf("[after] expected d2 to be after d1")
	}
}

func TestCountPastDays(t *testing.T) {
	tests := []struct {
		AfterDate        SimpleDate
		BeforeDate       SimpleDate
		ExpectedPastDays int
	}{
		{
			AfterDate:        mustParseSimpleDate("2023-01-01"),
			BeforeDate:       mustParseSimpleDate("2022-12-31"),
			ExpectedPastDays: 1,
		},
		{
			AfterDate:        mustParseSimpleDate("2023-01-01"),
			BeforeDate:       mustParseSimpleDate("2022-01-01"),
			ExpectedPastDays: 365,
		},
	}

	for i, tt := range tests {
		if expected, got := tt.ExpectedPastDays, tt.AfterDate.CountPastDays(tt.BeforeDate); expected != got {
			t.Fatalf("[%d] expected past days count: %d but got: %d", i, expected, got)
		}
	}
}

func mustParseSimpleDate(s string) SimpleDate {
	d, err := ParseSimpleDate(s)
	if err != nil {
		panic(err)
	}

	return d
}
