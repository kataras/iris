package hero

import (
	"net"
	"reflect"

	"github.com/kataras/iris/v12/context"
)

func valueOf(v interface{}) reflect.Value {
	if val, ok := v.(reflect.Value); ok {
		// check if it's already a reflect.Value.
		return val
	}

	return reflect.ValueOf(v)
}

// indirectType returns the value of a pointer-type "typ".
// If "typ" is a pointer, array, chan, map or slice it returns its Elem,
// otherwise returns the "typ" as it is.
func indirectType(typ reflect.Type) reflect.Type {
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

func isFunc(kindable interface{ Kind() reflect.Kind }) bool {
	return kindable.Kind() == reflect.Func
}

func isStructValue(v reflect.Value) bool {
	return indirectType(v.Type()).Kind() == reflect.Struct
}

// isBuiltin reports whether a reflect.Value is a builtin type
func isBuiltinValue(v reflect.Value) bool {
	switch v.Type().Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Chan,
		reflect.Map,
		reflect.Slice,
		reflect.String:
		return true
	default:
		return false
	}
}

var (
	inputTyp = reflect.TypeOf((*Input)(nil))
	errTyp   = reflect.TypeOf((*error)(nil)).Elem()

	ipTyp = reflect.TypeOf(net.IP{})
)

// isError returns true if "typ" is type of `error`.
func isError(typ reflect.Type) bool {
	return typ.Implements(errTyp)
}

func toError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}

	return v.Interface().(error)
}

var contextType = reflect.TypeOf((*context.Context)(nil))

// isContext returns true if the "typ" is a type of Context.
func isContext(typ reflect.Type) bool {
	return typ == contextType
}

var errorHandlerTyp = reflect.TypeOf((*ErrorHandler)(nil)).Elem()

func isErrorHandler(typ reflect.Type) bool {
	return typ.Implements(errorHandlerTyp)
}

var emptyValue reflect.Value

func equalTypes(binding reflect.Type, input reflect.Type) bool {
	if binding == input {
		return true
	}

	// fmt.Printf("got: %s expected: %s\n", got.String(), expected.String())
	// if accepts an interface, check if the given "got" type does
	// implement this "expected" user handler's input argument.
	if input.Kind() == reflect.Interface {
		// fmt.Printf("expected interface = %s and got to set on the arg is: %s\n", binding.String(), input.String())
		// return input.Implements(binding)
		return binding.AssignableTo(input)
	}

	// dependency: func(...) interface{} { return "string" }
	// expected input: string.
	if binding.Kind() == reflect.Interface {
		return input.AssignableTo(binding)
	}

	return false
}

func structFieldIgnored(f reflect.StructField) bool {
	if !f.Anonymous {
		return true // if not anonymous(embedded), ignore it.
	}

	if s := f.Tag.Get("ignore"); s == "true" {
		return true
	}

	if s := f.Tag.Get("stateless"); s == "true" {
		return true
	}

	return false
}

// all except non-zero.
func lookupFields(elem reflect.Value, skipUnexported bool, onlyZeros bool, parentIndex []int) (fields []reflect.StructField, stateless int) {
	// Note: embedded pointers are not supported.
	// elem = reflect.Indirect(elem)
	elemTyp := elem.Type()
	if elemTyp.Kind() == reflect.Pointer {
		return
	}

	for i, n := 0, elem.NumField(); i < n; i++ {
		field := elemTyp.Field(i)
		fieldValue := elem.Field(i)

		// embed any fields from other structs.
		if indirectType(field.Type).Kind() == reflect.Struct {
			if structFieldIgnored(field) {
				stateless++ // don't skip the loop yet, e.g. iris.Context.
			} else {
				structFields, statelessN := lookupFields(fieldValue, skipUnexported, onlyZeros, append(parentIndex, i))
				stateless += statelessN
				fields = append(fields, structFields...)
				continue
			}
		}

		if onlyZeros && !isZero(fieldValue) {
			continue
		}

		// skip unexported fields here.
		if isExported := field.PkgPath == ""; skipUnexported && !isExported {
			continue
		}

		index := []int{i}
		if len(parentIndex) > 0 {
			index = append(parentIndex, i)
		}

		tmp := make([]int, len(index))
		copy(tmp, index)
		field.Index = tmp

		fields = append(fields, field)
	}

	return
}

func lookupNonZeroFieldValues(elem reflect.Value) (nonZeroFields []reflect.StructField) {
	fields, _ := lookupFields(elem, true, false, nil)
	for _, f := range fields {
		if structFieldIgnored(f) {
			continue // re-check here for ignored struct fields so we don't include them on dependencies. Non-zeroes fields can be static, even if they are functions.
		}
		if fieldVal := elem.FieldByIndex(f.Index); goodVal(fieldVal) && !isZero(fieldVal) {
			/* && f.Type.Kind() == reflect.Ptr &&*/
			nonZeroFields = append(nonZeroFields, f)
		}
	}

	return
}

// isZero returns true if a value is nil.
// Remember; fields to be checked should be exported otherwise it returns false.
// Notes for users:
// Boolean's zero value is false, even if not set-ed.
// UintXX are not zero on 0 because they are pointers to.
func isZero(v reflect.Value) bool {
	// switch v.Kind() {
	// case reflect.Struct:
	// 	zero := true
	// 	for i := 0; i < v.NumField(); i++ {
	// 		f := v.Field(i)
	// 		if f.Type().PkgPath() != "" {
	// 			continue // unexported.
	// 		}
	// 		zero = zero && isZero(f)
	// 	}

	// 	if typ := v.Type(); typ != nil && v.IsValid() {
	// 		f, ok := typ.MethodByName("IsZero")
	// 		// if not found
	// 		// if has input arguments (1 is for the value receiver, so > 1 for the actual input args)
	// 		// if output argument is not boolean
	// 		// then skip this IsZero user-defined function.
	// 		if !ok || f.Type.NumIn() > 1 || f.Type.NumOut() != 1 && f.Type.Out(0).Kind() != reflect.Bool {
	// 			return zero
	// 		}

	// 		method := v.Method(f.Index)
	// 		// no needed check but:
	// 		if method.IsValid() && !method.IsNil() {
	// 			// it shouldn't panic here.
	// 			zero = method.Call([]reflect.Value{})[0].Interface().(bool)
	// 		}
	// 	}

	// 	return zero
	// case reflect.Func, reflect.Map, reflect.Slice:
	// 	return v.IsNil()
	// case reflect.Array:
	// 	zero := true
	// 	for i := 0; i < v.Len(); i++ {
	// 		zero = zero && isZero(v.Index(i))
	// 	}
	// 	return zero
	// }
	// if not any special type then use the reflect's .Zero
	// usually for fields, but remember if it's boolean and it's false
	// then it's zero, even if set-ed.

	if !v.CanInterface() {
		// if can't interface, i.e return value from unexported field or method then return false
		return false
	}

	if v.Type() == ipTyp {
		return len(v.Interface().(net.IP)) == 0
	}

	// zero := reflect.Zero(v.Type())
	// return v.Interface() == zero.Interface()

	return v.IsZero()
}

// IsNil same as `reflect.IsNil` but a bit safer to use, returns false if not a correct type.
func isNil(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
