// white-box testing
package memstore

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEntryStringDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		def      string
		expected string
	}{
		{"string", "value", "def", "value"},
		{"empty string is returned as is", "", "def", ""},
		{"nil falls back to the default", nil, "def", "def"},
		{"int is formatted", 42, "def", "42"},
		{"bool is formatted", true, "def", "true"},
		{"float is formatted", 3.5, "def", "3.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Entry{Key: "key", ValueRaw: tt.value}
			if got := e.StringDefault(tt.def); got != tt.expected {
				t.Fatalf("expected %q but got %q", tt.expected, got)
			}
		})
	}
}

func TestEntryString(t *testing.T) {
	if got := (Entry{ValueRaw: "value"}).String(); got != "value" {
		t.Fatalf("expected %q but got %q", "value", got)
	}
	// String defaults to the empty string
	if got := (Entry{ValueRaw: nil}).String(); got != "" {
		t.Fatalf("expected an empty string but got %q", got)
	}
}

func TestEntryStringTrim(t *testing.T) {
	if got := (Entry{ValueRaw: "  value  "}).StringTrim(); got != "value" {
		t.Fatalf("expected %q but got %q", "value", got)
	}
	if got := (Entry{ValueRaw: "\tvalue\n"}).StringTrim(); got != "value" {
		t.Fatalf("expected %q but got %q", "value", got)
	}
	if got := (Entry{ValueRaw: nil}).StringTrim(); got != "" {
		t.Fatalf("expected an empty string but got %q", got)
	}
}

func TestEntryIntDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected int
		wantErr  bool
	}{
		{"int", 42, 42, false},
		{"int8", int8(8), 8, false},
		{"int16", int16(16), 16, false},
		{"int32", int32(32), 32, false},
		{"int64", int64(64), 64, false},
		{"uint", uint(1), 1, false},
		{"uint8", uint8(8), 8, false},
		{"uint16", uint16(16), 16, false},
		{"uint32", uint32(32), 32, false},
		{"uint64", uint64(64), 64, false},
		{"numeric string", "7", 7, false},
		{"negative numeric string", "-7", -7, false},
		{"unparsable string", "abc", -1, true},
		{"nil", nil, -1, true},
		{"unsupported type", true, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Entry{Key: "key", ValueRaw: tt.value}
			got, err := e.IntDefault(-1)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %d but got %d", tt.expected, got)
			}
		})
	}
}

func TestEntryBoolDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
		wantErr  bool
	}{
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"string true", "true", true, false},
		{"string 1", "1", true, false},
		{"string false", "false", false, false},
		{"string 0", "0", false, false},
		{"int 1 is true", 1, true, false},
		{"int 0 is false", 0, false, false},
		{"any other int is false", 5, false, false},
		{"unparsable string", "maybe", false, true},
		{"nil", nil, false, true},
		{"unsupported type", 1.5, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Entry{Key: "key", ValueRaw: tt.value}
			got, err := e.BoolDefault(false)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %v but got %v", tt.expected, got)
			}
		})
	}
}

func TestEntryTimeDefault(t *testing.T) {
	now := time.Now()
	def := time.Time{}

	got, err := (Entry{Key: "key", ValueRaw: now}).TimeDefault(def)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if !got.Equal(now) {
		t.Fatalf("expected the stored time but got %v", got)
	}

	// a nil value reports not found
	if _, err = (Entry{Key: "key", ValueRaw: nil}).TimeDefault(def); err == nil {
		t.Fatalf("expected a not found error for a nil value")
	}

	// a value of the wrong type falls back to the default without an error
	got, err = (Entry{Key: "key", ValueRaw: "not a time"}).TimeDefault(def)
	if err != nil {
		t.Fatalf("expected no error for a mistyped value but got %v", err)
	}
	if !got.Equal(def) {
		t.Fatalf("expected the default time but got %v", got)
	}
}

func TestEntryWeekdayDefault(t *testing.T) {
	got, err := (Entry{Key: "key", ValueRaw: time.Friday}).WeekdayDefault(time.Monday)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if got != time.Friday {
		t.Fatalf("expected %v but got %v", time.Friday, got)
	}

	if _, err = (Entry{Key: "key", ValueRaw: nil}).WeekdayDefault(time.Monday); err == nil {
		t.Fatalf("expected a not found error for a nil value")
	}

	// a value of the wrong type falls back to the default without an error
	got, err = (Entry{Key: "key", ValueRaw: "not a weekday"}).WeekdayDefault(time.Monday)
	if err != nil {
		t.Fatalf("expected no error for a mistyped value but got %v", err)
	}
	if got != time.Monday {
		t.Fatalf("expected the default weekday but got %v", got)
	}
}

func TestErrEntryNotFoundError(t *testing.T) {
	err := &ErrEntryNotFound{Key: "key", Kind: reflect.Int, Type: intType}

	expected := "not found: key as int (int)"
	if err.Error() != expected {
		t.Fatalf("expected %q but got %q", expected, err.Error())
	}
}

func TestErrEntryNotFoundAs(t *testing.T) {
	err := &ErrEntryNotFound{Key: "key", Kind: reflect.Int, Type: intType}

	// an empty target matches any key and kind
	if !err.As(&ErrEntryNotFound{}) {
		t.Fatalf("expected an empty target to match")
	}
	if !err.As(&ErrEntryNotFound{Key: "key"}) {
		t.Fatalf("expected a matching key to match")
	}
	if !err.As(&ErrEntryNotFound{Key: "key", Kind: reflect.Int}) {
		t.Fatalf("expected a matching key and kind to match")
	}

	if err.As(&ErrEntryNotFound{Key: "other"}) {
		t.Fatalf("expected a different key not to match")
	}
	if err.As(&ErrEntryNotFound{Key: "key", Kind: reflect.String}) {
		t.Fatalf("expected a different kind not to match")
	}
	// a target of another type never matches
	if err.As(&struct{}{}) {
		t.Fatalf("expected an unrelated target type not to match")
	}
}

func TestErrEntryNotFoundErrorsAs(t *testing.T) {
	_, err := (Entry{Key: "key", ValueRaw: nil}).IntDefault(-1)
	if err == nil {
		t.Fatalf("expected a not found error")
	}

	var target *ErrEntryNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected the error to unwrap to *ErrEntryNotFound")
	}
	if target.Key != "key" {
		t.Fatalf("expected the error to carry the key %q but got %q", "key", target.Key)
	}
	if target.Kind != reflect.Int {
		t.Fatalf("expected the error to carry the int kind but got %v", target.Kind)
	}
}

func TestEntryGetByKindOrNil(t *testing.T) {
	e := Entry{Key: "key", ValueRaw: "value"}
	if got := e.GetByKindOrNil(reflect.String); got != "value" {
		t.Fatalf("expected %q but got %v", "value", got)
	}

	// a string entry cannot be read as an int
	if got := (Entry{Key: "key", ValueRaw: "abc"}).GetByKindOrNil(reflect.Int); got != nil {
		t.Fatalf("expected nil but got %v", got)
	}

	if got := (Entry{Key: "key", ValueRaw: 42}).GetByKindOrNil(reflect.Int); got != 42 {
		t.Fatalf("expected 42 but got %v", got)
	}
}
