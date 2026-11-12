package errors

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router/macro/handler"

	"github.com/kataras/iris/v14/middleware/cors"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/middleware/versioning"
)

// ApplyPatches registers the DefaultPathParameterTypeErrorHandler,
// DefaultContextErrorHandler, DefaultRateLimitExceededHandler,
// DefaultVersioningNotFoundHandler and DefaultCORSErrorHandler to their default places
// in order to be used by the Iris' router and middleware system globally.
// It should be called before any server starts.
// Usage:
//
//	func init() { errors.ApplyPatches() }
func ApplyPatches() {
	handler.DefaultErrorHandler = DefaultPathParameterTypeErrorHandler
	context.DefaultErrorHandler = DefaultContextErrorHandler
	rate.DefaultExceedHandler = DefaultRateLimitExceededHandler
	versioning.DefaultNotFoundHandler = DefaultVersioningNotFoundHandler
	cors.DefaultErrorHandler = DefaultCORSErrorHandler
}

// DefaultPathParameterTypeErrorHandler registers an error handler for macro path type parameter.
// Register it with Application.Macros().SetErrorHandler(DefaultPathParameterTypeErrorHandler).
var DefaultPathParameterTypeErrorHandler handler.ParamErrorHandler = func(ctx *context.Context, paramIndex int, err error) {
	param, _ := ctx.Params().LookupByIndex(paramIndex) // key, value fields.
	InvalidArgument.DataWithDetail(ctx, "invalid path parameter", err.Error(), param)
}
