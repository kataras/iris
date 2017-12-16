package mvc

import (
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/di"
)

var baseControllerTyp = reflect.TypeOf((*BaseController)(nil)).Elem()

func isBaseController(ctrlTyp reflect.Type) bool {
	return ctrlTyp.Implements(baseControllerTyp)
}

var contextTyp = reflect.TypeOf((*context.Context)(nil)).Elem()

func isContext(inTyp reflect.Type) bool {
	return inTyp.Implements(contextTyp)
}

func getInputArgsFromFunc(funcTyp reflect.Type) []reflect.Type {
	n := funcTyp.NumIn()
	funcIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		funcIn[i] = funcTyp.In(i)
	}
	return funcIn
}

var (
	typeChecker = func(fn reflect.Type) bool {
		// valid if that single input arg is a typeof context.Context.
		return fn.NumIn() == 1 && isContext(fn.In(0))
	}

	hijacker = func(fieldOrFuncInput reflect.Type) (*di.BindObject, bool) {
		if !isContext(fieldOrFuncInput) {
			return nil, false
		}

		// this is being used on both func injector and struct injector.
		// if the func's input argument or the struct's field is a type of Context
		// then we can do a fast binding using the ctxValue
		// which is used as slice of reflect.Value, because of the final method's `Call`.
		return &di.BindObject{
			Type:     contextTyp,
			BindType: di.Dynamic,
			ReturnValue: func(ctxValue []reflect.Value) reflect.Value {
				return ctxValue[0]
			},
		}, true
	}
)
