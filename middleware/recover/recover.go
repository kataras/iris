package recover

import (
	"fmt"
	"runtime"
	"strconv"

	"gopkg.in/kataras/iris.v6"
)

func getRequestLogs(ctx *iris.Context) string {
	var status, ip, method, path string
	status = strconv.Itoa(ctx.ResponseWriter.StatusCode())
	path = ctx.Path()
	method = ctx.Method()
	ip = ctx.RemoteAddr()
	// the date should be logged by iris' Logger, so we skip them
	return fmt.Sprintf("%v %s %s %s", status, path, method, ip)
}

// New returns a new recover middleware
// it logs to the LoggerOut iris' configuration field if its IsDeveloper configuration field is enabled.
// otherwise it just continues to serve
func New() iris.HandlerFunc {
	return func(ctx *iris.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() {
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
				logMessage := fmt.Sprintf("Recovered from a route's Handler('%s')\n", ctx.GetHandlerName())
				logMessage += fmt.Sprintf("At Request: %s\n", getRequestLogs(ctx))
				logMessage += fmt.Sprintf("Trace: %s\n", err)
				logMessage += fmt.Sprintf("\n%s\n", stacktrace)
				ctx.Log(iris.DevMode, logMessage)

				ctx.StopExecution()
				ctx.EmitError(iris.StatusInternalServerError)

			}
		}()

		ctx.Next()
	}
}
