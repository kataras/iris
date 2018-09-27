package formbinder

import (
	"encoding"
	"reflect"
	"time"
)

var (
	typeTime    = reflect.TypeOf(time.Time{})
	typeTimePtr = reflect.TypeOf(&time.Time{})
)

// unmarshalText returns a boolean and error. The boolean is true if the
// value implements TextUnmarshaler, and false if not.
func checkUnmarshalText(v reflect.Value, val string) (bool, error) {
	// check if implements the interface
	m, ok := v.Interface().(encoding.TextUnmarshaler)
	addr := v.CanAddr()
	if !ok && !addr {
		return false, nil
	} else if addr {
		return checkUnmarshalText(v.Addr(), val)
	}
	// skip if the type is time.Time
	n := v.Type()
	if n.ConvertibleTo(typeTime) || n.ConvertibleTo(typeTimePtr) {
		return false, nil
	}
	// return result
	return true, m.UnmarshalText([]byte(val))
}
