package errors

import (
	"github.com/kataras/iris/v14/context"
)

// Report runs the full error pipeline and returns the resulting Error without
// writing anything to the response.
//
// It exists for responses that have already committed a status code, which is
// every streaming response after its first byte. The ordinary path cannot help
// there: ErrorCodeName.Err and its relatives end in a render, and a render in
// the middle of an event stream writes a JSON body into the stream and corrupts
// it. Skipping the pipeline instead is worse, because it is the only thing that
// makes a failure visible: the logger and the collector are what carry an error
// to a cloud sink, and a stream that fails silently is a stream nobody knows
// failed.
//
// Report does everything the render path does except render:
//
//   - resolves the error against the route's error-code map, so mappings
//     registered through Options.MapErrors and friends apply exactly as they do
//     to an ordinary response;
//   - honours the global SkipCanceled setting, so a client that hung up does not
//     page anyone. That is decided by the request's own context, never by the
//     error's shape: an upstream timeout looks like a deadline to errors.Is and
//     is still reported;
//   - calls the logger, which is the package-level LogError unless a route set
//     its own. This is the hook a cloud logger replaces, so a stream failure
//     reaches it with no extra wiring;
//   - generates the correlation id through Options.SetIDGenerator;
//   - redacts a private or internal error down to a generic message, exactly as
//     the renderer would, so nothing internal reaches the client;
//   - calls the collector with the finished Error.
//
// It returns nil when there is nothing to report: a nil error, or a cancellation
// while SkipCanceled is on. The returned value is safe to send to the client;
// the unredacted original went to the logger and the collector.
//
// The sse package uses this to render its error frame, so a streaming handler
// gets the same treatment as any other without asking for it.
func Report(ctx *context.Context, err error) *Error {
	if err == nil || ctx == nil {
		return nil
	}

	opts := getReadOnlyOptions(ctx)

	// Resolve through the route's map first, so an application that already
	// classifies its errors does not have to classify them twice.
	codeName, errToReport := opts.ErrorCodeMap().Filter(ctx, err)
	if errToReport == nil {
		errToReport = err
	}
	if codeName.String() == "" {
		codeName = Internal
	}

	return codeName.Report(ctx, errToReport)
}

// Report runs the error pipeline under this specific error code and returns the
// resulting Error without writing anything to the response.
//
// Use it when the code is already known - a classified upstream failure, say -
// and the error-code map has nothing to add. See the package-level Report for
// what the pipeline does and why it is separate from rendering.
func (e ErrorCodeName) Report(ctx *context.Context, err error) *Error {
	if err == nil || ctx == nil {
		return nil
	}

	if SkipCanceled && ctx.IsCanceled() {
		// The client's own request is cancelled: nobody is waiting for this and
		// nobody should be woken by it.
		//
		// Only the client's context decides. The error's shape does not: an
		// upstream dial or read timeout satisfies errors.Is(err,
		// context.DeadlineExceeded) too, and it is the failure the client is
		// waiting to hear about and an operator most needs to see.
		return nil
	}

	errorCode, ok := errorCodeNameMap[e]
	if !ok {
		errorCode = errorCodeNameMap[ErrUnexpected]
	}

	opts := getReadOnlyOptions(ctx)

	// The logger runs before any redaction, so the sink gets the real error.
	// This is the line InitPatrol-style setups hook, through LogError.
	opts.logError(ctx, err)

	message, detail, _ := parseRootError(err)
	if detail == "" {
		if errWithDetail, isDetailed := err.(detailedError); isDetailed {
			detail = errWithDetail.Detail()
		}
	}

	reported := &Error{
		ID:        opts.getErrorID(ctx),
		ErrorCode: errorCode,
		Message:   message,
		Detail:    detail,
	}

	if isPrivateError(e, err) {
		// Same redaction the renderer applies, and for the same reason: the
		// client is told that something failed, and nothing else.
		reported.Message = "server error"
		reported.Detail = "an unexpected error occurred while processing the request"
	}

	opts.collect(ctx, reported)

	return reported
}

// isPrivateError reports whether the client must not see this error's text.
//
// An internal error is private by definition, and so is anything wrapped with
// Private: if any error in the chain is private then the whole chain is.
func isPrivateError(e ErrorCodeName, err error) bool {
	if e == Internal {
		return true
	}

	for _, childErr := range UnwrapAll(err) {
		if _, private := childErr.(context.ErrPrivate); private {
			return true
		}
	}

	return false
}
