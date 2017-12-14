package di

import (
	"errors"
	"reflect"
)

type bindType uint32

const (
	Static  bindType = iota // simple assignable value, a static value.
	Dynamic                 // dynamic value, depends on some input arguments from the caller.
)

type BindObject struct {
	Type  reflect.Type // the Type of 'Value' or the type of the returned 'ReturnValue' .
	Value reflect.Value

	BindType    bindType
	ReturnValue func([]reflect.Value) reflect.Value
}

// ReturnValueBad can be customized to skip func binders that are not allowed, a custom logic.
var ReturnValueBad = func(reflect.Type) bool {
	return false
}

func MakeBindObject(v reflect.Value, goodFunc TypeChecker) (b BindObject, err error) {
	if isFunc(v) {
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
	typ := indirectTyp(fn.Type())

	// invalid if not a func.
	if typ.Kind() != reflect.Func {
		return nil, typ, errBad
	}

	// invalid if not returns one single value.
	if typ.NumOut() != 1 {
		return nil, typ, errBad
	}

	if goodFunc != nil {
		if !goodFunc(typ) {
			return nil, typ, errBad
		}
	}

	outTyp := typ.Out(0)
	zeroOutVal := reflect.New(outTyp).Elem()

	bf := func(ctxValue []reflect.Value) reflect.Value {
		results := fn.Call(ctxValue)
		if len(results) == 0 {
			return zeroOutVal
		}

		v := results[0]
		if !v.IsValid() {
			return zeroOutVal
		}
		return v
	}

	return bf, outTyp, nil
}

func (b *BindObject) IsAssignable(to reflect.Type) bool {
	return equalTypes(b.Type, to)
}

func (b *BindObject) Assign(ctx []reflect.Value, toSetter func(reflect.Value)) {
	if b.BindType == Dynamic {
		toSetter(b.ReturnValue(ctx))
		return
	}
	toSetter(b.Value)
}
