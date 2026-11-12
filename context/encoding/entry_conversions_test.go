package encoding

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// numericAccessor is one numeric type seen through its four accessors: the
// lenient one (Entry.Int8), the strict pair (Entry.GetInt8, Entry.MustInt8) and
// the keyed one (Entries.GetInt8). The results are boxed so that a single table
// can hold every type.
type numericAccessor struct {
	name    string
	lo, hi  float64 // the range the type can hold.
	integer bool    // false for the two floats.
	missing any     // what Entries documents for a key that is not there.

	of      func(n float64) any // n as the accessor's own type.
	lenient func(e *Entry[any]) any
	strict  func(e *Entry[any]) (any, bool)
	must    func(e *Entry[any]) any
	keyed   func(entries Entries[any], key string) any
}

func accessorOf[T number](
	name string, lo, hi float64, integer bool, missing T,
	lenient func(*Entry[any]) T,
	strict func(*Entry[any]) (T, bool),
	must func(*Entry[any]) T,
	keyed func(Entries[any], string) T,
) numericAccessor {
	return numericAccessor{
		name:    name,
		lo:      lo,
		hi:      hi,
		integer: integer,
		missing: missing,
		of:      func(n float64) any { return T(n) },
		lenient: func(e *Entry[any]) any { return lenient(e) },
		strict: func(e *Entry[any]) (any, bool) {
			v, ok := strict(e)
			return v, ok
		},
		must:  func(e *Entry[any]) any { return must(e) },
		keyed: func(entries Entries[any], key string) any { return keyed(entries, key) },
	}
}

// The missing column follows the comment above Entries.GetString: -1 for the
// signed and the floating-point types, 0 for the unsigned ones.
var numericAccessors = []numericAccessor{
	accessorOf("Int", math.MinInt, math.MaxInt, true, -1,
		(*Entry[any]).Int, (*Entry[any]).GetInt, (*Entry[any]).MustInt, Entries[any].GetInt),
	accessorOf("Int8", math.MinInt8, math.MaxInt8, true, -1,
		(*Entry[any]).Int8, (*Entry[any]).GetInt8, (*Entry[any]).MustInt8, Entries[any].GetInt8),
	accessorOf("Int16", math.MinInt16, math.MaxInt16, true, -1,
		(*Entry[any]).Int16, (*Entry[any]).GetInt16, (*Entry[any]).MustInt16, Entries[any].GetInt16),
	accessorOf("Int32", math.MinInt32, math.MaxInt32, true, -1,
		(*Entry[any]).Int32, (*Entry[any]).GetInt32, (*Entry[any]).MustInt32, Entries[any].GetInt32),
	accessorOf("Int64", math.MinInt64, math.MaxInt64, true, -1,
		(*Entry[any]).Int64, (*Entry[any]).GetInt64, (*Entry[any]).MustInt64, Entries[any].GetInt64),
	accessorOf("Uint", 0, math.MaxUint, true, 0,
		(*Entry[any]).Uint, (*Entry[any]).GetUint, (*Entry[any]).MustUint, Entries[any].GetUint),
	accessorOf("Uint8", 0, math.MaxUint8, true, 0,
		(*Entry[any]).Uint8, (*Entry[any]).GetUint8, (*Entry[any]).MustUint8, Entries[any].GetUint8),
	accessorOf("Uint16", 0, math.MaxUint16, true, 0,
		(*Entry[any]).Uint16, (*Entry[any]).GetUint16, (*Entry[any]).MustUint16, Entries[any].GetUint16),
	accessorOf("Uint32", 0, math.MaxUint32, true, 0,
		(*Entry[any]).Uint32, (*Entry[any]).GetUint32, (*Entry[any]).MustUint32, Entries[any].GetUint32),
	accessorOf("Uint64", 0, math.MaxUint64, true, 0,
		(*Entry[any]).Uint64, (*Entry[any]).GetUint64, (*Entry[any]).MustUint64, Entries[any].GetUint64),
	accessorOf("Float32", -math.MaxFloat32, math.MaxFloat32, false, -1,
		(*Entry[any]).Float32, (*Entry[any]).GetFloat32, (*Entry[any]).MustFloat32, Entries[any].GetFloat32),
	accessorOf("Float64", -math.MaxFloat64, math.MaxFloat64, false, -1,
		(*Entry[any]).Float64, (*Entry[any]).GetFloat64, (*Entry[any]).MustFloat64, Entries[any].GetFloat64),
	// byte is uint8 and rune is int32, each with accessors of its own.
	accessorOf("Byte", 0, math.MaxUint8, true, 0,
		(*Entry[any]).Byte, (*Entry[any]).GetByte, (*Entry[any]).MustByte, Entries[any].GetByte),
	accessorOf("Rune", math.MinInt32, math.MaxInt32, true, -1,
		(*Entry[any]).Rune, (*Entry[any]).GetRune, (*Entry[any]).MustRune, Entries[any].GetRune),
}

func numericAccessorNamed(t *testing.T, name string) numericAccessor {
	t.Helper()

	for _, acc := range numericAccessors {
		if acc.name == name {
			return acc
		}
	}

	t.Fatalf("no numeric accessor is named %q", name)
	return numericAccessor{}
}

// want is the documented answer of the lenient accessors for a stored number:
// the number itself when the type can hold it intact, zero when it cannot.
func (a numericAccessor) want(n float64) any {
	if n < a.lo || n > a.hi || (a.integer && n != math.Trunc(n)) {
		return a.of(0)
	}
	return a.of(n)
}

// didPanic reports whether fn panicked.
func didPanic(fn func()) (panicked bool) {
	defer func() {
		panicked = recover() != nil
	}()

	fn()
	return false
}

// Every number below is small enough to be exact in a float32 too, so a float
// accessor can hold each of them intact.
func TestNumericAccessorsMatrix(t *testing.T) {
	type numericCase struct {
		name   string
		stored any
		n      float64 // the number the stored value stands for.
	}

	tests := []numericCase{
		// 42 fits everything: stored as each numeric type and as a string.
		{"int 42", 42, 42},
		{"int8 42", int8(42), 42},
		{"int16 42", int16(42), 42},
		{"int32 42", int32(42), 42},
		{"int64 42", int64(42), 42},
		{"uint 42", uint(42), 42},
		{"uint8 42", uint8(42), 42},
		{"uint16 42", uint16(42), 42},
		{"uint32 42", uint32(42), 42},
		{"uint64 42", uint64(42), 42},
		{"float32 42", float32(42), 42},
		{"float64 42", float64(42), 42},
		{"string 42", "42", 42},

		// 300 is past the 8-bit types.
		{"int 300", 300, 300},
		{"int16 300", int16(300), 300},
		{"uint16 300", uint16(300), 300},
		{"float64 300", float64(300), 300},
		{"string 300", "300", 300},

		// -1 is not unsigned, whichever signed width it comes from.
		{"int -1", -1, -1},
		{"int8 -1", int8(-1), -1},
		{"int16 -1", int16(-1), -1},
		{"int32 -1", int32(-1), -1},
		{"int64 -1", int64(-1), -1},
		{"float32 -1", float32(-1), -1},
		{"float64 -1", float64(-1), -1},
		{"string -1", "-1", -1},

		// 1<<40 is past the 32-bit types.
		{"int64 1<<40", int64(1) << 40, 1 << 40},
		{"uint64 1<<40", uint64(1) << 40, 1 << 40},
		{"float64 1<<40", float64(1 << 40), 1 << 40},
		{"string 1<<40", "1099511627776", 1 << 40},

		// A fraction is not an integer.
		{"float64 42.5", 42.5, 42.5},
		{"float32 42.5", float32(42.5), 42.5},
		{"string 42.5", "42.5", 42.5},
	}

	// Both sides of every edge of the 8-bit and the 16-bit types.
	for _, n := range []int{-32769, -32768, -129, -128, 127, 128, 255, 256, 32767, 32768, 65535, 65536} {
		tests = append(tests,
			numericCase{fmt.Sprintf("int %d", n), n, float64(n)},
			numericCase{fmt.Sprintf("string %d", n), strconv.Itoa(n), float64(n)},
		)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := entryOf(tc.stored)
			entries := Entries[any]{{Key: "k", Value: tc.stored}}

			for _, acc := range numericAccessors {
				want := acc.want(tc.n)
				if got := acc.lenient(e); got != want {
					t.Errorf("Entry.%s: got %v, want %v", acc.name, got, want)
				}
				if got := acc.keyed(entries, "k"); got != want {
					t.Errorf("Entries.Get%s: got %v, want %v", acc.name, got, want)
				}
			}
		})
	}
}

// A value that is not a number at all reads as zero. Through Entries that is
// the zero value too and not the missing-key sentinel, because the key is there.
func TestNumericAccessorsNonNumbers(t *testing.T) {
	for _, stored := range []any{"abc", "", "12abc", "forty-two", nil, true, []string{"42"}, struct{}{}} {
		e := entryOf(stored)
		entries := Entries[any]{{Key: "k", Value: stored}}

		for _, acc := range numericAccessors {
			want := acc.of(0)
			if got := acc.lenient(e); got != want {
				t.Errorf("Entry.%s of %#v: got %v, want %v", acc.name, stored, got, want)
			}
			if got := acc.keyed(entries, "k"); got != want {
				t.Errorf("Entries.Get%s of %#v: got %v, want %v", acc.name, stored, got, want)
			}
		}
	}
}

// The edges of the wider types, which a float32 cannot count exactly and so do
// not belong in the matrix above.
func TestNumericAccessorsWidthBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		stored   any
		accessor string
		want     any
	}{
		{"MaxInt32 is an int32", int64(math.MaxInt32), "Int32", int32(math.MaxInt32)},
		{"one past MaxInt32 is not", int64(math.MaxInt32) + 1, "Int32", int32(0)},
		{"MinInt32 is an int32", int64(math.MinInt32), "Int32", int32(math.MinInt32)},
		{"one below MinInt32 is not", int64(math.MinInt32) - 1, "Int32", int32(0)},
		{"MaxUint32 is a uint32", int64(math.MaxUint32), "Uint32", uint32(math.MaxUint32)},
		{"one past MaxUint32 is not", int64(math.MaxUint32) + 1, "Uint32", uint32(0)},
		{"MaxInt64 is an int64", uint64(math.MaxInt64), "Int64", int64(math.MaxInt64)},
		{"one past MaxInt64 is not", uint64(math.MaxInt64) + 1, "Int64", int64(0)},

		{"MaxUint64 as a string is a uint64", "18446744073709551615", "Uint64", uint64(math.MaxUint64)},
		{"one past MaxUint64 as a string is not", "18446744073709551616", "Uint64", uint64(0)},
		{"MinInt64 as a string is an int64", "-9223372036854775808", "Int64", int64(math.MinInt64)},
		{"one below MinInt64 as a string is not", "-9223372036854775809", "Int64", int64(0)},

		{"2^63 as a float is past int64", float64(1 << 63), "Int64", int64(0)},
		{"2^63 as a float is a uint64", float64(1 << 63), "Uint64", uint64(1 << 63)},
		{"-2^63 as a float is MinInt64", -float64(1 << 63), "Int64", int64(math.MinInt64)},
		{"2^64 as a float is past uint64", float64(1<<63) * 2, "Uint64", uint64(0)},

		{"MaxFloat32 is a float32", float64(math.MaxFloat32), "Float32", float32(math.MaxFloat32)},
		{"1e300 is past float32", 1e300, "Float32", float32(0)},
		{"-1e300 is past float32", -1e300, "Float32", float32(0)},
		{"1e300 as a string is past float32", "1e300", "Float32", float32(0)},

		{"NaN is not a uint", math.NaN(), "Uint", uint(0)},
		{"+Inf is not a uint64", math.Inf(1), "Uint64", uint64(0)},
		{"-Inf is not an int8", math.Inf(-1), "Int8", int8(0)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			acc := numericAccessorNamed(t, tc.accessor)

			if got := acc.lenient(entryOf(tc.stored)); got != tc.want {
				t.Errorf("Entry.%s: got %v, want %v", acc.name, got, tc.want)
			}

			entries := Entries[any]{{Key: "k", Value: tc.stored}}
			if got := acc.keyed(entries, "k"); got != tc.want {
				t.Errorf("Entries.Get%s: got %v, want %v", acc.name, got, tc.want)
			}
		})
	}
}

func TestEntriesNumericMissingKeySentinels(t *testing.T) {
	stores := map[string]Entries[any]{
		"another key is there": {{Key: "present", Value: 42}},
		"empty store":          {},
		"nil store":            nil,
	}

	for name, entries := range stores {
		for _, acc := range numericAccessors {
			if got := acc.keyed(entries, "missing"); got != acc.missing {
				t.Errorf("%s: Entries.Get%s of a missing key: got %v, want %v", name, acc.name, got, acc.missing)
			}
		}
	}
}

// The strict pair answers only for the exact stored type. Nothing is converted.
func TestEntryNumericStrictAndMust(t *testing.T) {
	for _, own := range numericAccessors {
		stored := own.of(42)
		e := entryOf(stored)

		for _, acc := range numericAccessors {
			// byte and uint8 are one type, and so are rune and int32.
			sameType := reflect.TypeOf(stored) == reflect.TypeOf(acc.of(0))

			got, ok := acc.strict(e)
			if ok != sameType {
				t.Errorf("Get%s of %T: ok is %v, want %v", acc.name, stored, ok, sameType)
			}

			if !sameType {
				if !didPanic(func() { acc.must(e) }) {
					t.Errorf("Must%s of %T did not panic", acc.name, stored)
				}
				continue
			}

			if got != stored {
				t.Errorf("Get%s of %T: got %v, want %v", acc.name, stored, got, stored)
			}
			if got := acc.must(e); got != stored {
				t.Errorf("Must%s of %T: got %v, want %v", acc.name, stored, got, stored)
			}
		}
	}

	// A numeric string is not the type, and neither is nil.
	for _, stored := range []any{"42", nil} {
		e := entryOf(stored)

		for _, acc := range numericAccessors {
			if _, ok := acc.strict(e); ok {
				t.Errorf("Get%s of %#v reported true", acc.name, stored)
			}
			if !didPanic(func() { acc.must(e) }) {
				t.Errorf("Must%s of %#v did not panic", acc.name, stored)
			}
		}
	}
}

// Query and path values are stored as strings, so the same getters have to work
// on a store of strings.
func TestStringEntriesNumericGetters(t *testing.T) {
	entries := Entries[string]{
		{Key: "n", Value: "42"},
		{Key: "f", Value: "42.5"},
		{Key: "neg", Value: "-1"},
		{Key: "big", Value: "300"},
		{Key: "word", Value: "abc"},
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"GetInt8(n)", entries.GetInt8("n"), int8(42)},
		{"GetInt16(n)", entries.GetInt16("n"), int16(42)},
		{"GetInt32(n)", entries.GetInt32("n"), int32(42)},
		{"GetInt64(n)", entries.GetInt64("n"), int64(42)},
		{"GetUint8(n)", entries.GetUint8("n"), uint8(42)},
		{"GetUint16(n)", entries.GetUint16("n"), uint16(42)},
		{"GetUint32(n)", entries.GetUint32("n"), uint32(42)},
		{"GetUint64(n)", entries.GetUint64("n"), uint64(42)},
		{"GetByte(n)", entries.GetByte("n"), byte(42)},
		{"GetRune(n)", entries.GetRune("n"), rune(42)},
		{"GetFloat32(f)", entries.GetFloat32("f"), float32(42.5)},
		{"GetFloat64(f)", entries.GetFloat64("f"), 42.5},

		// Present, but not a value the type can hold: zero, not the sentinel.
		{"GetInt(f)", entries.GetInt("f"), 0},
		{"GetInt8(big)", entries.GetInt8("big"), int8(0)},
		{"GetUint8(big)", entries.GetUint8("big"), uint8(0)},
		{"GetUint16(neg)", entries.GetUint16("neg"), uint16(0)},
		{"GetInt16(neg)", entries.GetInt16("neg"), int16(-1)},
		{"GetInt32(word)", entries.GetInt32("word"), int32(0)},
		{"GetFloat64(word)", entries.GetFloat64("word"), float64(0)},

		{"GetInt8(missing)", entries.GetInt8("missing"), int8(-1)},
		{"GetUint8(missing)", entries.GetUint8("missing"), uint8(0)},
		{"GetFloat32(missing)", entries.GetFloat32("missing"), float32(-1)},
	}

	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestEntryBool(t *testing.T) {
	for _, stored := range []bool{true, false} {
		e := entryOf(stored)

		if got := e.Bool(); got != stored {
			t.Errorf("Bool of %v: got %v", stored, got)
		}
		if got, ok := e.GetBool(); !ok || got != stored {
			t.Errorf("GetBool of %v: got (%v, %v), want (%v, true)", stored, got, ok, stored)
		}
		if got := e.MustBool(); got != stored {
			t.Errorf("MustBool of %v: got %v", stored, got)
		}
	}

	for _, stored := range []any{"false", "nope", "", 1, 1.0, nil} {
		e := entryOf(stored)

		if e.Bool() {
			t.Errorf("Bool of %#v: got true, want false", stored)
		}
		if _, ok := e.GetBool(); ok {
			t.Errorf("GetBool of %#v reported true", stored)
		}
		if !didPanic(func() { e.MustBool() }) {
			t.Errorf("MustBool of %#v did not panic", stored)
		}
	}
}

func TestEntryByteAndRune(t *testing.T) {
	const lambda = rune(955) // the Greek letter lambda, past the byte range.

	if got := entryOf(byte('A')).Byte(); got != 'A' {
		t.Errorf("Byte of a byte: got %d, want %d", got, 'A')
	}
	if got := entryOf(lambda).Rune(); got != lambda {
		t.Errorf("Rune of a rune: got %d, want %d", got, lambda)
	}

	// Each reads the other while the value fits.
	if got := entryOf(byte('A')).Rune(); got != 'A' {
		t.Errorf("Rune of a byte: got %d, want %d", got, 'A')
	}
	if got := entryOf('A').Byte(); got != 'A' {
		t.Errorf("Byte of a rune that fits: got %d, want %d", got, 'A')
	}
	if got := entryOf(lambda).Byte(); got != 0 {
		t.Errorf("Byte of a rune past 255: got %d, want 0", got)
	}
}

func TestEntryDate(t *testing.T) {
	want := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)

	typed := entryOf(want)
	if got := typed.Date(); !got.Equal(want) {
		t.Errorf("typed: got %s, want %s", got, want)
	}
	if got, ok := typed.GetDate(); !ok || !got.Equal(want) {
		t.Errorf("typed GetDate: got (%s, %v), want (%s, true)", got, ok, want)
	}
	if got := typed.MustDate(); !got.Equal(want) {
		t.Errorf("typed MustDate: got %s, want %s", got, want)
	}

	// The two layouts ParseDate accepts.
	for _, s := range []string{"2026/11/12", "2026-11-12"} {
		text := entryOf(s)
		if got := text.Date(); !got.Equal(want) {
			t.Errorf("string %q: got %s, want %s", s, got, want)
		}
		if _, ok := text.GetDate(); ok {
			t.Errorf("string %q: GetDate reported true for a string", s)
		}
		if !didPanic(func() { text.MustDate() }) {
			t.Errorf("string %q: MustDate did not panic", s)
		}
	}

	for _, stored := range []any{"12/11/2026", "tomorrow", "", 20261112, nil} {
		if got := entryOf(stored).Date(); !got.IsZero() {
			t.Errorf("value %#v: got %s, want the zero time", stored, got)
		}
	}
}

func TestEntryWeekday(t *testing.T) {
	// Sunday is 0, which must not be taken for the -1 that means "no weekday".
	for _, want := range []time.Weekday{time.Sunday, time.Tuesday, time.Saturday} {
		e := entryOf(want)

		if got := e.Weekday(); got != want {
			t.Errorf("Weekday of %s: got %d, want %d", want, got, want)
		}
		if got, ok := e.GetWeekday(); !ok || got != want {
			t.Errorf("GetWeekday of %s: got (%d, %v), want (%d, true)", want, got, ok, want)
		}
		if got := e.MustWeekday(); got != want {
			t.Errorf("MustWeekday of %s: got %d, want %d", want, got, want)
		}
	}

	for _, stored := range []any{"someday", "", "7", "-1", nil} {
		e := entryOf(stored)

		if got := e.Weekday(); got != -1 {
			t.Errorf("Weekday of %#v: got %d, want -1", stored, got)
		}
		if _, ok := e.GetWeekday(); ok {
			t.Errorf("GetWeekday of %#v reported true", stored)
		}
		if !didPanic(func() { e.MustWeekday() }) {
			t.Errorf("MustWeekday of %#v did not panic", stored)
		}
	}
}

func TestEntriesBoolDateWeekdayGetters(t *testing.T) {
	day := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)

	entries := Entries[any]{
		{Key: "on", Value: true},
		{Key: "off", Value: false},
		{Key: "day", Value: day},
		{Key: "day_text", Value: "2026-11-12"},
		{Key: "weekday", Value: time.Thursday},
		{Key: "junk", Value: "junk"},
	}

	if !entries.GetBool("on") {
		t.Error("GetBool(on): got false, want true")
	}
	for _, key := range []string{"off", "junk", "missing"} {
		if entries.GetBool(key) {
			t.Errorf("GetBool(%s): got true, want false", key)
		}
	}

	for _, key := range []string{"day", "day_text"} {
		if got := entries.GetDate(key); !got.Equal(day) {
			t.Errorf("GetDate(%s): got %s, want %s", key, got, day)
		}
	}
	for _, key := range []string{"junk", "missing"} {
		if got := entries.GetDate(key); !got.IsZero() {
			t.Errorf("GetDate(%s): got %s, want the zero time", key, got)
		}
	}

	if got := entries.GetWeekday("weekday"); got != time.Thursday {
		t.Errorf("GetWeekday(weekday): got %d, want %d", got, time.Thursday)
	}
	for _, key := range []string{"junk", "missing"} {
		if got := entries.GetWeekday(key); got != -1 {
			t.Errorf("GetWeekday(%s): got %d, want -1", key, got)
		}
	}
}

func TestEntryMustStringAndStringSlice(t *testing.T) {
	if got := entryOf("text").MustString(); got != "text" {
		t.Errorf("MustString: got %q, want %q", got, "text")
	}
	for _, stored := range []any{42, []string{"text"}, nil} {
		if !didPanic(func() { entryOf(stored).MustString() }) {
			t.Errorf("MustString of %#v did not panic", stored)
		}
	}

	list := []string{"a", "b"}
	if got := entryOf(list).MustStringSlice(); !slices.Equal(got, list) {
		t.Errorf("MustStringSlice: got %v, want %v", got, list)
	}
	for _, stored := range []any{"a/b", 42, nil} {
		if !didPanic(func() { entryOf(stored).MustStringSlice() }) {
			t.Errorf("MustStringSlice of %#v did not panic", stored)
		}
	}
}

func TestEntryStringSliceOfNonStrings(t *testing.T) {
	list := []string{"a", "b"}

	e := entryOf(list)
	if got := e.StringSlice(); !slices.Equal(got, list) {
		t.Errorf("StringSlice of a []string: got %v, want %v", got, list)
	}
	if got, ok := e.GetStringSlice(); !ok || !slices.Equal(got, list) {
		t.Errorf("GetStringSlice of a []string: got (%v, %v), want (%v, true)", got, ok, list)
	}

	for _, stored := range []any{42, true, nil} {
		e := entryOf(stored)
		if got := e.StringSlice(); got != nil {
			t.Errorf("StringSlice of %#v: got %v, want nil", stored, got)
		}
		if _, ok := e.GetStringSlice(); ok {
			t.Errorf("GetStringSlice of %#v reported true", stored)
		}
	}
}

func TestEntriesStringAndStringSliceGetters(t *testing.T) {
	entries := Entries[any]{
		{Key: "name", Value: "iris"},
		{Key: "n", Value: 42},
		{Key: "list", Value: []string{"a", "b"}},
	}

	for key, want := range map[string]string{"name": "iris", "n": "42", "missing": ""} {
		if got := entries.GetString(key); got != want {
			t.Errorf("GetString(%s): got %q, want %q", key, got, want)
		}
	}

	if got, want := entries.GetStringSlice("list"), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("GetStringSlice(list): got %v, want %v", got, want)
	}
	for _, key := range []string{"n", "missing"} {
		if got := entries.GetStringSlice(key); got != nil {
			t.Errorf("GetStringSlice(%s): got %v, want nil", key, got)
		}
	}
}

func TestEntryIsDefault(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"iris.locale", true},
		{"iris.view.layout", true},
		{"user", false},
		{"iris", false},
		{"irisette", false},
		{"my.iris.key", false},
		{"", false},
	}

	for _, tt := range tests {
		e := Entry[any]{Key: tt.key}
		if got := e.IsDefault(); got != tt.want {
			t.Errorf("IsDefault of %q: got %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestEntryGetTrim(t *testing.T) {
	tests := []struct {
		stored any
		want   string
	}{
		{"  iris  ", "iris"},
		{"\tiris\n", "iris"},
		{"iris", "iris"},
		{" two words ", "two words"},
		{"   ", ""},
		{"", ""},
		{42, "42"},
	}

	for _, tt := range tests {
		if got := entryOf(tt.stored).GetTrim(); got != tt.want {
			t.Errorf("GetTrim of %#v: got %q, want %q", tt.stored, got, tt.want)
		}
	}
}

func TestDecodeQueryUnescapesOnce(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"a%20b", "a b"},
		{"a+b", "a b"},
		{"a%2Bb", "a+b"},
		{"caf%C3%A9", "caf\xc3\xa9"},
		{"%2541", "%41"},
		{"%zz", "%zz"}, // not an escape sequence, so it comes back as it is.
	}

	for _, tt := range tests {
		if got := DecodeQuery(tt.in); got != tt.want {
			t.Errorf("DecodeQuery(%q): got %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEntryEscape(t *testing.T) {
	tests := []struct {
		stored any
		want   string
	}{
		{"a%20b", "a b"},
		{"hello+world", "hello world"},
		{"caf%C3%A9", "caf\xc3\xa9"},
		{"plain", "plain"},
		{"", ""},
		{42, "42"},
	}

	for _, tt := range tests {
		if got := entryOf(tt.stored).Escape(); got != tt.want {
			t.Errorf("Escape of %#v: got %q, want %q", tt.stored, got, tt.want)
		}
	}
}

func TestEntryTrimKnownExtensions(t *testing.T) {
	tests := []struct {
		name   string
		trim   func(e *Entry[any]) string
		stored string
		want   string
	}{
		{"TrimHTML", (*Entry[any]).TrimHTML, "index.html", "index"},
		{"TrimHTML", (*Entry[any]).TrimHTML, "index", "index"},
		{"TrimHTML", (*Entry[any]).TrimHTML, "", ""},
		{"TrimJSON", (*Entry[any]).TrimJSON, "page.json", "page"},
		{"TrimJSON", (*Entry[any]).TrimJSON, "page", "page"},
		{"TrimJSON", (*Entry[any]).TrimJSON, "", ""},
		{"TrimXML", (*Entry[any]).TrimXML, "feed.xml", "feed"},
		{"TrimXML", (*Entry[any]).TrimXML, "feed", "feed"},
		{"TrimXML", (*Entry[any]).TrimXML, "", ""},
	}

	for _, tt := range tests {
		if got := tt.trim(entryOf(tt.stored)); got != tt.want {
			t.Errorf("%s of %q: got %q, want %q", tt.name, tt.stored, got, tt.want)
		}
	}
}

func TestEntryIntUnslashed(t *testing.T) {
	tests := []struct {
		stored any
		want   int
	}{
		{"/42", 42},
		{"42", 42},
		{"/-5", -5},
		{42, 42},
		{"", -1},
		{"/", -1},
		{"abc", -1},
		{"/abc", -1},
		{"//42", -1}, // only the first slash is dropped.
		{"42/", -1},
	}

	for _, tt := range tests {
		if got := entryOf(tt.stored).IntUnslashed(); got != tt.want {
			t.Errorf("IntUnslashed of %#v: got %d, want %d", tt.stored, got, tt.want)
		}
	}
}

// The string helpers of Entries answer like the Entry ones for a key that is
// there, and with "" or -1 for one that is not.
func TestEntriesStringHelpers(t *testing.T) {
	entries := Entries[string]{
		{Key: "padded", Value: "  iris  "},
		{Key: "encoded", Value: "a%20b"},
		{Key: "file", Value: "main.css"},
		{Key: "id_file", Value: "42.json"},
		{Key: "html", Value: "index.html"},
		{Key: "json", Value: "page.json"},
		{Key: "xml", Value: "feed.xml"},
		{Key: "slashed", Value: "/42"},
	}

	strs := []struct {
		name string
		get  func(key string) string
		key  string
		want string
	}{
		{"GetTrim", entries.GetTrim, "padded", "iris"},
		{"GetEscape", entries.GetEscape, "encoded", "a b"},
		{"GetTrimFile", entries.GetTrimFile, "file", "main"},
		{"GetTrimHTML", entries.GetTrimHTML, "html", "index"},
		{"GetTrimJSON", entries.GetTrimJSON, "json", "page"},
		{"GetTrimXML", entries.GetTrimXML, "xml", "feed"},
	}
	for _, tt := range strs {
		if got := tt.get(tt.key); got != tt.want {
			t.Errorf("%s(%s): got %q, want %q", tt.name, tt.key, got, tt.want)
		}
		if got := tt.get("missing"); got != "" {
			t.Errorf("%s(missing): got %q, want \"\"", tt.name, got)
		}
	}

	ints := []struct {
		name string
		get  func(key string) int
		key  string
		want int
	}{
		{"GetTrimFileInt", entries.GetTrimFileInt, "id_file", 42},
		{"GetTrimFileInt", entries.GetTrimFileInt, "missing", -1},
		{"GetIntUnslashed", entries.GetIntUnslashed, "slashed", 42},
		{"GetIntUnslashed", entries.GetIntUnslashed, "file", -1},
		{"GetIntUnslashed", entries.GetIntUnslashed, "missing", -1},
	}
	for _, tt := range ints {
		if got := tt.get(tt.key); got != tt.want {
			t.Errorf("%s(%s): got %d, want %d", tt.name, tt.key, got, tt.want)
		}
	}
}

func TestErrEntryNotFound(t *testing.T) {
	tests := []struct {
		key   string
		typ   reflect.Type
		kind  reflect.Kind
		parts []string // what the message has to name.
	}{
		{"id", reflect.TypeFor[int](), reflect.Int, []string{"not found", "id", "int"}},
		{"since", reflect.TypeFor[time.Time](), reflect.Struct, []string{"not found", "since", "struct", "time.Time"}},
	}

	for _, tt := range tests {
		entry := Entry[any]{Key: tt.key, Value: "whatever is stored"}

		err := entry.NotFound(tt.typ)
		if err.Key != tt.key || err.Kind != tt.kind || err.Type != tt.typ {
			t.Errorf("NotFound(%s): got %+v, want the key %q, the kind %s and the type %s", tt.typ, *err, tt.key, tt.kind, tt.typ)
		}

		msg := err.Error()
		for _, part := range tt.parts {
			if !strings.Contains(msg, part) {
				t.Errorf("Error of %q as %s: %q does not mention %q", tt.key, tt.typ, msg, part)
			}
		}
	}
}

// As called by hand with a probe: an empty Key and an Invalid Kind match
// anything, a set one has to be equal.
func TestErrEntryNotFoundAs(t *testing.T) {
	err := Entry[any]{Key: "id"}.NotFound(reflect.TypeFor[int]())

	tests := []struct {
		name   string
		target any
		want   bool
	}{
		{"empty probe", &ErrEntryNotFound{}, true},
		{"same key", &ErrEntryNotFound{Key: "id"}, true},
		{"another key", &ErrEntryNotFound{Key: "name"}, false},
		{"same kind", &ErrEntryNotFound{Kind: reflect.Int}, true},
		{"another kind", &ErrEntryNotFound{Kind: reflect.String}, false},
		{"same key and kind", &ErrEntryNotFound{Key: "id", Kind: reflect.Int}, true},
		{"same key, another kind", &ErrEntryNotFound{Key: "id", Kind: reflect.String}, false},
		{"another key, same kind", &ErrEntryNotFound{Key: "name", Kind: reflect.Int}, false},
		{"not a probe", new(string), false},
	}

	for _, tt := range tests {
		if got := err.As(tt.target); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestErrEntryNotFoundThroughErrorsAs(t *testing.T) {
	notFound := Entry[any]{Key: "id"}.NotFound(reflect.TypeFor[int]())
	wrapped := fmt.Errorf("read the params: %w", notFound)

	// The usage of the doc comment: a pointer variable, then its fields.
	var target *ErrEntryNotFound
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As did not find the ErrEntryNotFound in a wrapped error")
	}
	if target.Key != "id" || target.Kind != reflect.Int {
		t.Errorf("errors.As target: got %+v, want the key id and the kind int", *target)
	}

	var pathErr *fs.PathError
	if errors.As(wrapped, &pathErr) {
		t.Error("errors.As matched an ErrEntryNotFound as a *fs.PathError")
	}

	var none *ErrEntryNotFound
	if errors.As(errors.New("another error"), &none) {
		t.Error("errors.As found an ErrEntryNotFound in an unrelated error")
	}
}
