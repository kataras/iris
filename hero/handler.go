package hero

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v12/context"
)

type (
	ErrorHandler interface {
		HandleError(context.Context, error)
	}
	ErrorHandlerFunc func(context.Context, error)
)

func (fn ErrorHandlerFunc) HandleError(ctx context.Context, err error) {
	fn(ctx, err)
}

var (
	// DefaultErrStatusCode is the default error status code (400)
	// when the response contains a non-nil error or a request-scoped binding error occur.
	DefaultErrStatusCode = 400

	// DefaultErrorHandler is the default error handler which is fired
	// when a function returns a non-nil error or a request-scoped dependency failed to binded.
	DefaultErrorHandler = ErrorHandlerFunc(func(ctx context.Context, err error) {
		if status := ctx.GetStatusCode(); status == 0 || !context.StatusCodeNotSuccessful(status) {
			ctx.StatusCode(DefaultErrStatusCode)
		}

		ctx.WriteString(err.Error())
		ctx.StopExecution()
	})
)

var (
	// ErrSeeOther may be returned from a dependency handler to skip a specific dependency
	// based on custom logic.
	ErrSeeOther = fmt.Errorf("see other")
	// ErrStopExecution may be returned from a dependency handler to stop
	// and return the execution of the function without error (it calls ctx.StopExecution() too).
	// It may be occurred from request-scoped dependencies as well.
	ErrStopExecution = fmt.Errorf("stop execution")
)

func makeHandler(fn interface{}, c *Container) context.Handler {
	if fn == nil {
		panic("makeHandler: function is nil")
	}

	// 0. A normal handler.
	if handler, ok := isHandler(fn); ok {
		return handler
	}

	// 1. A handler which returns just an error, handle it faster.
	if handlerWithErr, ok := isHandlerWithError(fn); ok {
		return func(ctx context.Context) {
			if err := handlerWithErr(ctx); err != nil {
				c.GetErrorHandler(ctx).HandleError(ctx, err)
			}
		}
	}

	v := valueOf(fn)
	numIn := v.Type().NumIn()

	bindings := getBindingsForFunc(v, c.Dependencies, c.ParamStartIndex)

	return func(ctx context.Context) {
		inputs := make([]reflect.Value, numIn)

		for _, binding := range bindings {
			input, err := binding.Dependency.Handle(ctx, binding.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				} else if err == ErrStopExecution {
					ctx.StopExecution()
					return // return without error.
				}

				c.GetErrorHandler(ctx).HandleError(ctx, err)
				return
			}

			inputs[binding.Input.Index] = input
		}

		outputs := v.Call(inputs)
		if err := dispatchFuncResult(ctx, outputs); err != nil {
			c.GetErrorHandler(ctx).HandleError(ctx, err)
		}
	}
}

func isHandler(fn interface{}) (context.Handler, bool) {
	if handler, ok := fn.(context.Handler); ok {
		return handler, ok
	}

	if handler, ok := fn.(func(context.Context)); ok {
		return handler, ok
	}

	return nil, false
}

func isHandlerWithError(fn interface{}) (func(context.Context) error, bool) {
	if handlerWithErr, ok := fn.(func(context.Context) error); ok {
		return handlerWithErr, true
	}

	return nil, false
}
