package reflex

import "reflect"

// IndirectType returns the value of a pointer-type "typ".
// If "IndirectType" is a pointer, array, chan, map or slice it returns its Elem,
// otherwise returns the "typ" as it is.
func IndirectType(typ reflect.Type) reflect.Type {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return typ.Elem()
	}
	return typ
}

// IndirectValue returns the element type (e.g. if pointer of *User it will return the User type).
func IndirectValue(val reflect.Value) reflect.Value {
	return reflect.Indirect(val)
}
