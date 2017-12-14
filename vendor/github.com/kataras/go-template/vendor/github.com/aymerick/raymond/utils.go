package raymond

import (
	"path"
	"reflect"
)

// indirect returns the item at the end of indirection, and a bool to indicate if it's nil.
// We indirect through pointers and empty interfaces (only) because
// non-empty interfaces have methods we might need.
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/exec.go
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
		if v.Kind() == reflect.Interface && v.NumMethod() > 0 {
			break
		}
	}
	return v, false
}

// IsTrue returns true if obj is a truthy value.
func IsTrue(obj interface{}) bool {
	thruth, ok := isTrueValue(reflect.ValueOf(obj))
	if !ok {
		return false
	}
	return thruth
}

// isTrueValue reports whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/exec.go
func isTrueValue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}
	return truth, true
}

// canBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/exec.go
func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

// fileBase returns base file name
//
// example: /foo/bar/baz.png => baz
func fileBase(filePath string) string {
	fileName := path.Base(filePath)
	fileExt := path.Ext(filePath)

	return fileName[:len(fileName)-len(fileExt)]
}
