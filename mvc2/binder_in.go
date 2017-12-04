package mvc2

import (
	"reflect"
)

// InputBinder is the result of `MakeBinder`.
// It contains the binder wrapped information, like the
// type that is responsible to bind
// and a function which will accept a context and returns a value of something.
type InputBinder struct {
	BinderType binderType
	BindType   reflect.Type
	BindFunc   func(ctx []reflect.Value) reflect.Value
}

// key = the func input argument index, value is the responsible input binder.
type bindersMap map[int]*InputBinder

// joinBindersMap joins the "m2" to m1 and returns the result, it's the same "m1" map.
// if "m2" is not nil and "m2" is not nil then it loops the "m2"'s keys and sets the values
// to the "m1", if "m2" is not and not empty nil but m1 is nil then "m1" = "m2".
// The result may be nil if the "m1" and "m2" are nil or "m2" is empty and "m1" is nil.
func joinBindersMap(m1, m2 bindersMap) bindersMap {
	if m2 != nil && len(m2) > 0 {
		if m1 == nil {
			m1 = m2
		} else {
			for k, v := range m2 {
				m1[k] = v
			}
		}
	}
	return m1
}

// getBindersForInput returns a map of the responsible binders for the "expected" types,
// which are the expected input parameters' types,
// based on the available "binders" collection.
//
// It returns a map which its key is the index of the "expected" which
// a valid binder for that in's type found,
// the value is the pointer of the responsible `InputBinder`.
//
// Check of "a nothing responsible for those expected types"
// should be done using the `len(m) == 0`.
func getBindersForInput(binders []*InputBinder, expected ...reflect.Type) (m bindersMap) {
	for idx, in := range expected {
		if idx == 0 && isContext(in) {
			// if the first is context then set it directly here.
			m = make(bindersMap)
			m[0] = &InputBinder{
				BindType: contextTyp,
				BindFunc: func(ctxValues []reflect.Value) reflect.Value {
					return ctxValues[0]
				},
			}
			continue
		}

		for _, b := range binders {
			if equalTypes(b.BindType, in) {
				if m == nil {
					m = make(bindersMap)
				}
				// fmt.Printf("set index: %d to type: %s where input type is: %s\n", idx, b.BindType.String(), in.String())
				m[idx] = b
				break
			}
		}
	}

	return m
}

// MustMakeFuncInputBinder calls the `MakeFuncInputBinder` and returns its first result, see its docs.
// It panics on error.
func MustMakeFuncInputBinder(binder interface{}) *InputBinder {
	b, err := MakeFuncInputBinder(binder)
	if err != nil {
		panic(err)
	}
	return b
}

type binderType uint32

const (
	functionType binderType = iota
	serviceType
	invalidType
)

func resolveBinderType(binder interface{}) binderType {
	if binder == nil {
		return invalidType
	}

	return resolveBinderTypeFromKind(reflect.TypeOf(binder).Kind())
}

func resolveBinderTypeFromKind(k reflect.Kind) binderType {
	switch k {
	case reflect.Func:
		return functionType
	case reflect.Struct, reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Array:
		return serviceType
	}

	return invalidType
}

// MakeFuncInputBinder takes a binder function or a struct which contains a "Bind"
// function and returns an `InputBinder`, which Iris uses to
// resolve and set the input parameters when a handler is executed.
//
// The "binder" can have the following form:
// `func(iris.Context) UserViewModel`.
//
// The return type of the "binder" should be a value instance, not a pointer, for your own protection.
// The binder function should return only one value and
// it can accept only one input argument, the Iris' Context (`context.Context` or `iris.Context`).
func MakeFuncInputBinder(binder interface{}) (*InputBinder, error) {
	v := reflect.ValueOf(binder)
	return makeFuncInputBinder(v)
}

func makeFuncInputBinder(fn reflect.Value) (*InputBinder, error) {
	typ := indirectTyp(fn.Type())

	// invalid if not a func.
	if typ.Kind() != reflect.Func {
		return nil, errBad
	}

	// invalid if not returns one single value.
	if typ.NumOut() != 1 {
		return nil, errBad
	}

	// invalid if input args length is not one.
	if typ.NumIn() != 1 {
		return nil, errBad
	}

	// invalid if that single input arg is not a typeof context.Context.
	if !isContext(typ.In(0)) {
		return nil, errBad
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

	return &InputBinder{
		BinderType: functionType,
		BindType:   outTyp,
		BindFunc:   bf,
	}, nil
}
