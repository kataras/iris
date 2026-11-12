package context

import (
	"errors"
	"fmt"
	"net/http"
)

// RestErrorRenderer renders the error responses of the Context's typed
// binding/response methods (BindJSON, BindParams, Count, Paginated, ...).
// The canonical implementation lives in rest/errors and installs itself at
// package-init time; the built-in fallback keeps package context usable alone.
type RestErrorRenderer interface {
	// HandleError renders "err" as a server-side failure.
	// Unrecognized errors render as 500. It reports whether a response
	// was written and the caller should stop.
	HandleError(ctx *Context, err error) bool
	// HandleRequestError renders "err" as a client-side failure.
	// Unrecognized errors render as 400. It reports whether a response
	// was written and the caller should stop.
	HandleRequestError(ctx *Context, err error) bool
	// HandleRequestJSONError renders a failure to decode the request body
	// as a client error (400). It reports whether a response was written.
	HandleRequestJSONError(ctx *Context, err error) bool
	// HandleResponseJSONError renders a failure to encode the response as
	// a server error (500). It reports whether a response was written.
	HandleResponseJSONError(ctx *Context, err error) bool
	// HandleValidationError classifies-or-wraps a validation failure,
	// then renders it as a 4xx client error.
	HandleValidationError(ctx *Context, err error)
	// HandleNotFound renders a 404 with the formatted message.
	HandleNotFound(ctx *Context, format string, args ...any)
	// HandleInvalidArgument renders a 400 with the formatted message.
	HandleInvalidArgument(ctx *Context, format string, args ...any)
}

// restErrorRenderer is the active renderer. The rest/errors package replaces
// the minimal fallback through SetRestErrorRenderer at its init time.
var restErrorRenderer RestErrorRenderer = minimalRestErrorRenderer{}

// SetRestErrorRenderer sets the renderer used by the Context's typed
// binding/response methods to render their error responses.
// It's called by the rest/errors package at init time; a nil value is ignored.
func SetRestErrorRenderer(r RestErrorRenderer) {
	if r != nil {
		restErrorRenderer = r
	}
}

// restErrorMessage is the wire shape of the minimal fallback renderer:
// a single-member JSON object, e.g. {"message":"unable to parse request body"}.
type restErrorMessage struct {
	Message string `json:"message"`
}

// minimalRestErrorRenderer is the built-in RestErrorRenderer fallback.
// It keeps package context usable without the rest/errors machinery:
// each method writes a {"message": ...} JSON body with the named status
// code and stops the handlers chain. The canonical rest/errors renderer
// replaces it at init time when that package is linked in.
type minimalRestErrorRenderer struct{}

func (minimalRestErrorRenderer) stop(ctx *Context, statusCode int, message string) {
	_ = ctx.StopWithJSON(statusCode, restErrorMessage{Message: message})
}

// HandleError renders a 500 response. It never leaks err.Error() to the
// client; the error is logged through the application logger instead.
func (r minimalRestErrorRenderer) HandleError(ctx *Context, err error) bool {
	if err == nil || ctx.IsStopped() {
		return false
	}

	ctx.app.Logger().Error(err)
	r.stop(ctx, http.StatusInternalServerError, "internal server error")
	return true
}

// HandleRequestError renders a 400 response carrying the error's text
// (client-caused errors are safe to disclose). An oversize body
// (ErrRequestEntityTooLarge) renders as 413, like the canonical renderer.
func (r minimalRestErrorRenderer) HandleRequestError(ctx *Context, err error) bool {
	if err == nil || ctx.IsStopped() {
		return false
	}

	if errors.Is(err, ErrRequestEntityTooLarge) {
		r.stop(ctx, http.StatusRequestEntityTooLarge, "request body too large")
		return true
	}

	r.stop(ctx, http.StatusBadRequest, err.Error())
	return true
}

// HandleRequestJSONError renders any non-nil error as a 400 with the exact
// generic text the canonical rest/errors renderer uses, without inspecting
// the error type.
func (r minimalRestErrorRenderer) HandleRequestJSONError(ctx *Context, err error) bool {
	if err == nil || ctx.IsStopped() {
		return false
	}

	r.stop(ctx, http.StatusBadRequest, "unable to parse request body")
	return true
}

// HandleResponseJSONError renders any non-nil error as a 500 with the exact
// generic text the canonical rest/errors renderer uses, logging the error
// instead of disclosing it.
func (r minimalRestErrorRenderer) HandleResponseJSONError(ctx *Context, err error) bool {
	if err == nil || ctx.IsStopped() {
		return false
	}

	ctx.app.Logger().Error(err)
	r.stop(ctx, http.StatusInternalServerError, "unable to parse response body")
	return true
}

// HandleValidationError renders a validation failure as a 400 client error.
func (r minimalRestErrorRenderer) HandleValidationError(ctx *Context, err error) {
	if err == nil || ctx.IsStopped() {
		return
	}

	r.stop(ctx, http.StatusBadRequest, err.Error())
}

// HandleNotFound renders a 404 with the formatted message.
func (r minimalRestErrorRenderer) HandleNotFound(ctx *Context, format string, args ...any) {
	if ctx.IsStopped() {
		return
	}

	r.stop(ctx, http.StatusNotFound, fmt.Sprintf(format, args...))
}

// HandleInvalidArgument renders a 400 with the formatted message.
func (r minimalRestErrorRenderer) HandleInvalidArgument(ctx *Context, format string, args ...any) {
	if ctx.IsStopped() {
		return
	}

	r.stop(ctx, http.StatusBadRequest, fmt.Sprintf(format, args...))
}
