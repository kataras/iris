package methodfunc

import (
	"github.com/kataras/iris/context"
)

// FuncCaller is responsible to call the controller's function
// which is responsible
// for that request for this http method.
type FuncCaller interface {
	// MethodCall fires the actual handler.
	// The "ctx" is the current context, helps us to get any path parameter's values.
	//
	// The "f" is the controller's function which is responsible
	// for that request for this http method.
	// That function can accept one parameter.
	//
	// The default callers (and the only one for now)
	// are pre-calculated by the framework.
	MethodCall(ctx context.Context, f interface{})
}

type callerFunc func(ctx context.Context, f interface{})

func (c callerFunc) MethodCall(ctx context.Context, f interface{}) {
	c(ctx, f)
}

func resolveCaller(p pathInfo) callerFunc {
	// if it's standard `Get`, `Post` without parameters.
	if p.ParamType == "" {
		return func(ctx context.Context, f interface{}) {
			f.(func())()
		}
	}

	// remember,
	// the router already checks for the correct type,
	// we did pre-calculate everything
	// and now we will pre-calculate the method caller itself as well.

	if p.ParamType == paramTypeInt {
		return func(ctx context.Context, f interface{}) {
			paramValue, _ := ctx.Params().GetInt(paramName)
			f.(func(int))(paramValue)
		}
	}

	if p.ParamType == paramTypeLong {
		return func(ctx context.Context, f interface{}) {
			paramValue, _ := ctx.Params().GetInt64(paramName)
			f.(func(int64))(paramValue)
		}
	}

	if p.ParamType == paramTypeBoolean {
		return func(ctx context.Context, f interface{}) {
			paramValue, _ := ctx.Params().GetBool(paramName)
			f.(func(bool))(paramValue)
		}
	}

	// else it's string or path, both of them are simple strings.
	return func(ctx context.Context, f interface{}) {
		paramValue := ctx.Params().Get(paramName)
		f.(func(string))(paramValue)
	}
}
