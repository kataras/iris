// Package recover provides recovery for specific routes or for the whole app via middleware. See _examples/recover
package recover

import (
	"fmt"
	"net/http/httputil"
	"runtime"

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

				var stacktrace string
				for i := 1; ; i++ {
					_, f, l, got := runtime.Caller(i)
					if !got {
						break
					}

					stacktrace += fmt.Sprintf("%s:%d\n", f, l)
				}

				// when stack finishes
				logMessage := fmt.Sprintf("Recovered from a route's Handler('%s')\n", ctx.HandlerName())
				logMessage += fmt.Sprintf("At Request: %s\n", getRequestLogs(ctx))
				logMessage += fmt.Sprintf("Trace: %s\n", err)
				logMessage += fmt.Sprintf("\n%s", stacktrace)
				ctx.Application().Logger().Warn(logMessage)

				ctx.StopWithStatus(500)
			}
		}()

		ctx.Next()
	}
}
