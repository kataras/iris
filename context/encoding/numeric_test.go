package encoding

import (
	"math"
	"testing"
)

func entryOf(value any) *Entry[any] {
	return &Entry[any]{Key: "k", Value: value}
}

// The case this exists for: ctx.Values().Set("id", 42) stores an untyped 42 as
// an int, and reading it back as a uint used to return zero.
func TestEntryNumericCrossType(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"int", 42},
		{"int8", int8(42)},
		{"int16", int16(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},
		{"uint", uint(42)},
		{"uint8", uint8(42)},
		{"uint16", uint16(42)},
		{"uint32", uint32(42)},
		{"uint64", uint64(42)},
		{"float32", float32(42)},
		{"float64 as JSON decodes every number", float64(42)},
		{"string", "42"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := entryOf(tc.value)

			if got := e.Int(); got != 42 {
				t.Errorf("Int: got %d, want 42", got)
			}
			if got := e.Int64(); got != 42 {
				t.Errorf("Int64: got %d, want 42", got)
			}
			if got := e.Uint(); got != 42 {
				t.Errorf("Uint: got %d, want 42", got)
			}
			if got := e.Uint64(); got != 42 {
				t.Errorf("Uint64: got %d, want 42", got)
			}
			if got := e.Uint8(); got != 42 {
				t.Errorf("Uint8: got %d, want 42", got)
			}
			if got := e.Float64(); got != 42 {
				t.Errorf("Float64: got %v, want 42", got)
			}
		})
	}
}

// A conversion that would change the value is refused, not truncated. The
// accessors report the refusal as the zero value.
func TestEntryNumericRefusesLossyConversions(t *testing.T) {
	t.Run("negative int is not a uint", func(t *testing.T) {
		e := entryOf(-1)
		if got := e.Uint(); got != 0 {
			t.Errorf("Uint: got %d, want 0", got)
		}
		if got := e.Uint64(); got != 0 {
			t.Errorf("Uint64: got %d, want 0", got)
		}
		// ...but it is still an int.
		if got := e.Int(); got != -1 {
			t.Errorf("Int: got %d, want -1", got)
		}
	})

	t.Run("300 does not fit a uint8", func(t *testing.T) {
		e := entryOf(300)
		if got := e.Uint8(); got != 0 {
			t.Errorf("Uint8: got %d, want 0", got)
		}
		if got := e.Uint16(); got != 300 {
			t.Errorf("Uint16: got %d, want 300", got)
		}
	})

	t.Run("a fraction is not an integer", func(t *testing.T) {
		e := entryOf(42.5)
		if got := e.Int(); got != 0 {
			t.Errorf("Int: got %d, want 0", got)
		}
		if got := e.Uint(); got != 0 {
			t.Errorf("Uint: got %d, want 0", got)
		}
		if got := e.Float64(); got != 42.5 {
			t.Errorf("Float64: got %v, want 42.5", got)
		}
	})

	t.Run("MaxUint64 is not an int64", func(t *testing.T) {
		e := entryOf(uint64(math.MaxUint64))
		if got := e.Int64(); got != 0 {
			t.Errorf("Int64: got %d, want 0", got)
		}
		if got := e.Uint64(); got != math.MaxUint64 {
			t.Errorf("Uint64: got %d, want %d", got, uint64(math.MaxUint64))
		}
	})

	t.Run("NaN and infinity are not integers", func(t *testing.T) {
		for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			if got := entryOf(f).Int64(); got != 0 {
				t.Errorf("Int64 of %v: got %d, want 0", f, got)
			}
		}
	})

	t.Run("a non-numeric value is not a number", func(t *testing.T) {
		for _, v := range []any{nil, true, "abc", []string{"1"}, struct{}{}} {
			if got := entryOf(v).Int(); got != 0 {
				t.Errorf("Int of %v: got %d, want 0", v, got)
			}
		}
	})
}

// The bit size used to parse a string comes from the destination type. Parsing
// every unsigned accessor at 64 bits and converting afterwards silently
// truncated on a 32-bit platform.
func TestEntryNumericStringRespectsWidth(t *testing.T) {
	if got := entryOf("4294967296").Uint32(); got != 0 {
		t.Errorf("Uint32 of a value one past its ceiling: got %d, want 0", got)
	}
	if got := entryOf("4294967295").Uint32(); got != math.MaxUint32 {
		t.Errorf("Uint32 at its ceiling: got %d, want %d", got, uint32(math.MaxUint32))
	}
	if got := entryOf("300").Uint8(); got != 0 {
		t.Errorf("Uint8 of 300: got %d, want 0", got)
	}
	if got := entryOf("-1").Uint(); got != 0 {
		t.Errorf("Uint of -1: got %d, want 0", got)
	}
}

// The strict accessors are the escape hatch for callers that need to know the
// stored type, so coercion must not leak into them.
func TestEntryStrictAccessorsUnchanged(t *testing.T) {
	e := entryOf(42) // an int

	if _, ok := e.GetUint(); ok {
		t.Error("GetUint accepted an int; the strict accessors must stay strict")
	}
	if v, ok := e.GetInt(); !ok || v != 42 {
		t.Errorf("GetInt: got (%d, %v), want (42, true)", v, ok)
	}

	defer func() {
		if recover() == nil {
			t.Error("MustUint did not panic on an int")
		}
	}()
	e.MustUint()
}

// Entries reads through the same conversion, which is the path ctx.Values() and
// ctx.Params() actually take.
func TestEntriesNumericCrossType(t *testing.T) {
	entries := Entries[any]{{Key: "id", Value: 42}}

	if got := entries.GetUint("id"); got != 42 {
		t.Errorf("GetUint: got %d, want 42", got)
	}
	if got := entries.GetInt64("id"); got != 42 {
		t.Errorf("GetInt64: got %d, want 42", got)
	}

	// A missing key still reports its documented sentinel: -1 where the type can
	// express one, 0 where it cannot.
	if got := entries.GetInt("nope"); got != -1 {
		t.Errorf("GetInt of a missing key: got %d, want -1", got)
	}
	if got := entries.GetUint("nope"); got != 0 {
		t.Errorf("GetUint of a missing key: got %d, want 0", got)
	}
}

type userID uint32

func TestNumericDefinedType(t *testing.T) {
	// The number constraint is written with ~, so a value stored as a defined
	// type is still a number.
	if got, ok := toNumber[uint32](userID(42)); !ok || got != 42 {
		t.Errorf("toNumber of a defined type: got (%d, %v), want (42, true)", got, ok)
	}
	if got, ok := toNumber[userID](42); !ok || got != 42 {
		t.Errorf("toNumber into a defined type: got (%d, %v), want (42, true)", got, ok)
	}
}
