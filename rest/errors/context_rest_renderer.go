// This file installs the canonical RestErrorRenderer of the Context's typed
// binding/response methods (BindJSON, BindParams, Count, Paginated, ...).
package errors

import (
	"github.com/kataras/iris/v14/context"
)

// The registration happens at package-init time, mirroring the controller
// resolver registration in controller.go. It is deliberately NOT an
// ApplyPatches-style cosmetic override: this is structural layering
// (dependency inversion). Package context cannot import rest/errors - that
// would form an import cycle - so it declares the RestErrorRenderer seam and
// this package, which owns the canonical wire format, installs itself into it
// the moment it is linked into the program.
func init() {
	context.SetRestErrorRenderer(renderer{})
}

// renderer adapts the rest/errors package-level error handling to the
// context.RestErrorRenderer seam.
type renderer struct{}

// HandleError delegates to HandleError: unrecognized errors render as 500.
func (renderer) HandleError(ctx *context.Context, err error) bool {
	return HandleError(ctx, err)
}

// HandleRequestError delegates to HandleRequestError: unrecognized errors
// render as 400.
func (renderer) HandleRequestError(ctx *context.Context, err error) bool {
	return HandleRequestError(ctx, err)
}

// HandleRequestJSONError delegates to HandleRequestJSONError.
func (renderer) HandleRequestJSONError(ctx *context.Context, err error) bool {
	return HandleRequestJSONError(ctx, err)
}

// HandleResponseJSONError delegates to HandleResponseJSONError.
func (renderer) HandleResponseJSONError(ctx *context.Context, err error) bool {
	return HandleResponseJSONError(ctx, err)
}

// HandleValidationError renders a validation failure as a client error.
// Plain errors are tagged as InvalidArgument; already-classified errors
// keep their own code and formatting.
func (renderer) HandleValidationError(ctx *context.Context, err error) {
	var (
		code ErrorCodeName
		ve   ValidationError
		ves  ValidationErrors
	)
	if !As(err, &code) && !As(err, &ve) && !As(err, &ves) {
		err = InvalidArgument.Wrap(err)
	}

	HandleRequestError(ctx, err)
}

// HandleNotFound delegates to NotFound.Messagef.
func (renderer) HandleNotFound(ctx *context.Context, format string, args ...any) {
	NotFound.Messagef(ctx, format, args...)
}

// HandleInvalidArgument delegates to InvalidArgument.Messagef.
func (renderer) HandleInvalidArgument(ctx *context.Context, format string, args ...any) {
	InvalidArgument.Messagef(ctx, format, args...)
}
