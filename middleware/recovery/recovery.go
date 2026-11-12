// Package recovery turns a panic in a handler into a 500 response instead of a
// crashed server.
//
// It has no settings, so following the plain-handler rule its entry point is the
// `Handler` function itself: register it first with app.UseRouter, or on the
// specific routes that need it.
package recovery

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"runtime"
	"runtime/debug"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recovery.*", "iris.recovery")
}

// Handler returns a new recovery middleware,
// it recovers from panics and logs the
// panic message to the application's logger "Warn" level.
//
// See `WithErrorHandler` to set a custom error handler.
func Handler(ctx *context.Context) {
	defer func() {
		rec := recover()

		// Note(@kataras): http.ErrAbortHandler is not a panic to recover from.
		//
		// It is the standard library's deliberate signal that a handler is
		// abandoning the response; net/http understands it and closes the
		// connection quietly. Swallowing it logged a full stack trace and turned
		// an intentional abort into a 500.
		if rec == http.ErrAbortHandler {
			panic(rec)
		}

		if err := PanicRecoveryError(ctx, rec); err != nil {
			handleError(ctx, err)
		} // else it's already handled.
	}()

	ctx.Next()
}

// maxCallers bounds how many stack frames are recorded for a panic.
//
// The walk used to be unbounded and formatted every frame, then called
// debug.Stack() on top - a large allocation at exactly the moment the process is
// least healthy, and on a deep-recursion panic an enormous one.
const maxCallers = 64

// PanicRecoveryError returns a new ErrPanicRecovery error.
func PanicRecoveryError(ctx *context.Context, err any) *context.ErrPanicRecovery {
	if err == nil || ctx.IsStopped() {
		return nil
	}

	if recoveryErr, ok := ctx.IsRecovered(); ok {
		// If registered before any other recovery middleware, get its error.
		// Because of defer this will be executed last, after the recovery middleware in this case.
		return recoveryErr
	}

	callers := make([]string, 0, 16)
	for i := 2; len(callers) < maxCallers; /* 1 for Handler 2 for NewPanicRecoveryError */ i++ {
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

// RedactedRequestHeaders are the request headers whose values getRequestLogs
// replaces with "[redacted]" before the dump reaches ErrPanicRecovery.
//
// The dump ends up in the server log and, through ErrPanicRecovery.Error, in
// the string any custom error handler may render, so credentials must not
// travel in it. Header names are matched case-insensitively.
var RedactedRequestHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"Set-Cookie",
	"X-Api-Key",
	"X-Auth-Token",
	"X-Csrf-Token",
}

func getRequestLogs(ctx *context.Context) string {
	r := ctx.Request()

	// Dump a shallow copy so redaction cannot disturb the live request; the
	// headers are cloned because DumpRequest reads the map as given.
	redacted := *r
	redacted.Header = r.Header.Clone()
	for _, name := range RedactedRequestHeaders {
		if len(redacted.Header.Values(name)) > 0 {
			redacted.Header.Set(name, "[redacted]")
		}
	}

	rawReq, _ := httputil.DumpRequest(&redacted, false)
	return string(rawReq)
}

// WithErrorHandler returns a middleware which sets a custom recover error handler for the current request.
func WithErrorHandler(errorHandler context.ErrorHandler) context.Handler {
	return func(ctx *context.Context) {
		setErrorHandler(ctx, errorHandler)
		ctx.Next()
	}
}

var errorHandlerContextKey = "iris.recovery.error_handler"

func setErrorHandler(ctx *context.Context, handler context.ErrorHandler) {
	ctx.Values().Set(errorHandlerContextKey, handler)
}

// getExceedHandler returns the handler for the current request.
func getErrorHandler(ctx *context.Context) context.ErrorHandler {
	if errorHandler, ok := ctx.ValueAs[context.ErrorHandler](errorHandlerContextKey); ok {
		return errorHandler
	}

	return nil
}

func handleError(ctx *context.Context, err error) {
	errorHandler := getErrorHandler(ctx)
	if errorHandler == nil {
		ctx.StopWithPlainError(500, err)
		ctx.Application().Logger().Warn(err.(*context.ErrPanicRecovery).LogMessage())
		return
	}
	errorHandler(ctx, err)
}
