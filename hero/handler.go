package hero

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v12/context"
)

type (
	// ErrorHandler describes an interface to handle errors per hero handler and its dependencies.
	//
	// Handles non-nil errors return from a hero handler or a controller's method (see `getBindingsFor` and `Handler`)
	// the error may return from a request-scoped dependency too (see `Handler`).
	ErrorHandler interface {
		HandleError(*context.Context, error)
	}
	// ErrorHandlerFunc implements the `ErrorHandler`.
	// It describes the type defnition for an error function handler.
	ErrorHandlerFunc func(*context.Context, error)

	// Code is a special type for status code.
	Code int
)

// HandleError fires when a non-nil error returns from a request-scoped dependency at serve-time or the handler itself.
func (fn ErrorHandlerFunc) HandleError(ctx *context.Context, err error) {
	fn(ctx, err)
}

var (
	// ErrSeeOther may be returned from a dependency handler to skip a specific dependency
	// based on custom logic.
	ErrSeeOther = fmt.Errorf("see other")
	// ErrStopExecution may be returned from a dependency handler to stop
	// and return the execution of the function without error (it calls ctx.StopExecution() too).
	// It may be occurred from request-scoped dependencies as well.
	ErrStopExecution = fmt.Errorf("stop execution")
)

var (
	// DefaultErrStatusCode is the default error status code (400)
	// when the response contains a non-nil error or a request-scoped binding error occur.
	DefaultErrStatusCode = 400

	// DefaultErrorHandler is the default error handler which is fired
	// when a function returns a non-nil error or a request-scoped dependency failed to binded.
	DefaultErrorHandler = ErrorHandlerFunc(func(ctx *context.Context, err error) {
		if err != ErrStopExecution {
			if status := ctx.GetStatusCode(); status == 0 || !context.StatusCodeNotSuccessful(status) {
				ctx.StatusCode(DefaultErrStatusCode)
			}

			_, _ = ctx.WriteString(err.Error())
		}

		ctx.StopExecution()
	})
)

func makeHandler(fn interface{}, c *Container, paramsCount int) context.Handler {
	if fn == nil {
		panic("makeHandler: function is nil")
	}

	// 0. A normal handler.
	if handler, ok := isHandler(fn); ok {
		return handler
	}

	// 1. A handler which returns just an error, handle it faster.
	if handlerWithErr, ok := isHandlerWithError(fn); ok {
		return func(ctx *context.Context) {
			if err := handlerWithErr(ctx); err != nil {
				c.GetErrorHandler(ctx).HandleError(ctx, err)
			}
		}
	}

	v := valueOf(fn)
	typ := v.Type()
	numIn := typ.NumIn()

	bindings := getBindingsForFunc(v, c.Dependencies, paramsCount)
	c.fillReport(context.HandlerName(fn), bindings)

	resultHandler := defaultResultHandler
	for i, lidx := 0, len(c.resultHandlers)-1; i <= lidx; i++ {
		resultHandler = c.resultHandlers[lidx-i](resultHandler)
	}

	return func(ctx *context.Context) {
		inputs := make([]reflect.Value, numIn)

		for _, binding := range bindings {
			input, err := binding.Dependency.Handle(ctx, binding.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}
				// handled inside ErrorHandler.
				// else if err == ErrStopExecution {
				// 	ctx.StopExecution()
				// 	return // return without error.
				// }

				c.GetErrorHandler(ctx).HandleError(ctx, err)
				return
			}

			// If ~an error status code is set or~ execution has stopped
			// from within the dependency (something went wrong while validating the request),
			// then stop everything and let handler fire that status code.
			if ctx.IsStopped() /* || context.StatusCodeNotSuccessful(ctx.GetStatusCode())*/ {
				return
			}

			inputs[binding.Input.Index] = input
		}

		// fmt.Printf("For func: %s | valid input deps length(%d)\n", typ.String(), len(inputs))
		// for idx, in := range inputs {
		// 	fmt.Printf("[%d] (%s) %#+v\n", idx, in.Type().String(), in.Interface())
		// }

		outputs := v.Call(inputs)
		if err := dispatchFuncResult(ctx, outputs, resultHandler); err != nil {
			c.GetErrorHandler(ctx).HandleError(ctx, err)
		}
	}
}

func isHandler(fn interface{}) (context.Handler, bool) {
	if handler, ok := fn.(context.Handler); ok {
		return handler, ok
	}

	if handler, ok := fn.(func(*context.Context)); ok {
		return handler, ok
	}

	return nil, false
}

func isHandlerWithError(fn interface{}) (func(*context.Context) error, bool) {
	if handlerWithErr, ok := fn.(func(*context.Context) error); ok {
		return handlerWithErr, true
	}

	return nil, false
}
