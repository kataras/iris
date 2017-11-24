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

func makeHandler(handler interface{}, binders []*InputBinder) context.Handler {
	if err := validateHandler(handler); err != nil {
		golog.Errorf("mvc handler: %v", err)
		return nil
	}

	if h, is := isContextHandler(handler); is {
		golog.Warnf("mvc handler: you could just use the low-level API to register a context handler instead")
		return h
	}

	typ := indirectTyp(reflect.TypeOf(handler))
	n := typ.NumIn()
	typIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		typIn[i] = typ.In(i)
	}

	m := getBindersForInput(binders, typIn...)
	if len(m) != n {
		golog.Errorf("mvc handler: input arguments length(%d) and valid binders length(%d) are not equal", n, len(m))
		return nil
	}

	hasIn := len(m) > 0
	fn := reflect.ValueOf(handler)

	return func(ctx context.Context) {
		if !hasIn {
			methodfunc.DispatchFuncResult(ctx, fn.Call(emptyIn))
			return
		}

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
		}
		methodfunc.DispatchFuncResult(ctx, fn.Call(in))
	}
}
