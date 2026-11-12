package reflex

import "reflect"

// IndirectType returns the value of a pointer-type "typ".
// If "IndirectType" is a pointer, array, chan, map or slice it returns its Elem,
// otherwise returns the "typ" as it is.
func IndirectType(typ reflect.Type) reflect.Type {
	switch typ.Kind() {
	case reflect.Pointer, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return typ.Elem()
	}
	return typ
}

// TypeOf returns the reflect.Type of the given value.
func TypeOf(v any) reflect.Type {
	typ, ok := v.(reflect.Type)
	if !ok {
		typ = reflect.TypeOf(v)
	}

	return typ
}

// IndirectValue returns the value a pointer points to (e.g. for a *User value it returns the User value).
// Non-pointer values are returned as they are and a nil pointer returns the zero reflect.Value.
func IndirectValue(v any) reflect.Value {
	return reflect.Indirect(ValueOf(v))
}

// ValueOf returns the reflect.Value of the given value.
func ValueOf(v any) reflect.Value {
	val, ok := v.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(v)
	}

	return val
}
