package hero

import (
	"reflect"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/hero/di"
)

func init() {
	di.DefaultHijacker = func(fieldOrFuncInput reflect.Type) (*di.BindObject, bool) {
		if !IsContext(fieldOrFuncInput) {
			return nil, false
		}
		// this is being used on both func injector and struct injector.
		// if the func's input argument or the struct's field is a type of Context
		// then we can do a fast binding using the ctxValue
		// which is used as slice of reflect.Value, because of the final method's `Call`.
		return &di.BindObject{
			Type:     contextTyp,
			BindType: di.Dynamic,
			ReturnValue: func(ctx context.Context) reflect.Value {
				return ctx.ReflectValue()[0]
			},
		}, true
	}

	di.DefaultTypeChecker = func(fn reflect.Type) bool {
		// valid if that single input arg is a typeof context.Context
		// or first argument is context.Context and second argument is a variadic, which is ignored (i.e new sessions#Start).
		return (fn.NumIn() == 1 || (fn.NumIn() == 2 && fn.IsVariadic())) && IsContext(fn.In(0))
	}

	di.DefaultErrorHandler = di.ErrorHandlerFunc(func(ctx context.Context, err error) {
		if err == nil {
			return
		}

		ctx.StatusCode(400)
		ctx.WriteString(err.Error())
		ctx.StopExecution()
	})
}
