package mvc

import (
	"reflect"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/macro"
)

func getPathParamsForInput(startParamIndex int, params []macro.TemplateParam, funcIn ...reflect.Type) (values []reflect.Value) {
	if len(funcIn) == 0 || len(params) == 0 {
		return
	}

	consumed := make(map[int]struct{})
	for _, in := range funcIn {
		for j, param := range params {
			if _, ok := consumed[j]; ok {
				continue
			}

			funcDep, ok := context.ParamResolverByTypeAndIndex(in, startParamIndex+param.Index)
			if !ok {
				continue
			}

			values = append(values, funcDep)
			consumed[j] = struct{}{}
			break
		}
	}

	return
}
