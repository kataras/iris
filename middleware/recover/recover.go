// Package recover provides recovery for specific routes or for the whole app via middleware. See _examples/recover
package recover

import (
	"fmt"
	"net/http/httputil"
	"runtime"
	"runtime/debug"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recover.*", "iris.recover") // this name won't work because New() is a function that returns a handler.
}

// New returns a new recovery middleware,
// it recovers from panics and logs the
// panic message to the application's logger "Warn" level.
func New() context.Handler {
	return func(ctx *context.Context) {
		defer func() {
			if err := PanicRecoveryError(ctx, recover()); err != nil {
				ctx.StopWithPlainError(500, err)
				ctx.Application().Logger().Warn(err.LogMessage())
			} // else it's already handled.
		}()

		ctx.Next()
	}
}

// PanicRecoveryError returns a new ErrPanicRecovery error.
func PanicRecoveryError(ctx *context.Context, err any) *context.ErrPanicRecovery {
	if recoveryErr, ok := ctx.IsRecovered(); ok {
		// If registered before any other recovery middleware, get its error.
		// Because of defer this will be executed last, after the recovery middleware in this case.
		return recoveryErr
	}

	if err == nil {
		return nil
	} else if ctx.IsStopped() {
		return nil
	}

	var callers []string
	for i := 2; ; /* 1 for New() 2 for NewPanicRecoveryError */ i++ {
		_, file, line, got := runtime.Caller(i)
		if !got {
			break
		}

		callers = append(callers, fmt.Sprintf("%s:%d", file, line))
	}

	// get the list of registered handlers and the
	// handler which panic derived from.
	handlers := ctx.Handlers()
	handlersFileLines := make([]string, 0, len(handlers))
	currentHandlerIndex := ctx.HandlerIndex(-1)
	currentHandlerFileLine := "???"
	for i, h := range ctx.Handlers() {
		file, line := context.HandlerFileLine(h)
		fileline := fmt.Sprintf("%s:%d", file, line)
		handlersFileLines = append(handlersFileLines, fileline)
		if i == currentHandlerIndex {
			currentHandlerFileLine = fileline
		}
	}

	// see accesslog.wasRecovered too.
	recoveryErr := &context.ErrPanicRecovery{
		Cause:                  err,
		Callers:                callers,
		Stack:                  debug.Stack(),
		RegisteredHandlers:     handlersFileLines,
		CurrentHandlerFileLine: currentHandlerFileLine,
		CurrentHandlerName:     ctx.HandlerName(),
		Request:                getRequestLogs(ctx),
	}

	return recoveryErr
}

func getRequestLogs(ctx *context.Context) string {
	rawReq, _ := httputil.DumpRequest(ctx.Request(), false)
	return string(rawReq)
}
