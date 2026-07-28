// white-box testing
package memstore

import (
	"testing"
)

func entry(value any) Entry {
	return Entry{Key: "key", ValueRaw: value}
}

func TestEntrySignedDefaults(t *testing.T) {
	// in-range values round-trip through every signed getter, from both
	// native integer types and their decimal string form
	t.Run("int8", func(t *testing.T) {
		for _, v := range []any{int8(8), int(8), int16(8), int32(8), int64(8), uint(8), "8"} {
			got, err := entry(v).Int8Default(-1)
			if err != nil {
				t.Fatalf("%T(%v): unexpected error %v", v, v, err)
			}
			if got != 8 {
				t.Fatalf("%T(%v): expected 8 but got %d", v, v, got)
			}
		}
	})

	t.Run("int16", func(t *testing.T) {
		for _, v := range []any{int16(300), int(300), int32(300), int64(300), "300"} {
			got, err := entry(v).Int16Default(-1)
			if err != nil {
				t.Fatalf("%T(%v): unexpected error %v", v, v, err)
			}
			if got != 300 {
				t.Fatalf("%T(%v): expected 300 but got %d", v, v, got)
			}
		}
	})

	t.Run("int32", func(t *testing.T) {
		got, err := entry(int32(70000)).Int32Default(-1)
		if err != nil || got != 70000 {
			t.Fatalf("expected 70000 but got %d (err=%v)", got, err)
		}
	})

	t.Run("int64", func(t *testing.T) {
		got, err := entry(int64(5_000_000_000)).Int64Default(-1)
		if err != nil || got != 5_000_000_000 {
			t.Fatalf("expected 5000000000 but got %d (err=%v)", got, err)
		}
	})
}

func TestEntrySignedDefaultsErrors(t *testing.T) {
	// nil, unparsable strings, strings that do not fit the target width,
	// and unsupported types all report an error and yield the default
	t.Run("int8", func(t *testing.T) {
		for _, v := range []any{nil, "abc", "200", true} {
			got, err := entry(v).Int8Default(-1)
			if err == nil {
				t.Fatalf("%#v: expected an error", v)
			}
			if got != -1 {
				t.Fatalf("%#v: expected the default but got %d", v, got)
			}
		}
	})

	t.Run("int16", func(t *testing.T) {
		for _, v := range []any{nil, "abc", "40000", true} {
			if _, err := entry(v).Int16Default(-1); err == nil {
				t.Fatalf("%#v: expected an error", v)
			}
		}
	})
}

func TestEntryUnsignedDefaults(t *testing.T) {
	t.Run("uint", func(t *testing.T) {
		for _, v := range []any{uint(8), uint8(8), uint16(8), uint32(8), uint64(8), "8"} {
			got, err := entry(v).UintDefault(0)
			if err != nil {
				t.Fatalf("%T(%v): unexpected error %v", v, v, err)
			}
			if got != 8 {
				t.Fatalf("%T(%v): expected 8 but got %d", v, v, got)
			}
		}
	})

	t.Run("uint8", func(t *testing.T) {
		got, err := entry(uint8(255)).Uint8Default(0)
		if err != nil || got != 255 {
			t.Fatalf("expected 255 but got %d (err=%v)", got, err)
		}
		got, err = entry("255").Uint8Default(0)
		if err != nil || got != 255 {
			t.Fatalf("expected 255 from a string but got %d (err=%v)", got, err)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		got, err := entry(uint64(5_000_000_000)).Uint64Default(0)
		if err != nil || got != 5_000_000_000 {
			t.Fatalf("expected 5000000000 but got %d (err=%v)", got, err)
		}
	})
}

// Unlike the signed getters, the unsigned ones reject values that do not fit
// the target width instead of truncating them.
func TestEntryUnsignedDefaultsRejectOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"uint16 above uint8 max", uint16(300)},
		{"uint32 above uint8 max", uint32(300)},
		{"uint64 above uint8 max", uint64(300)},
		{"uint above uint8 max", uint(300)},
		{"int above uint8 max", int(300)},
		{"negative int", int(-1)},
		{"string above uint8 max", "256"},
		{"negative string", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entry(tt.value).Uint8Default(7)
			if err == nil {
				t.Fatalf("expected an out of range value to report an error")
			}
			if got != 7 {
				t.Fatalf("expected the default 7 but got %d", got)
			}
		})
	}
}

func TestEntryFloat64Default(t *testing.T) {
	got, err := entry(3.5).Float64Default(-1)
	if err != nil || got != 3.5 {
		t.Fatalf("expected 3.5 but got %v (err=%v)", got, err)
	}

	got, err = entry("3.5").Float64Default(-1)
	if err != nil || got != 3.5 {
		t.Fatalf("expected 3.5 from a string but got %v (err=%v)", got, err)
	}

	got, err = entry(int(3)).Float64Default(-1)
	if err != nil || got != 3 {
		t.Fatalf("expected 3 from an int but got %v (err=%v)", got, err)
	}

	if _, err = entry(nil).Float64Default(-1); err == nil {
		t.Fatalf("expected an error for a nil value")
	}
	if _, err = entry("abc").Float64Default(-1); err == nil {
		t.Fatalf("expected an error for an unparsable string")
	}
}

// the Store level getters delegate to the Entry ones and fall back to the
// default when the key is absent
func TestStoreSizedGetters(t *testing.T) {
	var p Store
	p.Set("i8", int8(8))
	p.Set("i16", int16(16))
	p.Set("i32", int32(32))
	p.Set("i64", int64(64))
	p.Set("u", uint(1))
	p.Set("u8", uint8(8))
	p.Set("u16", uint16(16))
	p.Set("u32", uint32(32))
	p.Set("u64", uint64(64))
	p.Set("f", 1.5)

	if v := p.GetInt8Default("i8", -1); v != 8 {
		t.Fatalf("expected 8 but got %d", v)
	}
	if v := p.GetInt16Default("i16", -1); v != 16 {
		t.Fatalf("expected 16 but got %d", v)
	}
	if v := p.GetInt32Default("i32", -1); v != 32 {
		t.Fatalf("expected 32 but got %d", v)
	}
	if v := p.GetInt64Default("i64", -1); v != 64 {
		t.Fatalf("expected 64 but got %d", v)
	}
	if v := p.GetUintDefault("u", 0); v != 1 {
		t.Fatalf("expected 1 but got %d", v)
	}
	if v := p.GetUint8Default("u8", 0); v != 8 {
		t.Fatalf("expected 8 but got %d", v)
	}
	if v := p.GetUint16Default("u16", 0); v != 16 {
		t.Fatalf("expected 16 but got %d", v)
	}
	if v := p.GetUint32Default("u32", 0); v != 32 {
		t.Fatalf("expected 32 but got %d", v)
	}
	if v := p.GetUint64Default("u64", 0); v != 64 {
		t.Fatalf("expected 64 but got %d", v)
	}
	if v := p.GetFloat64Default("f", -1); v != 1.5 {
		t.Fatalf("expected 1.5 but got %v", v)
	}

	// every getter falls back to its default for a missing key
	if v := p.GetInt8Default("missing", -1); v != -1 {
		t.Fatalf("expected the default but got %d", v)
	}
	if v := p.GetUint64Default("missing", 9); v != 9 {
		t.Fatalf("expected the default but got %d", v)
	}
	if v := p.GetFloat64Default("missing", -1); v != -1 {
		t.Fatalf("expected the default but got %v", v)
	}
}
