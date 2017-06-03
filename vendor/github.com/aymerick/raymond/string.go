package raymond

import (
	"fmt"
	"reflect"
	"strconv"
)

// SafeString represents a string that must not be escaped.
//
// A SafeString can be returned by helpers to disable escaping.
type SafeString string

// isSafeString returns true if argument is a SafeString
func isSafeString(value interface{}) bool {
	if _, ok := value.(SafeString); ok {
		return true
	}
	return false
}

// Str returns string representation of any basic type value.
func Str(value interface{}) string {
	return strValue(reflect.ValueOf(value))
}

// strValue returns string representation of a reflect.Value
func strValue(value reflect.Value) string {
	result := ""

	ival, ok := printableValue(value)
	if !ok {
		panic(fmt.Errorf("Can't print value: %q", value))
	}

	val := reflect.ValueOf(ival)

	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			result += strValue(val.Index(i))
		}
	case reflect.Bool:
		result = "false"
		if val.Bool() {
			result = "true"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		result = fmt.Sprintf("%d", ival)
	case reflect.Float32, reflect.Float64:
		result = strconv.FormatFloat(val.Float(), 'f', -1, 64)
	case reflect.Invalid:
		result = ""
	default:
		result = fmt.Sprintf("%s", ival)
	}

	return result
}

// printableValue returns the, possibly indirected, interface value inside v that
// is best for a call to formatted printer.
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/exec.go
func printableValue(v reflect.Value) (interface{}, bool) {
	if v.Kind() == reflect.Ptr {
		v, _ = indirect(v) // fmt.Fprint handles nil.
	}
	if !v.IsValid() {
		return "", true
	}

	if !v.Type().Implements(errorType) && !v.Type().Implements(fmtStringerType) {
		if v.CanAddr() && (reflect.PtrTo(v.Type()).Implements(errorType) || reflect.PtrTo(v.Type()).Implements(fmtStringerType)) {
			v = v.Addr()
		} else {
			switch v.Kind() {
			case reflect.Chan, reflect.Func:
				return nil, false
			}
		}
	}
	return v.Interface(), true
}
