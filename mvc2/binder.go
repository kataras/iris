package mvc2

import (
	"reflect"

	"github.com/kataras/iris/context"
)

// InputBinder is the result of `MakeBinder`.
// It contains the binder wrapped information, like the
// type that is responsible to bind
// and a function which will accept a context and returns a value of something.
type InputBinder struct {
	BindType reflect.Type
	BindFunc func(context.Context) reflect.Value
}

// MustMakeBinder calls the `MakeBinder` and returns its first result, see its docs.
// It panics on error.
func MustMakeBinder(binder interface{}) *InputBinder {
	b, err := MakeBinder(binder)
	if err != nil {
		panic(err)
	}
	return b
}

// MakeBinder takes a binder function or a struct which contains a "Bind"
// function and returns an `InputBinder`, which Iris uses to
// resolve and set the input parameters when a handler is executed.
//
// The "binder" can have the following form:
// `func(iris.Context) UserViewModel`
// and a struct which contains a "Bind" method
// of the same binder form that was described above.
//
// The return type of the "binder" should be a value instance, not a pointer, for your own protection.
// The binder function should return only one value and
// it can accept only one input argument, the Iris' Context (`context.Context` or `iris.Context`).
func MakeBinder(binder interface{}) (*InputBinder, error) {
	v := reflect.ValueOf(binder)

	// check if it's a struct or a pointer to a struct
	// and contains a "Bind" method, if yes use that as the binder func.
	if indirectTyp(v.Type()).Kind() == reflect.Struct {
		if m := v.MethodByName("Bind"); m.IsValid() && m.CanInterface() {
			v = m
		}
	}

	return makeBinder(v)
}

func makeBinder(fn reflect.Value) (*InputBinder, error) {
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

	bf := func(ctx context.Context) reflect.Value {
		results := fn.Call([]reflect.Value{reflect.ValueOf(ctx)})
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
		BindType: outTyp,
		BindFunc: bf,
	}, nil
}

// searchBinders returns a map of the responsible binders for the "expected" types,
// which are the expected input parameters' types,
// based on the available "binders" collection.
//
// It returns a map which its key is the index of the "expected" which
// a valid binder for that in's type found,
// the value is the pointer of the responsible `InputBinder`.
//
// Check of "a nothing responsible for those expected types"
// should be done using the `len(m) == 0`.
func searchBinders(binders []*InputBinder, expected ...reflect.Type) map[int]*InputBinder {
	var m map[int]*InputBinder

	for idx, in := range expected {
		for _, b := range binders {
			// if same type or the result of binder implements the expected in's type.
			if b.BindType == in || (in.Kind() == reflect.Interface && b.BindType.Implements(in)) {
				if m == nil {
					m = make(map[int]*InputBinder)
				}
				// fmt.Printf("set index: %d to type: %s where input type is: %s\n", idx, b.BindType.String(), in.String())
				m[idx] = b
				break
			}
		}
	}

	return m
}
