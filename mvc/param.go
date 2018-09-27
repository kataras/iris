package mvc

import (
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

func getPathParamsForInput(params []macro.TemplateParam, funcIn ...reflect.Type) (values []reflect.Value) {
	if len(funcIn) == 0 || len(params) == 0 {
		return
	}

	consumedParams := make(map[int]bool, 0)
	for _, in := range funcIn {
		for j, p := range params {
			if _, consumed := consumedParams[j]; consumed {
				continue
			}
			paramType := p.Type
			paramName := p.Name
			// 	fmt.Printf("%s input arg type vs %s param type\n", in.Kind().String(), p.Type.Kind().String())
			if paramType.Assignable(in.Kind()) {
				consumedParams[j] = true
				// fmt.Printf("param.go: bind path param func for paramName = '%s' and paramType = '%s'\n", paramName, paramType.String())
				values = append(values, makeFuncParamGetter(paramType, paramName))
			}
		}
	}

	return
}

func makeFuncParamGetter(paramType ast.ParamType, paramName string) reflect.Value {
	var fn interface{}

	switch paramType {
	case ast.ParamTypeInt:
		fn = func(ctx context.Context) int {
			v, _ := ctx.Params().GetInt(paramName)
			return v
		}
	case ast.ParamTypeLong:
		fn = func(ctx context.Context) int64 {
			v, _ := ctx.Params().GetInt64(paramName)
			return v
		}
	case ast.ParamTypeBoolean:
		fn = func(ctx context.Context) bool {
			v, _ := ctx.Params().GetBool(paramName)
			return v
		}
	default:
		// string, path...
		fn = func(ctx context.Context) string {
			return ctx.Params().Get(paramName)
		}
	}

	return reflect.ValueOf(fn)
}
