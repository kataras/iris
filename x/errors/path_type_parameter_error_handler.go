package errors

import "github.com/kataras/iris/v12/context"

// DefaultPathTypeParameterErrorHandler registers an error handler for macro path type parameter.
// Register it with Application.Macros().SetErrorHandler(DefaultPathTypeParameterErrorHandler).
func DefaultPathTypeParameterErrorHandler(ctx *context.Context, paramIndex int, err error) {
	param := ctx.Params().GetEntryAt(paramIndex) // key, value fields.
	InvalidArgument.DataWithDetails(ctx, "invalid path parameter", err.Error(), param)
}
