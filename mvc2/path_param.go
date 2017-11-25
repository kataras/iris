package mvc2

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/memstore"
)

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
