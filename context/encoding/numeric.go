package encoding

import (
	"math"
	"reflect"
	"strconv"
)

// number is every type an Entry accessor can resolve a value to.
//
// Written with ~ so a defined type over a numeric one (`type UserID uint32`)
// belongs to it, because a value stored as one is still a number.
type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// These are the exact float64 boundaries of the 64-bit integer types, not the
// nearest float to math.MaxInt64. float64(math.MaxInt64) rounds *up* to 2^63, so
// comparing against it would admit a value that int64 cannot hold, and Go leaves
// an out-of-range float-to-integer conversion implementation-defined. 2^63 and
// 2^64 are both exactly representable, so a half-open bound is precise.
const (
	twoTo63 = float64(1 << 63)
	twoTo64 = twoTo63 * 2
)

// toNumber resolves a stored value to T.
//
// Three chances, in order: the value already is a T; the value is a numeric
// string; the value is some other Go numeric that survives the conversion
// intact.
//
// That third case is the one worth explaining. Before it existed, an exact type
// assertion and a string were the only things tried, so this returned zero:
//
//	ctx.Values().Set("id", 42) // an untyped 42 is stored as an int
//	ctx.Values().GetUint("id") // 0
//
// A value went in and nothing came back, which is the same complaint that
// kataras/iris#2608 fixed in the v12 store, reached from the opposite direction:
// there an int was rejected by a broken range check, here it was never offered
// to one. JSON makes it routine rather than exotic — every number decoded from a
// request body is a float64, so reading one back as an int hit this too.
//
// Conversions that would not survive are refused rather than truncated: a
// negative int is not a uint, 300 is not a uint8, and 42.5 is not an int. The
// caller gets the same false a non-numeric string gets, and the accessors turn
// that into the zero value.
func toNumber[T number](v any) (T, bool) {
	if typed, ok := v.(T); ok {
		return typed, true
	}

	if s, ok := v.(string); ok {
		return parseNumber[T](s)
	}

	return convertNumber[T](v)
}

// parseNumber reads T out of its decimal representation.
//
// The bit size comes from T rather than being passed in, which is the point:
// the hand-written version of this parsed every unsigned accessor at 64 bits and
// then converted, so on a 32-bit platform Uint() turned "4294967296" into 0
// instead of refusing it.
func parseNumber[T number](s string) (T, bool) {
	var zero T
	dst := reflect.ValueOf(&zero).Elem()

	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, dst.Type().Bits())
		if err != nil {
			return zero, false
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, dst.Type().Bits())
		if err != nil {
			return zero, false
		}
		dst.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, dst.Type().Bits())
		if err != nil {
			return zero, false
		}
		dst.SetFloat(f)
	default:
		return zero, false
	}

	return zero, true
}

// convertNumber converts any Go numeric to T, or reports false if the value
// would not survive the trip.
func convertNumber[T number](v any) (T, bool) {
	var zero T
	if v == nil {
		return zero, false
	}

	src := reflect.ValueOf(v)
	dst := reflect.ValueOf(&zero).Elem()

	var ok bool
	switch src.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ok = setSigned(dst, src.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		ok = setUnsigned(dst, src.Uint())
	case reflect.Float32, reflect.Float64:
		ok = setFloat(dst, src.Float())
	}
	if !ok {
		return zero, false
	}

	return zero, true
}

func setSigned(dst reflect.Value, n int64) bool {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if dst.OverflowInt(n) {
			return false
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n < 0 || dst.OverflowUint(uint64(n)) {
			return false
		}
		dst.SetUint(uint64(n))
	case reflect.Float32, reflect.Float64:
		if dst.OverflowFloat(float64(n)) {
			return false
		}
		dst.SetFloat(float64(n))
	default:
		return false
	}

	return true
}

func setUnsigned(dst reflect.Value, n uint64) bool {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n > math.MaxInt64 || dst.OverflowInt(int64(n)) {
			return false
		}
		dst.SetInt(int64(n))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if dst.OverflowUint(n) {
			return false
		}
		dst.SetUint(n)
	case reflect.Float32, reflect.Float64:
		if dst.OverflowFloat(float64(n)) {
			return false
		}
		dst.SetFloat(float64(n))
	default:
		return false
	}

	return true
}

func setFloat(dst reflect.Value, f float64) bool {
	switch dst.Kind() {
	case reflect.Float32, reflect.Float64:
		if dst.OverflowFloat(f) {
			return false
		}
		dst.SetFloat(f)
		return true
	}

	// Everything below targets an integer, so a fraction, an infinity or a NaN
	// has no answer. 42.0 converts, 42.5 does not: refusing beats rounding a
	// value the caller never wrote.
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) {
		return false
	}

	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f < -twoTo63 || f >= twoTo63 {
			return false
		}
		n := int64(f)
		if dst.OverflowInt(n) {
			return false
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if f < 0 || f >= twoTo64 {
			return false
		}
		n := uint64(f)
		if dst.OverflowUint(n) {
			return false
		}
		dst.SetUint(n)
	default:
		return false
	}

	return true
}
