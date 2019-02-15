package di

import (
	"errors"
	"reflect"
)

// BindType is the type of a binded object/value, it's being used to
// check if the value is accessible after a function call with a  "ctx" when needed ( Dynamic type)
// or it's just a struct value (a service | Static type).
type BindType uint32

const (
	// Static is the simple assignable value, a static value.
	Static BindType = iota
	// Dynamic returns a value but it depends on some input arguments from the caller,
	// on serve time.
	Dynamic
)

func bindTypeString(typ BindType) string {
	switch typ {
	case Dynamic:
		return "Dynamic"
	default:
		return "Static"
	}
}

// BindObject contains the dependency value's read-only information.
// FuncInjector and StructInjector keeps information about their
// input arguments/or fields, these properties contain a `BindObject` inside them.
type BindObject struct {
	Type  reflect.Type // the Type of 'Value' or the type of the returned 'ReturnValue' .
	Value reflect.Value

	BindType    BindType
	ReturnValue func([]reflect.Value) reflect.Value
}

// MakeBindObject accepts any "v" value, struct, pointer or a function
// and a type checker that is used to check if the fields (if "v.elem()" is struct)
// or the input arguments (if "v.elem()" is func)
// are valid to be included as the final object's dependencies, even if the caller added more
// the "di" is smart enough to select what each "v" needs and what not before serve time.
func MakeBindObject(v reflect.Value, goodFunc TypeChecker) (b BindObject, err error) {
	if IsFunc(v) {
		b.BindType = Dynamic
		b.ReturnValue, b.Type, err = MakeReturnValue(v, goodFunc)
	} else {
		b.BindType = Static
		b.Type = v.Type()
		b.Value = v
	}

	return
}

var errBad = errors.New("bad")

// MakeReturnValue takes any function
// that accept custom values and returns something,
// it returns a binder function, which accepts a slice of reflect.Value
// and returns a single one reflect.Value for that.
// It's being used to resolve the input parameters on a "x" consumer faster.
//
// The "fn" can have the following form:
// `func(myService) MyViewModel`.
//
// The return type of the "fn" should be a value instance, not a pointer, for your own protection.
// The binder function should return only one value.
func MakeReturnValue(fn reflect.Value, goodFunc TypeChecker) (func([]reflect.Value) reflect.Value, reflect.Type, error) {
	typ := IndirectType(fn.Type())

	// invalid if not a func.
	if typ.Kind() != reflect.Func {
		return nil, typ, errBad
	}

	n := typ.NumOut()

	// invalid if not returns one single value or two values but the second is not an error.
	if !(n == 1 || (n == 2 && IsError(typ.Out(1)))) {
		return nil, typ, errBad
	}

	if goodFunc != nil {
		if !goodFunc(typ) {
			return nil, typ, errBad
		}
	}

	firstOutTyp := typ.Out(0)
	firstZeroOutVal := reflect.New(firstOutTyp).Elem()

	bf := func(ctxValue []reflect.Value) reflect.Value {
		results := fn.Call(ctxValue)

		v := results[0]
		if !v.IsValid() { // check the first value, second is error.
			return firstZeroOutVal
		}

		if n == 2 {
			// two, second is always error.
			errVal := results[1]
			if !errVal.IsNil() {
				// error is not nil, do something with it.
				if ctx, ok := ctxValue[0].Interface().(interface {
					StatusCode(int)
					WriteString(string) (int, error)
					StopExecution()
				}); ok {
					ctx.StatusCode(400)
					ctx.WriteString(errVal.Interface().(error).Error())
					ctx.StopExecution()
				}

				return firstZeroOutVal
			}
		}

		// if v.String() == "<interface {} Value>" {
		// 	println("di/object.go: " + v.String())
		// 	// println("di/object.go: because it's interface{} it should be returned as: " + v.Elem().Type().String() + " and its value: " + v.Elem().Interface().(string))
		// 	return v.Elem()
		// }
		return v
	}

	return bf, firstOutTyp, nil
}

// IsAssignable checks if "to" type can be used as "b.Value/ReturnValue".
func (b *BindObject) IsAssignable(to reflect.Type) bool {
	return equalTypes(b.Type, to)
}

// Assign sets the values to a setter, "toSetter" contains the setter, so the caller
// can use it for multiple and different structs/functions as well.
func (b *BindObject) Assign(ctx []reflect.Value, toSetter func(reflect.Value)) {
	if b.BindType == Dynamic {
		toSetter(b.ReturnValue(ctx))
		return
	}
	toSetter(b.Value)
}
