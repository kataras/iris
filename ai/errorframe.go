package ai

import (
	json "encoding/json/v2"
	"net/http"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

// ErrorCode returns the canonical rest/errors code for this failure.
//
// The mapping is deliberate, and it is not a passthrough of the upstream's own
// status. A provider 401 means this gateway's credential is finished, which is
// not the caller's problem to fix; telling them they are unauthorised sends them
// to check a key they do not own.
//
// It returns an empty code for a cancelled request, which means there is nothing
// to report: the caller has gone, so nothing should be written and nobody should
// be paged. Test it with ErrorCodeName.String() == "", the same sentinel the
// errors package uses elsewhere.
func (e *UpstreamError) ErrorCode() errors.ErrorCodeName {
	switch e.Class {
	case ClassRateLimited:
		return errors.ResourceExhausted // 429, and the one code worth forwarding.
	case ClassExhausted:
		return errors.Unavailable // 503: this service cannot serve you right now.
	case ClassFatal:
		if e.Status == http.StatusRequestEntityTooLarge {
			return errors.TooLarge
		}

		return errors.InvalidArgument // 400: the request was wrong.
	case ClassCanceled:
		return "" // nobody is listening.
	default:
		if e.Timeout() {
			return errors.DeadlineExceeded // 504: the upstream did not answer in time.
		}

		return errors.BadGateway // 502: an upstream we depend on failed.
	}
}

// Report runs this failure through the rest/errors pipeline under its canonical
// code, returning the client-safe Error, or nil when there is nothing to report.
//
// The client is told what the failure means, never what the provider said. A
// provider's message is text this service did not write, about a system the
// caller cannot see, and it routinely names the vendor, the model or the host;
// forwarding it is a leak whatever the status code was. It goes to the logger
// and the collector, where it is useful, and the caller gets a sentence derived
// from the class.
//
// The error is wrapped as private on the way in, so even if this function's own
// text were removed the pipeline would still redact rather than forward.
func (e *UpstreamError) Report(ctx *context.Context) *errors.Error {
	code := e.ErrorCode()
	if code.String() == "" {
		return nil
	}

	reported := code.Report(ctx, errors.Private(e))
	if reported == nil {
		return nil
	}

	reported.Message, reported.Detail = e.clientText()

	return reported
}

// clientText describes the failure in terms of what the caller can do about it,
// naming nothing the caller cannot see.
//
// Rewrite it through Options.Redact, or replace the frame entirely through
// Options.SSE.ErrorFrame, if an application wants its own wording.
func (e *UpstreamError) clientText() (message, detail string) {
	switch e.Class {
	case ClassRateLimited:
		return "too many requests", "the service is rate limited, try again shortly"
	case ClassExhausted:
		return "service unavailable", "the service cannot handle this request right now"
	case ClassFatal:
		return "invalid request", "the request was rejected"
	default:
		if e.Timeout() {
			return "upstream timeout", "the service did not answer in time, try again"
		}

		return "upstream error", "the service is temporarily unavailable, try again"
	}
}

// reportError is the one error policy of a turn, shared by the terminal frame
// Close writes and by a KindError part given to Send.
//
// It is the AI-aware version of sse.ReportErrorFrame's pipeline: a classified
// upstream failure keeps its canonical code, so a rate limit reaches the client
// as RESOURCE_EXHAUSTED rather than as a generic internal error, and anything
// else falls through to the ordinary error-code map, which redacts what was not
// deliberately mapped.
//
// Redact runs last, over the text that is already client-safe, because it exists
// for the case the pipeline cannot cover on its own: an error the application
// deliberately mapped and described, whose description still names a vendor.
//
// It returns nil when there is nothing to tell the client: the request is
// cancelled, or Report had nothing to say.
func (r *Response) reportError(err error) *errors.Error {
	var reported *errors.Error

	if ue, ok := AsUpstream(err); ok {
		reported = ue.Report(r.ctx)
	} else {
		reported = errors.Report(r.ctx, err)
	}

	if reported == nil {
		return nil
	}

	if redact := r.opts.Redact; redact != nil {
		reported.Message = redact(reported.Message)
		reported.Detail = redact(reported.Detail)
	}

	return reported
}

// errorFrame renders the event a failed stream ends with: the envelope
// reportError produced, or no frame when it produced nothing.
func (r *Response) errorFrame(_ *context.Context, err error) (sse.Event, bool) {
	reported := r.reportError(err)
	if reported == nil {
		return sse.Event{}, false
	}

	data, marshalErr := json.Marshal(reported, jsonutil.DefaultMarshalOptionsSlice...)
	if marshalErr != nil {
		return sse.Event{Name: "error", Data: []byte(reported.Message)}, true
	}

	return sse.Event{Name: "error", Data: data}, true
}
