package errors

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/macro/handler"
)

// DefaultPathParameterTypeErrorHandler registers an error handler for macro path type parameter.
// Register it with Application.Macros().SetErrorHandler(DefaultPathParameterTypeErrorHandler).
var DefaultPathParameterTypeErrorHandler handler.ParamErrorHandler = func(ctx *context.Context, paramIndex int, err error) {
	param := ctx.Params().GetEntryAt(paramIndex) // key, value fields.
	InvalidArgument.DataWithDetails(ctx, "invalid path parameter", err.Error(), param)
}
