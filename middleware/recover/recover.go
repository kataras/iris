// Package recover provides recovery for specific routes or for the whole app via middleware. See _examples/recover
package recover

import (
	"fmt"
	"net/http/httputil"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recover.*", "iris.recover")
}

func getRequestLogs(ctx *context.Context) string {
	rawReq, _ := httputil.DumpRequest(ctx.Request(), false)
	return string(rawReq)
}

// New returns a new recover middleware,
// it recovers from panics and logs
// the panic message to the application's logger "Warn" level.
func New() context.Handler {
	return func(ctx *context.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() { // handled by other middleware.
					return
				}

				var callers []string
				for i := 1; ; i++ {
					_, file, line, got := runtime.Caller(i)
					if !got {
						break
					}

					callers = append(callers, fmt.Sprintf("%s:%d", file, line))
				}

				// when stack finishes
				logMessage := fmt.Sprintf("Recovered from a route's Handler('%s')\n", ctx.HandlerName())
				logMessage += fmt.Sprint(getRequestLogs(ctx))
				logMessage += fmt.Sprintf("%s\n", err)
				logMessage += fmt.Sprintf("%s\n", strings.Join(callers, "\n"))
				ctx.Application().Logger().Warn(logMessage)

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
				ctx.StopWithPlainError(500, &context.ErrPanicRecovery{
					Cause:              err,
					Callers:            callers,
					Stack:              debug.Stack(),
					RegisteredHandlers: handlersFileLines,
					CurrentHandler:     currentHandlerFileLine,
				})
			}
		}()

		ctx.Next()
	}
}
