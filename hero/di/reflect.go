package di

import (
	"reflect"

	"github.com/kataras/iris/v12/context"
)

// EmptyIn is just an empty slice of reflect.Value.
var EmptyIn = []reflect.Value{}

// IsZero returns true if a value is nil.
// Remember; fields to be checked should be exported otherwise it returns false.
// Notes for users:
// Boolean's zero value is false, even if not set-ed.
// UintXX are not zero on 0 because they are pointers to.
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct:
		zero := true
		for i := 0; i < v.NumField(); i++ {
			zero = zero && IsZero(v.Field(i))
		}

		if typ := v.Type(); typ != nil && v.IsValid() {
			f, ok := typ.MethodByName("IsZero")
			// if not found
			// if has input arguments (1 is for the value receiver, so > 1 for the actual input args)
			// if output argument is not boolean
			// then skip this IsZero user-defined function.
			if !ok || f.Type.NumIn() > 1 || f.Type.NumOut() != 1 && f.Type.Out(0).Kind() != reflect.Bool {
				return zero
			}

			method := v.Method(f.Index)
			// no needed check but:
			if method.IsValid() && !method.IsNil() {
				// it shouldn't panic here.
				zero = method.Call(EmptyIn)[0].Interface().(bool)
			}
		}

		return zero
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		zero := true
		for i := 0; i < v.Len(); i++ {
			zero = zero && IsZero(v.Index(i))
		}
		return zero
	}
	// if not any special type then use the reflect's .Zero
	// usually for fields, but remember if it's boolean and it's false
	// then it's zero, even if set-ed.

	if !v.CanInterface() {
		// if can't interface, i.e return value from unexported field or method then return false
		return false
	}

	zero := reflect.Zero(v.Type())
	return v.Interface() == zero.Interface()
}

var errTyp = reflect.TypeOf((*error)(nil)).Elem()

// IsError returns true if "typ" is type of `error`.
func IsError(typ reflect.Type) bool {
	return typ.Implements(errTyp)
}

// IndirectValue returns the reflect.Value that "v" points to.
// If "v" is a nil pointer, Indirect returns a zero Value.
// If "v" is not a pointer, Indirect returns v.
func IndirectValue(v reflect.Value) reflect.Value {
	if k := v.Kind(); k == reflect.Ptr { //|| k == reflect.Interface {
		return v.Elem()
	}

	return v
}

// ValueOf returns the reflect.Value of "o".
// If "o" is already a reflect.Value returns "o".
func ValueOf(o interface{}) reflect.Value {
	if v, ok := o.(reflect.Value); ok {
		return v
	}

	return reflect.ValueOf(o)
}

// ValuesOf same as `ValueOf` but accepts a slice of
// somethings and returns a slice of reflect.Value.
func ValuesOf(valuesAsInterface []interface{}) (values []reflect.Value) {
	for _, v := range valuesAsInterface {
		values = append(values, ValueOf(v))
	}
	return
}

// IndirectType returns the value of a pointer-type "typ".
// If "typ" is a pointer, array, chan, map or slice it returns its Elem,
// otherwise returns the typ as it's.
func IndirectType(typ reflect.Type) reflect.Type {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return typ.Elem()
	}
	return typ
}

// IsNil same as `reflect.IsNil` but a bit safer to use, returns false if not a correct type.
func IsNil(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
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

var contextTyp = reflect.TypeOf((*context.Context)(nil)).Elem()

// IsContext returns true if the "inTyp" is a type of Context.
func IsContext(inTyp reflect.Type) bool {
	return inTyp.Implements(contextTyp)
}

func goodFunc(fn reflect.Type) bool {
	// valid if that single input arg is a typeof context.Context
	// or first argument is context.Context and second argument is a variadic, which is ignored (i.e new sessions#Start).
	return (fn.NumIn() == 1 || (fn.NumIn() == 2 && fn.IsVariadic())) && IsContext(fn.In(0))
}

// IsFunc returns true if the passed type is function.
func IsFunc(kindable interface {
	Kind() reflect.Kind
}) bool {
	return kindable.Kind() == reflect.Func
}

var reflectValueType = reflect.TypeOf(reflect.Value{})

func equalTypes(got reflect.Type, expected reflect.Type) bool {
	if got == expected {
		return true
	}

	// fmt.Printf("got: %s expected: %s\n", got.String(), expected.String())
	// if accepts an interface, check if the given "got" type does
	// implement this "expected" user handler's input argument.
	if expected.Kind() == reflect.Interface {
		// fmt.Printf("expected interface = %s and got to set on the arg is: %s\n", expected.String(), got.String())
		// return got.Implements(expected)
		// return expected.AssignableTo(got)
		return got.AssignableTo(expected)
	}

	return false
}

// for controller's fields only.
func structFieldIgnored(f reflect.StructField) bool {
	if !f.Anonymous {
		return true // if not anonymous(embedded), ignore it.
	}

	s := f.Tag.Get("ignore")
	return s == "true" // if has an ignore tag then ignore it.
}

// for controller's fields only. Explicit set a stateless to a field
// in order to make the controller a Stateless one even if no other dynamic dependencies exist.
func structFieldStateless(f reflect.StructField) bool {
	s := f.Tag.Get("stateless")
	return s == "true"
}

type field struct {
	Type   reflect.Type
	Name   string // the actual name.
	Index  []int  // the index of the field, slice if it's part of a embedded struct
	CanSet bool   // is true if it's exported.
}

// NumFields returns the total number of fields, and the embedded, even if the embedded struct is not exported,
// it will check for its exported fields.
func NumFields(elemTyp reflect.Type, skipUnexported bool) int {
	return len(lookupFields(elemTyp, skipUnexported, nil))
}

func lookupFields(elemTyp reflect.Type, skipUnexported bool, parentIndex []int) (fields []field) {
	if elemTyp.Kind() != reflect.Struct {
		return
	}

	for i, n := 0, elemTyp.NumField(); i < n; i++ {
		f := elemTyp.Field(i)
		if IndirectType(f.Type).Kind() == reflect.Struct &&
			!structFieldIgnored(f) && !structFieldStateless(f) {
			fields = append(fields, lookupFields(f.Type, skipUnexported, append(parentIndex, i))...)
			continue
		}

		// skip unexported fields here,
		// after the check for embedded structs, these can be binded if their
		// fields are exported.
		isExported := f.PkgPath == ""
		if skipUnexported && !isExported {
			continue
		}

		index := []int{i}
		if len(parentIndex) > 0 {
			index = append(parentIndex, i)
		}

		tmp := make([]int, len(index))
		copy(tmp, index)

		field := field{
			Type:   f.Type,
			Name:   f.Name,
			Index:  tmp,
			CanSet: isExported,
		}

		fields = append(fields, field)
	}

	return
}

// LookupNonZeroFieldsValues lookup for filled fields based on the "v" struct value instance.
// It returns a slice of reflect.Value (same type as `Values`) that can be binded,
// like the end-developer's custom values.
func LookupNonZeroFieldsValues(v reflect.Value, skipUnexported bool) (bindValues []reflect.Value) {
	elem := IndirectValue(v)
	fields := lookupFields(IndirectType(v.Type()), skipUnexported, nil)

	for _, f := range fields {
		if fieldVal := elem.FieldByIndex(f.Index); /*f.Type.Kind() == reflect.Ptr &&*/
		goodVal(fieldVal) && !IsZero(fieldVal) {
			// fmt.Printf("[%d][field index = %d] append to bindValues: %s = %s\n", i, f.Index[0], f.Name, fieldVal.String())
			bindValues = append(bindValues, fieldVal)
		}
	}

	return
}
