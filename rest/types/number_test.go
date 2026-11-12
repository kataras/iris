package types_test

import (
	jsonv1 "encoding/json"
	json "encoding/json/v2"
	"math"
	"testing"

	"github.com/kataras/iris/v14/rest/types"
)

func TestNumberMarshal(t *testing.T) {
	tests := []struct {
		value    types.Number
		expected string
	}{
		{types.Number(0), "0"},
		{types.Number(1.5), "1.5"},
		{types.Number(-42), "-42"},
		{types.Number(1e21), "1e+21"}, // the two encoders must not disagree here.
		{types.Number(math.NaN()), "null"},
		{types.Number(math.Inf(1)), "null"},
		{types.Number(math.Inf(-1)), "null"},
		{types.NullNumber(), "null"},
	}

	for _, tt := range tests {
		// v2, through MarshalJSONTo.
		got, err := json.Marshal(tt.value)
		if err != nil {
			t.Fatalf("%v: %v", float64(tt.value), err)
		}
		if string(got) != tt.expected {
			t.Fatalf("v2: %v: expected: %s but got: %s", float64(tt.value), tt.expected, got)
		}

		// v1, through MarshalJSON.
		got, err = jsonv1.Marshal(tt.value)
		if err != nil {
			t.Fatalf("%v: %v", float64(tt.value), err)
		}
		if string(got) != tt.expected {
			t.Fatalf("v1: %v: expected: %s but got: %s", float64(tt.value), tt.expected, got)
		}
	}
}

func TestNumberUnmarshal(t *testing.T) {
	tests := []struct {
		data     string
		expected float64
		isNull   bool
	}{
		{"0", 0, false},
		{"1.5", 1.5, false},
		{"-42", -42, false},
		{"null", 0, true},
	}

	for _, tt := range tests {
		var v2 types.Number
		if err := json.Unmarshal([]byte(tt.data), &v2); err != nil {
			t.Fatalf("v2: %s: %v", tt.data, err)
		}

		var v1 types.Number
		if err := jsonv1.Unmarshal([]byte(tt.data), &v1); err != nil {
			t.Fatalf("v1: %s: %v", tt.data, err)
		}

		for name, got := range map[string]types.Number{"v2": v2, "v1": v1} {
			if tt.isNull {
				if !got.IsNull() {
					t.Fatalf("%s: %s: expected a null number but got: %v", name, tt.data, float64(got))
				}
				continue
			}

			if float64(got) != tt.expected {
				t.Fatalf("%s: %s: expected: %v but got: %v", name, tt.data, tt.expected, float64(got))
			}
		}
	}
}

func TestNumberScanValue(t *testing.T) {
	tests := []struct {
		src      any
		expected float64
		isNull   bool
	}{
		{float64(1.5), 1.5, false},
		{float32(2), 2, false},
		{int64(3), 3, false},
		{"4.25", 4.25, false},
		{[]byte("5.5"), 5.5, false},
		{nil, 0, true},
	}

	for _, tt := range tests {
		var n types.Number
		if err := n.Scan(tt.src); err != nil {
			t.Fatalf("%v: %v", tt.src, err)
		}

		if tt.isNull {
			if !n.IsNull() {
				t.Fatalf("%v: expected a null number but got: %v", tt.src, float64(n))
			}
		} else if float64(n) != tt.expected {
			t.Fatalf("%v: expected: %v but got: %v", tt.src, tt.expected, float64(n))
		}

		value, err := n.Value()
		if err != nil {
			t.Fatalf("%v: value: %v", tt.src, err)
		}

		if tt.isNull {
			if value != nil {
				t.Fatalf("%v: expected a NULL driver value but got: %v", tt.src, value)
			}
			continue
		}

		if value != tt.expected {
			t.Fatalf("%v: expected the driver value: %v but got: %v", tt.src, tt.expected, value)
		}
	}

	// A number which is not a number at all is a scan error, not a silent zero.
	var n types.Number
	if err := n.Scan("not a number"); err == nil {
		t.Fatal("expected a scan error for a non-numeric text")
	}
}

// A quoted number is rejected by both encoders: a payload must not decode
// under one and fail under the other.
func TestNumberUnmarshalRejectsQuoted(t *testing.T) {
	var v2 types.Number
	if err := json.Unmarshal([]byte(`"1.5"`), &v2); err == nil {
		t.Fatal("v2: expected a quoted number to be rejected")
	}

	var v1 types.Number
	if err := jsonv1.Unmarshal([]byte(`"1.5"`), &v1); err == nil {
		t.Fatal("v1: expected a quoted number to be rejected")
	}
}

// NULL and NaN are the same value on both sides of the driver.
func TestNumberNullRoundTrip(t *testing.T) {
	n := types.NullNumber()

	value, err := n.Value()
	if err != nil {
		t.Fatal(err)
	}

	var back types.Number
	if err = back.Scan(value); err != nil {
		t.Fatal(err)
	}

	if !back.IsNull() {
		t.Fatalf("expected a null number but got: %v", float64(back))
	}
}
