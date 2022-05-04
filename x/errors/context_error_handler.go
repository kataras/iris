package errors

import "github.com/kataras/iris/v12/context"

// DefaultContextErrorHandler returns a context error handler
// which fires a JSON bad request (400) error message when
// a rich rest response failed to be written to the client.
// Register it on Application.SetContextErrorHandler method.
var DefaultContextErrorHandler context.ErrorHandler = new(jsonErrorHandler)

type jsonErrorHandler struct{}

// HandleContextError completes the context.ErrorHandler interface. It's fired on
// rich rest response failures.
func (e *jsonErrorHandler) HandleContextError(ctx *context.Context, err error) {
	InvalidArgument.Err(ctx, err)
}
