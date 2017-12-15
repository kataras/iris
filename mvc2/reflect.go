package mvc2

import (
	"reflect"

	"github.com/kataras/iris/context"
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
