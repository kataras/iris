package sse

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/rest/errors"
)

// ErrorFrameFunc renders the event Close writes when a stream fails after it has
// already committed. It reports false to write no frame at all.
type ErrorFrameFunc func(ctx *context.Context, err error) (Event, bool)

// ReportErrorFrame is the default renderer for a failure that arrives after the
// first event.
//
// It runs the error through rest/errors.Report, which is what makes a mid-stream
// failure behave like every other failure in the application: the route's error
// mappings apply, the logger and the collector fire so a cloud sink sees it, a
// correlation id is generated, and an internal or private error is redacted to a
// generic message before anything reaches the client.
//
// That redaction is the part worth noticing. An unmapped error becomes "server
// error" on the wire, so an upstream message naming a vendor, a model or a host
// cannot leak into a stream by default. Map the errors you do want to describe,
// through errors.Options.MapErrors and its relatives, exactly as for an ordinary
// response.
//
// The payload is the rest/errors Error envelope, so a client parses the same
// shape whether the failure happened before the first byte or after it.
//
// It writes no frame when Report has nothing to report, which is the case for a
// client that disconnected while errors.SkipCanceled is on: there is nobody left
// to tell, and nobody should be paged.
func ReportErrorFrame(ctx *context.Context, err error) (Event, bool) {
	reported := errors.Report(ctx, err)
	if reported == nil {
		return Event{}, false
	}

	buf, marshalErr := marshalJSONTo(reported)
	if marshalErr != nil {
		// Failing to encode the failure is not a reason to end the stream
		// without saying anything.
		return Event{Name: "error", Data: []byte(reported.Message)}, true
	}

	// The Event outlives this call (the caller sends it later), so the bytes
	// must be owned by the Event, not by a pooled buffer.
	data := append([]byte(nil), buf.Bytes()...)
	jsonutil.ReleaseBuffer(buf)

	return Event{Name: "error", Data: data}, true
}

// RawErrorFrame renders the error's own text, with no mapping, no redaction and
// no reporting.
//
// It is the shape this package used before ReportErrorFrame became the default,
// and it is appropriate only when the error text is already known to be safe for
// the client to read. Anything derived from an upstream response is not.
func RawErrorFrame(_ *context.Context, err error) (Event, bool) {
	return Event{Name: "error", Data: []byte(err.Error())}, true
}
