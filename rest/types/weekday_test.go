package types_test

import (
	jsonv1 "encoding/json"
	json "encoding/json/v2"
	"testing"
	"time"

	"github.com/kataras/iris/v14/rest/types"
)

func TestParseWeekday(t *testing.T) {
	tests := []struct {
		input    string
		expected types.Weekday
	}{
		{"monday", types.Weekday(time.Monday)},
		{"Monday", types.Weekday(time.Monday)},
		{"MONDAY", types.Weekday(time.Monday)},
		{"mon", types.Weekday(time.Monday)},
		{"MON", types.Weekday(time.Monday)},
		{"Sun", types.Weekday(time.Sunday)},
		{" thursday ", types.Weekday(time.Thursday)},
		{"saturday", types.Weekday(time.Saturday)},
	}

	for _, tt := range tests {
		got, err := types.ParseWeekday(tt.input)
		if err != nil {
			t.Fatalf("%q: %v", tt.input, err)
		}

		if got != tt.expected {
			t.Fatalf("%q: expected: %s but got: %s", tt.input, tt.expected, got)
		}
	}

	for _, input := range []string{"", "someday", "mo", "8"} {
		if _, err := types.ParseWeekday(input); err == nil {
			t.Fatalf("%q: expected an error", input)
		}
	}
}

func TestWeekdayString(t *testing.T) {
	if expected, got := "monday", types.Weekday(time.Monday).String(); expected != got {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}

	if expected, got := "sunday", types.Weekday(time.Sunday).String(); expected != got {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}

	if types.Weekday(9).IsValid() {
		t.Fatal("expected an out of range weekday to be invalid")
	}
}

// The named constants, Std and WeekdayOf agree with the standard library in
// both directions.
func TestWeekdayConstantsAndConversions(t *testing.T) {
	pairs := []struct {
		day types.Weekday
		std time.Weekday
	}{
		{types.Sunday, time.Sunday},
		{types.Monday, time.Monday},
		{types.Tuesday, time.Tuesday},
		{types.Wednesday, time.Wednesday},
		{types.Thursday, time.Thursday},
		{types.Friday, time.Friday},
		{types.Saturday, time.Saturday},
	}

	for _, pair := range pairs {
		if got := pair.day.Std(); got != pair.std {
			t.Fatalf("Std: expected: %s but got: %s", pair.std, got)
		}

		if got := types.WeekdayOf(pair.std); got != pair.day {
			t.Fatalf("WeekdayOf: expected: %s but got: %s", pair.day, got)
		}
	}
}

func TestWeekdayMarshal(t *testing.T) {
	for _, tt := range []struct {
		value    types.Weekday
		expected string
	}{
		{types.Weekday(time.Monday), `"monday"`},
		{types.Weekday(time.Sunday), `"sunday"`},
	} {
		got, err := json.Marshal(tt.value)
		if err != nil {
			t.Fatalf("v2: %s: %v", tt.value, err)
		}
		if string(got) != tt.expected {
			t.Fatalf("v2: expected: %s but got: %s", tt.expected, got)
		}

		got, err = jsonv1.Marshal(tt.value)
		if err != nil {
			t.Fatalf("v1: %s: %v", tt.value, err)
		}
		if string(got) != tt.expected {
			t.Fatalf("v1: expected: %s but got: %s", tt.expected, got)
		}
	}

	if _, err := json.Marshal(types.Weekday(9)); err == nil {
		t.Fatal("expected an out of range weekday to fail to marshal")
	}
}

func TestWeekdayUnmarshal(t *testing.T) {
	for _, tt := range []struct {
		data     string
		expected types.Weekday
	}{
		{`"Monday"`, types.Weekday(time.Monday)},
		{`"MON"`, types.Weekday(time.Monday)},
		{`"friday"`, types.Weekday(time.Friday)},
	} {
		var v2 types.Weekday
		if err := json.Unmarshal([]byte(tt.data), &v2); err != nil {
			t.Fatalf("v2: %s: %v", tt.data, err)
		}
		if v2 != tt.expected {
			t.Fatalf("v2: %s: expected: %s but got: %s", tt.data, tt.expected, v2)
		}

		var v1 types.Weekday
		if err := jsonv1.Unmarshal([]byte(tt.data), &v1); err != nil {
			t.Fatalf("v1: %s: %v", tt.data, err)
		}
		if v1 != tt.expected {
			t.Fatalf("v1: %s: expected: %s but got: %s", tt.data, tt.expected, v1)
		}
	}

	// null and an empty text leave the value alone, they are not an error.
	for _, data := range []string{"null", `""`} {
		v := types.Weekday(time.Friday)
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			t.Fatalf("v2: %s: %v", data, err)
		}
		if v != types.Weekday(time.Friday) {
			t.Fatalf("v2: %s: expected the value to be left alone but got: %s", data, v)
		}
	}

	var invalid types.Weekday
	if err := json.Unmarshal([]byte(`"someday"`), &invalid); err == nil {
		t.Fatal("expected an unknown weekday to fail to unmarshal")
	}
}

func TestWeekdayScanValue(t *testing.T) {
	for _, tt := range []struct {
		src      any
		expected types.Weekday
	}{
		{"monday", types.Weekday(time.Monday)},
		{"MON", types.Weekday(time.Monday)},
		{[]byte("tuesday"), types.Weekday(time.Tuesday)},
		{int64(3), types.Weekday(time.Wednesday)},
	} {
		var d types.Weekday
		if err := d.Scan(tt.src); err != nil {
			t.Fatalf("%v: %v", tt.src, err)
		}

		if d != tt.expected {
			t.Fatalf("%v: expected: %s but got: %s", tt.src, tt.expected, d)
		}

		value, err := d.Value()
		if err != nil {
			t.Fatalf("%v: value: %v", tt.src, err)
		}

		if value != tt.expected.String() {
			t.Fatalf("%v: expected the driver value: %s but got: %v", tt.src, tt.expected, value)
		}
	}

	// A NULL column leaves the value alone.
	d := types.Weekday(time.Friday)
	if err := d.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if d != types.Weekday(time.Friday) {
		t.Fatalf("expected the value to be left alone but got: %s", d)
	}

	var invalid types.Weekday
	if err := invalid.Scan("someday"); err == nil {
		t.Fatal("expected a scan error for an unknown weekday")
	}
}
