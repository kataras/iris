package hero

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/hero/di"

	"github.com/kataras/golog"
)

var (
	contextTyp = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// IsContext returns true if the "inTyp" is a type of Context.
func IsContext(inTyp reflect.Type) bool {
	return inTyp.Implements(contextTyp)
}

// checks if "handler" is context.Handler: func(context.Context).
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

// makeHandler accepts a "handler" function which can accept any input arguments that match
// with the "values" types and any output result, that matches the hero types, like string, int (string,int),
// custom structs, Result(View | Response) and anything that you can imagine,
// and returns a low-level `context/iris.Handler` which can be used anywhere in the Iris Application,
// as middleware or as simple route handler or party handler or subdomain handler-router.
func makeHandler(handler interface{}, values ...reflect.Value) (context.Handler, error) {
	if err := validateHandler(handler); err != nil {
		return nil, err
	}

	if h, is := isContextHandler(handler); is {
		golog.Warnf("the standard API to register a context handler could be used instead")
		return h, nil
	}

	fn := reflect.ValueOf(handler)
	n := fn.Type().NumIn()

	if n == 0 {
		h := func(ctx context.Context) {
			DispatchFuncResult(ctx, fn.Call(di.EmptyIn))
		}

		return h, nil
	}

	funcInjector := di.Func(fn, values...)
	valid := funcInjector.Length == n

	if !valid {
		// is invalid when input len and values are not match
		// or their types are not match, we will take look at the
		// second statement, here we will re-try it
		// using binders for path parameters: string, int, int64, uint8, uint64, bool and so on.
		// We don't have access to the path, so neither to the macros here,
		// but in mvc. So we have to do it here.
		if valid = funcInjector.Retry(new(params).resolve); !valid {
			pc := fn.Pointer()
			fpc := runtime.FuncForPC(pc)
			callerFileName, callerLineNumber := fpc.FileLine(pc)
			callerName := fpc.Name()

			err := fmt.Errorf("input arguments length(%d) and valid binders length(%d) are not equal for typeof '%s' which is defined at %s:%d by %s",
				n, funcInjector.Length, fn.Type().String(), callerFileName, callerLineNumber, callerName)
			return nil, err
		}
	}

	h := func(ctx context.Context) {
		// in := make([]reflect.Value, n, n)
		// funcInjector.Inject(&in, reflect.ValueOf(ctx))
		// DispatchFuncResult(ctx, fn.Call(in))
		DispatchFuncResult(ctx, funcInjector.Call(reflect.ValueOf(ctx)))
	}

	return h, nil

}
