package mvc2

import "reflect"

func isContext(inTyp reflect.Type) bool {
	return inTyp.String() == "context.Context" // I couldn't find another way; context/context.go is not exported.
}

func indirectVal(v reflect.Value) reflect.Value {
	return reflect.Indirect(v)
}

func indirectTyp(typ reflect.Type) reflect.Type {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return typ.Elem()
	}
	return typ
}

func goodVal(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		if v.IsNil() {
			return false
		}
	}

	return v.IsValid()
}

func isFunc(typ reflect.Type) bool {
	return typ.Kind() == reflect.Func
}

/*
// no f. this, it's too complicated and it will be harder to maintain later on:
func isSliceAndExpectedItem(got reflect.Type, in []reflect.Type, currentBindersIdx int) bool {
	kind := got.Kind()
	// if got result is slice or array.
	return (kind == reflect.Slice || kind == reflect.Array) &&
		// if has expected next input.
		len(in)-1 > currentBindersIdx &&
		// if the current input's type is not the same as got (if it's not a slice of that types or anything else).
		equalTypes(got, in[currentBindersIdx])
}
*/

func equalTypes(got reflect.Type, expected reflect.Type) bool {
	if got == expected {
		return true
	}
	// if accepts an interface, check if the given "got" type does
	// implement this "expected" user handler's input argument.
	if expected.Kind() == reflect.Interface {
		// fmt.Printf("expected interface = %s and got to set on the arg is: %s\n", expected.String(), got.String())
		return got.Implements(expected)
	}
	return false
}

// for controller only.

func structFieldIgnored(f reflect.StructField) bool {
	if !f.Anonymous {
		return true // if not anonymous(embedded), ignore it.
	}

	s := f.Tag.Get("ignore")
	return s == "true" // if has an ignore tag then ignore it.
}

type field struct {
	Type  reflect.Type
	Index []int  // the index of the field, slice if it's part of a embedded struct
	Name  string // the actual name

	// this could be empty, but in our cases it's not,
	// it's filled with the service and it's filled from the lookupFields' caller.
	AnyValue reflect.Value
}

func lookupFields(typ reflect.Type, parentIndex int) (fields []field) {
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)

		if f.Type.Kind() == reflect.Struct && !structFieldIgnored(f) {
			fields = append(fields, lookupFields(f.Type, i)...)
			continue
		}

		index := []int{i}
		if parentIndex >= 0 {
			index = append([]int{parentIndex}, index...)
		}

		field := field{
			Type:  f.Type,
			Name:  f.Name,
			Index: index,
		}

		fields = append(fields, field)
	}

	return
}
