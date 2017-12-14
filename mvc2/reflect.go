package mvc2

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/pkg/zerocheck"
)

var baseControllerTyp = reflect.TypeOf((*BaseController)(nil)).Elem()

func isBaseController(ctrlTyp reflect.Type) bool {
	return ctrlTyp.Implements(baseControllerTyp)
}

var contextTyp = reflect.TypeOf((*context.Context)(nil)).Elem()

func isContext(inTyp reflect.Type) bool {
	return inTyp.Implements(contextTyp)
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

func isFunc(kindable interface {
	Kind() reflect.Kind
}) bool {
	return kindable.Kind() == reflect.Func
}

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

func getNameOf(typ reflect.Type) string {
	elemTyp := indirectTyp(typ)

	typName := elemTyp.Name()
	pkgPath := elemTyp.PkgPath()
	fullname := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + typName

	return fullname
}

func getInputArgsFromFunc(funcTyp reflect.Type) []reflect.Type {
	n := funcTyp.NumIn()
	funcIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		funcIn[i] = funcTyp.In(i)
	}
	return funcIn
}

// for controller's fields only.
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
	// it's filled with the bind object (as service which means as static value)
	// and it's filled from the lookupFields' caller.
	AnyValue reflect.Value
}

func lookupFields(elemTyp reflect.Type, parentIndex []int) (fields []field) {
	if elemTyp.Kind() != reflect.Struct {
		return
	}

	for i, n := 0, elemTyp.NumField(); i < n; i++ {
		f := elemTyp.Field(i)

		if f.PkgPath != "" {
			continue // skip unexported.
		}

		if indirectTyp(f.Type).Kind() == reflect.Struct &&
			!structFieldIgnored(f) {
			fields = append(fields, lookupFields(f.Type, append(parentIndex, i))...)
			continue
		}

		index := []int{i}
		if len(parentIndex) > 0 {
			index = append(parentIndex, i)
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

func lookupNonZeroFieldsValues(v reflect.Value) (bindValues []reflect.Value) {
	elem := indirectVal(v)
	fields := lookupFields(indirectTyp(v.Type()), nil)
	for _, f := range fields {

		if fieldVal := elem.FieldByIndex(f.Index); f.Type.Kind() == reflect.Ptr && !zerocheck.IsZero(fieldVal) {
			bindValues = append(bindValues, fieldVal)
		}
	}

	return
}
