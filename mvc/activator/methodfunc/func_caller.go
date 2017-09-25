package methodfunc

import (
	"reflect"

	"github.com/kataras/iris/context"
)

// buildMethodCall builds the method caller.
// We have repeated code here but it's the only way
// to support more than one input arguments without performance cost compared to previous implementation.
// so it's hard-coded written to check the length of input args and their types.
func buildMethodCall(a *ast) func(ctx context.Context, f reflect.Value) {
	// if accepts one or more parameters.
	if a.dynamic {
		// if one function input argument then call the function
		// by "casting" (faster).
		if l := len(a.paramKeys); l == 1 {
			paramType := a.paramTypes[0]
			paramKey := a.paramKeys[0]

			if paramType == paramTypeInt {
				return func(ctx context.Context, f reflect.Value) {
					v, _ := ctx.Params().GetInt(paramKey)
					f.Interface().(func(int))(v)
				}
			}

			if paramType == paramTypeLong {
				return func(ctx context.Context, f reflect.Value) {
					v, _ := ctx.Params().GetInt64(paramKey)
					f.Interface().(func(int64))(v)
				}

			}

			if paramType == paramTypeBoolean {
				return func(ctx context.Context, f reflect.Value) {
					v, _ := ctx.Params().GetBool(paramKey)
					f.Interface().(func(bool))(v)
				}
			}

			// string, path...
			return func(ctx context.Context, f reflect.Value) {
				f.Interface().(func(string))(ctx.Params().Get(paramKey))
			}

		}

		// if func input arguments are more than one then
		// use the Call method (slower).
		return func(ctx context.Context, f reflect.Value) {
			f.Call(a.paramValues(ctx))
		}
	}

	// if it's static without any receivers then just call it.
	return func(ctx context.Context, f reflect.Value) {
		f.Interface().(func())()
	}
}
