package mvc

import (
	"fmt"
	"github.com/kataras/iris/mvc/di"
	"reflect"
	"runtime"

	"github.com/kataras/golog"
	"github.com/kataras/iris/context"
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
	if typ := reflect.TypeOf(handler); !di.IsFunc(typ) {
		return fmt.Errorf("handler expected to be a kind of func but got typeof(%s)", typ.String())
	}
	return nil
}

// MustMakeHandler calls the `MakeHandler` and panics on any error.
func MustMakeHandler(handler interface{}, bindValues ...reflect.Value) context.Handler {
	h, err := MakeHandler(handler, bindValues...)
	if err != nil {
		panic(err)
	}

	return h
}

// MakeHandler accepts a "handler" function which can accept any input arguments that match
// with the "bindValues" types and any output result, that matches the mvc types, like string, int (string,int),
// custom structs, Result(View | Response) and anything that you already know that mvc implementation supports,
// and returns a low-level `context/iris.Handler` which can be used anywhere in the Iris Application,
// as middleware or as simple route handler or party handler or subdomain handler-router.
func MakeHandler(handler interface{}, bindValues ...reflect.Value) (context.Handler, error) {
	if err := validateHandler(handler); err != nil {
		return nil, err
	}

	if h, is := isContextHandler(handler); is {
		golog.Warnf("mvc handler: you could just use the low-level API to register a context handler instead")
		return h, nil
	}

	fn := reflect.ValueOf(handler)
	n := fn.Type().NumIn()

	if n == 0 {
		h := func(ctx context.Context) {
			DispatchFuncResult(ctx, fn.Call(emptyIn))
		}

		return h, nil
	}

	funcInjector := di.MakeFuncInjector(fn, hijacker, typeChecker, bindValues...)
	if !funcInjector.Valid {
		pc := fn.Pointer()
		fpc := runtime.FuncForPC(pc)
		callerFileName, callerLineNumber := fpc.FileLine(pc)
		callerName := fpc.Name()

		err := fmt.Errorf("input arguments length(%d) and valid binders length(%d) are not equal for typeof '%s' which is defined at %s:%d by %s",
			n, funcInjector.Length, fn.Type().String(), callerFileName, callerLineNumber, callerName)
		return nil, err
	}

	h := func(ctx context.Context) {
		// in := make([]reflect.Value, n, n)
		// funcInjector.Inject(&in, reflect.ValueOf(ctx))
		// DispatchFuncResult(ctx, fn.Call(in))
		DispatchFuncResult(ctx, funcInjector.Call(reflect.ValueOf(ctx)))
	}

	return h, nil

}
