package hero

import (
	"reflect"

	"github.com/kataras/iris/hero/di"
)

func init() {
	di.DefaultHijacker = func(fieldOrFuncInput reflect.Type) (*di.BindObject, bool) {
		// if IsExpectingStore(fieldOrFuncInput) {
		// 	return &di.BindObject{
		// 		Type:     memstoreTyp,
		// 		BindType: di.Dynamic,
		// 		ReturnValue: func(ctxValue []reflect.Value) reflect.Value {
		// 			// return ctxValue[0].MethodByName("Params").Call(di.EmptyIn)[0]
		// 			return ctxValue[0].MethodByName("Params").Call(di.EmptyIn)[0].Field(0) // the Params' memstore.Store.
		// 		},
		// 	}, true
		// }

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
			ReturnValue: func(ctxValue []reflect.Value) reflect.Value {
				return ctxValue[0]
			},
		}, true
	}

	di.DefaultTypeChecker = func(fn reflect.Type) bool {
		// valid if that single input arg is a typeof context.Context.
		return fn.NumIn() == 1 && IsContext(fn.In(0))
	}
}
