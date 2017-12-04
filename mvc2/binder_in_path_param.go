package mvc2

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

func getInputArgsFromFunc(funcTyp reflect.Type) []reflect.Type {
	n := funcTyp.NumIn()
	funcIn := make([]reflect.Type, n, n)
	for i := 0; i < n; i++ {
		funcIn[i] = funcTyp.In(i)
	}
	return funcIn
}

func getPathParamsForInput(params []macro.TemplateParam, funcIn ...reflect.Type) (values []reflect.Value) {
	if len(funcIn) == 0 || len(params) == 0 {
		return
	}

	funcInIdx := 0
	// it's a valid param type.
	for _, p := range params {
		in := funcIn[funcInIdx]
		paramType := p.Type
		paramName := p.Name

		// fmt.Printf("%s input arg type vs %s param type\n", in.Kind().String(), p.Type.Kind().String())
		if p.Type.Assignable(in.Kind()) {

			// b = append(b, &InputBinder{
			// 	BindType: in, // or p.Type.Kind, should be the same.
			// 	BindFunc: func(ctx []reflect.Value) reflect.Value {
			// 		// I don't like this ctx[0].Interface(0)
			// 		// it will be slow, and silly because we have ctx already
			// 		// before the bindings at serve-time, so we will create
			// 		// a func for each one of the param types, they are just 4 so
			// 		// it worths some dublications.
			// 		return getParamValueFromType(ctx[0].Interface(), paramType, paramName)
			// 	},
			// })

			var fn interface{}

			if paramType == ast.ParamTypeInt {
				fn = func(ctx context.Context) int {
					v, _ := ctx.Params().GetInt(paramName)
					return v
				}
			} else if paramType == ast.ParamTypeLong {
				fn = func(ctx context.Context) int64 {
					v, _ := ctx.Params().GetInt64(paramName)
					return v
				}

			} else if paramType == ast.ParamTypeBoolean {
				fn = func(ctx context.Context) bool {
					v, _ := ctx.Params().GetBool(paramName)
					return v
				}

			} else {
				// string, path...
				fn = func(ctx context.Context) string {
					return ctx.Params().Get(paramName)
				}
			}

			fmt.Printf("binder_in_path_param.go: bind path param func for paramName = '%s' and paramType = '%s'\n", paramName, paramType.String())
			values = append(values, reflect.ValueOf(fn))

			// inputBinder, err := MakeFuncInputBinder(fn)
			// if err != nil {
			// 	fmt.Printf("err on make func binder: %v\n", err.Error())
			// 	continue
			// }

			// if m == nil {
			// 	m = make(bindersMap, 0)
			// }

			// // fmt.Printf("set param input binder for func arg index: %d\n", funcInIdx)
			// m[funcInIdx] = inputBinder
		}

		funcInIdx++
	}

	return
	// return m
}

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
