package errors

import (
	"github.com/kataras/iris/v14/context"
)

// No. See "ApplyPatches".
// func init() {
// 	// Register the default error handler.
// 	cors.DefaultErrorHandler = DefaultCORSErrorHandler
// }

// DefaultCORSErrorHandler returns a context error handler
// which calls the FailedPrecondition.Err on any incoming error when
// cors middleware's HandleErrorFunc is not set.
var DefaultCORSErrorHandler context.ErrorHandler = func(ctx *context.Context, err error) {
	FailedPrecondition.Err(ctx, err)
}
