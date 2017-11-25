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
func MustMakeHandler(handler interface{}, binders []*InputBinder) context.Handler {
	h, err := MakeHandler(handler, binders)
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
func MakeHandler(handler interface{}, binders []*InputBinder) (context.Handler, error) {
	if err := validateHandler(handler); err != nil {
		golog.Errorf("mvc handler: %v", err)
		return nil, err
	}

	if h, is := isContextHandler(handler); is {
		golog.Warnf("mvc handler: you could just use the low-level API to register a context handler instead")
		return h, nil
	}

	typ := indirectTyp(reflect.TypeOf(handler))
	n := typ.NumIn()
	typIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		typIn[i] = typ.In(i)
	}

	m := getBindersForInput(binders, typIn...)

	/*
		// no f. this, it's too complicated and it will be harder to maintain later on:
		// the only case that these are not equal is when
		// binder returns a slice and input contains one or more inputs.
	*/
	if len(m) != n {
		err := fmt.Errorf("input arguments length(%d) of types(%s) and valid binders length(%d) are not equal", n, typIn, len(m))
		golog.Errorf("mvc handler: %v", err)
		return nil, err
	}

	hasIn := len(m) > 0
	fn := reflect.ValueOf(handler)

	// if has no input to bind then execute the "handler" using the mvc style
	// for any output parameters.
	if !hasIn {
		return func(ctx context.Context) {
			methodfunc.DispatchFuncResult(ctx, fn.Call(emptyIn))
		}, nil
	}

	return func(ctx context.Context) {
		// we could use other tricks for "in"
		// here but let's stick to that which is clearly
		// that it doesn't keep any previous state
		// and it allocates exactly what we need,
		// so we can set via index instead of append.
		// The other method we could use is to
		// declare the in on the build state (before the return)
		// and use in[0:0] with append later on.
		in := make([]reflect.Value, n, n)
		ctxValues := []reflect.Value{reflect.ValueOf(ctx)}
		for k, v := range m {
			in[k] = v.BindFunc(ctxValues)
			/*
				// no f. this, it's too complicated and it will be harder to maintain later on:
				// now an additional check if it's array and has more inputs of the same type
				// and all these results to the expected inputs.
				// 																		   n-1: if has more to set.
				result := v.BindFunc(ctxValues)
				if isSliceAndExpectedItem(result.Type(), in, k) {
					// if kind := result.Kind(); (kind == reflect.Slice || kind == reflect.Array) && n-1 > k {
					prev := 0
					for j, nn := 1, result.Len(); j < nn; j++ {
						item := result.Slice(prev, j)
						prev++
						// remember; we already set the inputs type, so we know
						// what the function expected to have.
						if !equalTypes(item.Type(), in[k+1].Type()) {
							break
						}

						in[k+1] = item
					}
				} else {
					in[k] = result
				}
			*/

			if ctx.IsStopped() {
				return
			}
		}
		methodfunc.DispatchFuncResult(ctx, fn.Call(in))
	}, nil
}
