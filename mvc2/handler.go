package mvc2

import (
	"fmt"
	"reflect"

	"github.com/kataras/golog"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator/methodfunc"
)

// checks if "handler" is context.Handler; func(context.Context).
func isContextHandler(handler interface{}) (context.Handler, bool) {
	h, is := handler.(context.Handler)
	if !is {
		fh, is := handler.(func(context.Context))
		if is {
			return fh, is
		}
	}
	return h, is
}

func validateHandler(handler interface{}) error {
	if typ := reflect.TypeOf(handler); !isFunc(typ) {
		return fmt.Errorf("handler expected to be a kind of func but got typeof(%s)", typ.String())
	}
	return nil
}

var (
	contextTyp = reflect.TypeOf(context.NewContext(nil))
	emptyIn    = []reflect.Value{}
)

// MustMakeHandler calls the `MakeHandler` and returns its first resultthe low-level handler), see its docs.
// It panics on error.
func MustMakeHandler(handler interface{}, binders ...interface{}) context.Handler {
	h, err := MakeHandler(handler, binders...)
	if err != nil {
		panic(err)
	}
	return h
}

// MakeHandler accepts a "handler" function which can accept any input that matches
// with the "binders" and any output, that matches the mvc types, like string, int (string,int),
// custom structs, Result(View | Response) and anything that you already know that mvc implementation supports,
// and returns a low-level `context/iris.Handler` which can be used anywhere in the Iris Application,
// as middleware or as simple route handler or party handler or subdomain handler-router.
func MakeHandler(handler interface{}, binders ...interface{}) (context.Handler, error) {
	if err := validateHandler(handler); err != nil {
		golog.Errorf("mvc handler: %v", err)
		return nil, err
	}

	if h, is := isContextHandler(handler); is {
		golog.Warnf("mvc handler: you could just use the low-level API to register a context handler instead")
		return h, nil
	}

	inputBinders := make([]reflect.Value, len(binders), len(binders))

	for i := range binders {
		inputBinders[i] = reflect.ValueOf(binders[i])
	}

	return makeHandler(reflect.ValueOf(handler), inputBinders), nil

	// typ := indirectTyp(reflect.TypeOf(handler))
	// n := typ.NumIn()
	// typIn := make([]reflect.Type, n, n)
	// for i := 0; i < n; i++ {
	// 	typIn[i] = typ.In(i)
	// }

	// m := getBindersForInput(binders, typIn...)
	// if len(m) != n {
	// 	err := fmt.Errorf("input arguments length(%d) of types(%s) and valid binders length(%d) are not equal", n, typIn, len(m))
	// 	golog.Errorf("mvc handler: %v", err)
	// 	return nil, err
	// }

	// return makeHandler(reflect.ValueOf(handler), m), nil
}

func makeHandler(fn reflect.Value, inputBinders []reflect.Value) context.Handler {
	inLen := fn.Type().NumIn()

	if inLen == 0 {
		return func(ctx context.Context) {
			methodfunc.DispatchFuncResult(ctx, fn.Call(emptyIn))
		}
	}

	s := getServicesFor(fn, inputBinders)
	if len(s) == 0 {
		golog.Errorf("mvc handler: input arguments length(%d) and valid binders length(%d) are not equal", inLen, len(s))
		return nil
	}

	n := fn.Type().NumIn()
	// contextIndex := -1
	// if n > 0 {
	// 	if isContext(fn.Type().In(0)) {
	// 		contextIndex = 0
	// 	}
	// }
	return func(ctx context.Context) {
		ctxValue := []reflect.Value{reflect.ValueOf(ctx)}

		in := make([]reflect.Value, n, n)
		// if contextIndex >= 0 {
		// 	in[contextIndex] = ctxValue[0]
		// }
		// ctxValues := []reflect.Value{reflect.ValueOf(ctx)}
		// for k, v := range m {
		// 	in[k] = v.BindFunc(ctxValues)
		// 	if ctx.IsStopped() {
		// 		return
		// 	}
		// }
		// methodfunc.DispatchFuncResult(ctx, fn.Call(in))

		s.FillFuncInput(ctxValue, &in)

		methodfunc.DispatchFuncResult(ctx, fn.Call(in))
	}
}
