package mvc

import "reflect"

var baseControllerTyp = reflect.TypeOf((*BaseController)(nil)).Elem()

func isBaseController(ctrlTyp reflect.Type) bool {
	return ctrlTyp.Implements(baseControllerTyp)
}

func getInputArgsFromFunc(funcTyp reflect.Type) []reflect.Type {
	n := funcTyp.NumIn()
	funcIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		funcIn[i] = funcTyp.In(i)
	}
	return funcIn
}
