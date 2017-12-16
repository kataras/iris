package mvc

import (
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

// for methods inside a controller.

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
				// fmt.Printf("path_param_binder.go: bind path param func for paramName = '%s' and paramType = '%s'\n", paramName, paramType.String())
				values = append(values, makeFuncParamGetter(paramType, paramName))
			}
		}
	}
	// funcInIdx := 0
	// // it's a valid param type.
	// for _, p := range params {
	// 	in := funcIn[funcInIdx]
	// 	paramType := p.Type
	// 	paramName := p.Name
	// 	// 	fmt.Printf("%s input arg type vs %s param type\n", in.Kind().String(), p.Type.Kind().String())
	// 	if paramType.Assignable(in.Kind()) {
	// 		// fmt.Printf("path_param_binder.go: bind path param func for paramName = '%s' and paramType = '%s'\n", paramName, paramType.String())
	// 		values = append(values, makeFuncParamGetter(paramType, paramName))
	// 	}

	// 	funcInIdx++
	// }

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

// for raw handlers, independent of a controller.

// PathParams is the context's named path parameters, see `PathParamsBinder` too.
type PathParams = context.RequestParams

// PathParamsBinder is the binder which will bind the `PathParams` type value to the specific
// handler's input argument, see `PathParams` as well.
func PathParamsBinder(ctx context.Context) PathParams {
	return *ctx.Params()
}

// PathParam describes a named path parameter, it's the result of the PathParamBinder and the expected
// handler func's input argument's type, see `PathParamBinder` too.
type PathParam struct {
	memstore.Entry
	Empty bool
}

// PathParamBinder is the binder which binds a handler func's input argument to a named path parameter
// based on its name, see `PathParam` as well.
func PathParamBinder(name string) func(ctx context.Context) PathParam {
	return func(ctx context.Context) PathParam {
		e, found := ctx.Params().GetEntry(name)
		if !found {

			// useless check here but it doesn't hurt,
			// useful only when white-box tests run.
			if ctx.Application() != nil {
				ctx.Application().Logger().Warnf(ctx.HandlerName()+": expected parameter name '%s' to be described in the route's path in order to be received by the `ParamBinder`, please fix it.\n The main handler will not be executed for your own protection.", name)
			}

			ctx.StopExecution()
			return PathParam{
				Empty: true,
			}
		}
		return PathParam{e, false}
	}
}
