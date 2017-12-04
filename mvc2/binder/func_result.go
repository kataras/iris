package binder

import (
	"errors"
	"reflect"
)

var (
	errBad = errors.New("bad")
)

func makeReturnValue(fn reflect.Value) (func([]reflect.Value) reflect.Value, reflect.Type, error) {
	typ := indirectTyp(fn.Type())

	// invalid if not a func.
	if typ.Kind() != reflect.Func {
		return nil, typ, errBad
	}

	// invalid if not returns one single value.
	if typ.NumOut() != 1 {
		return nil, typ, errBad
	}

	// invalid if input args length is not one.
	if typ.NumIn() != 1 {
		return nil, typ, errBad
	}

	// invalid if that single input arg is not a typeof context.Context.
	if !isContext(typ.In(0)) {
		return nil, typ, errBad
	}

	outTyp := typ.Out(0)
	zeroOutVal := reflect.New(outTyp).Elem()

	bf := func(ctxValue []reflect.Value) reflect.Value {
		// []reflect.Value{reflect.ValueOf(ctx)}
		results := fn.Call(ctxValue) // ctxValue is like that because of; read makeHandler.
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
