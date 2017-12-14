package mvc2

import (
	"github.com/kataras/di"
	"reflect"
)

var (
	typeChecker = func(fn reflect.Type) bool {
		// invalid if that single input arg is not a typeof context.Context.
		return isContext(fn.In(0))
	}

	hijacker = func(fieldOrFuncInput reflect.Type) (*di.BindObject, bool) {
		if isContext(fieldOrFuncInput) {
			return newContextBindObject(), true
		}
		return nil, false
	}
)

// newContextBindObject is being used on both targetFunc and targetStruct.
// if the func's input argument or the struct's field is a type of Context
// then we can do a fast binding using the ctxValue
// which is used as slice of reflect.Value, because of the final method's `Call`.
func newContextBindObject() *di.BindObject {
	return &di.BindObject{
		Type:     contextTyp,
		BindType: di.Dynamic,
		ReturnValue: func(ctxValue []reflect.Value) reflect.Value {
			return ctxValue[0]
		},
	}
}
