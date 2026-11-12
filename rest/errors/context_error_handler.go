package errors

import "github.com/kataras/iris/v14/context"

// No. See "ApplyPatches".
// func init() {
// 	// Register the default context error handler.
// 	context.DefaultErrorHandler = DefaultContextErrorHandler
// }

// DefaultContextErrorHandler returns a context error handler
// which calls the HandleError on any incoming error when
// a rich rest response failed to be written to the client.
// Register it globally by calling the "ApplyPatches" function, which assigns it to
// the context.DefaultErrorHandler variable, or per application through
// iris.Options{Context: iris.ContextOptions{ErrorHandler: ...}}
// (or per group and route with the iris.WithErrorHandler middleware).
var DefaultContextErrorHandler context.ErrorHandler = func(ctx *context.Context, err error) {
	HandleError(ctx, err)
}
