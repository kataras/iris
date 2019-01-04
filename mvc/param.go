package mvc

import (
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/macro"
)

func getPathParamsForInput(params []macro.TemplateParam, funcIn ...reflect.Type) (values []reflect.Value) {
	if len(funcIn) == 0 || len(params) == 0 {
		return
	}

	// consumedParams := make(map[int]bool, 0)
	// for _, in := range funcIn {
	// 	for j, p := range params {
	// 		if _, consumed := consumedParams[j]; consumed {
	// 			continue
	// 		}

	// 		// 	fmt.Printf("%s input arg type vs %s param type\n", in.Kind().String(), p.Type.Kind().String())
	// 		if m := macros.Lookup(p.Type); m != nil && m.GoType == in.Kind() {
	// 			consumedParams[j] = true
	// 			// fmt.Printf("param.go: bind path param func for paramName = '%s' and paramType = '%s'\n", paramName, paramType.String())
	// 			funcDep, ok := context.ParamResolverByKindAndIndex(m.GoType, p.Index)
	// 			//	funcDep, ok := context.ParamResolverByKindAndKey(in.Kind(), paramName)
	// 			if !ok {
	// 				// here we can add a logger about invalid parameter type although it should never happen here
	// 				// unless the end-developer modified the macro/macros with a special type but not the context/ParamResolvers.
	// 				continue
	// 			}
	// 			values = append(values, funcDep)
	// 		}
	// 	}
	// }

	for i, param := range params {
		if len(funcIn) <= i {
			return
		}
		funcDep, ok := context.ParamResolverByTypeAndIndex(funcIn[i], param.Index)
		if !ok {
			continue
		}

		values = append(values, funcDep)
	}

	return
}
