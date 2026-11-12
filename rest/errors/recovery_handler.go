package errors

import (
	"github.com/kataras/iris/v14/context"
	recovery "github.com/kataras/iris/v14/middleware/recovery"
)

func init() {
	// The package moved under rest/ — the old "iris/errors." pattern never
	// matched the real symbol and the mapping had never fired.
	context.SetHandlerName("iris/rest/errors.RecoveryHandler.*", "iris.errors.recover")
}

// RecoveryHandler is a middleware which recovers from panics and sends an appropriate error response
// to the logger and the client.
func RecoveryHandler(ctx *context.Context) {
	defer func() {
		if err := recovery.PanicRecoveryError(ctx, recover()); err != nil {
			Internal.LogErr(ctx, err)
			ctx.StopExecution()
		}
	}()

	ctx.Next()
}
