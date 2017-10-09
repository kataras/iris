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
	// if func input arguments are more than one then
	// use the Call method (slower).
	return func(ctx context.Context, f reflect.Value) {
		DispatchFuncResult(ctx, f.Call(a.paramValues(ctx)))
	}
}
