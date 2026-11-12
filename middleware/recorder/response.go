package recorder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/textproto"
	"slices"
	"strconv"
	"sync"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recorder.Response*", "iris.recorder.response")
}

// Response captures the response of the request sent by the server.
//
// It should be used as a middleware.
// See `GetResponseRecorder` to get the response recorder from the context.
// See `GetResponse` to get the captured response as an http.Response.
// Usage:
//
//	app.Use(recorder.Response)
//	app.Get("/path", handler)
func Response(ctx *context.Context) {
	// Acquire a response recorder. If it's new set it and release it by this handler,
	// otherwise another handler initialized a response recorder and we should not release it here.
	previous := ctx.ResponseWriter()
	if rr, ok := AcquireResponseRecorder(previous); ok {
		// Release the captured response when the whole handler chain is done,
		// and put the previous writer back on the Context first: a handler
		// registered outside this one reads the Context's writer after the
		// chain returns, and a released recorder may already belong to another
		// request.
		defer func() {
			ctx.SetResponseWriter(previous)
			ReleaseResponseRecorder(rr)
		}()
		// Set the response writer to the response recorder.
		ctx.SetResponseWriter(rr)
	}

	// Call the next handler.
	ctx.Next()
}

// GetResponseRecorder returns the ResponseRecorder from the context.
func GetResponseRecorder(ctx *context.Context) *ResponseRecorder {
	rr, ok := ctx.ResponseWriter().(*ResponseRecorder)
	if !ok {
		return nil
	}

	return rr
}

// IsRecordingResponse returns true if the response is being recorded.
func IsRecordingResponse(ctx *context.Context) bool {
	return GetResponseRecorder(ctx) != nil
}

// GetResponse returns the captured response as an http.Response.
func GetResponse(ctx *context.Context) *http.Response {
	rr := GetResponseRecorder(ctx)
	if rr == nil {
		return nil
	}

	return rr.Result()
}

// ResponseRecorder is a custom response writer that captures the response.
type ResponseRecorder struct {
	context.ResponseWriter

	Body    *bytes.Buffer
	Code    int
	Headers http.Header

	result *http.Response

	// disabled reports that this recorder has stepped out of the chain: writes,
	// headers and flushes pass straight through to the wrapped writer and
	// release forwards nothing, because everything captured was already handed
	// over by Disable.
	disabled bool
	// mediaTypeChecked guards the one-time streaming media type check.
	mediaTypeChecked bool
}

// streamingMediaTypes are response media types that must never be recorded.
// Recording buffers the whole body, so a stream recorded is a stream destroyed:
// the client receives one lump when the handler returns instead of each event as
// it is produced.
//
// Compared after TrimHeaderValue has removed any parameters, so
// "text/event-stream;charset=utf-8" cannot fall through. Unexported and never
// written after init, so reading it per response is not a data race.
var streamingMediaTypes = map[string]struct{}{
	"text/event-stream":    {},
	"application/x-ndjson": {},
}

var _ context.ResponseWriter = (*ResponseRecorder)(nil)

var recorderPool = sync.Pool{New: func() any { return &ResponseRecorder{} }}

// AcquireResponseRecorder returns a ResponseRecorder from the pool.
// See `ReleaseResponseRecorder` to release the response recorder back to the pool.
// The second argument returns true if the response recorder was initialized by the pool,
// if the second argument is true then this response recorder should return back to the pool
// by the same caller.
func AcquireResponseRecorder(w context.ResponseWriter) (*ResponseRecorder, bool) {
	// First check if the given writer is already a ResponseRecorder.
	rr, ok := w.(*ResponseRecorder)
	if ok {
		return rr, false
	}

	// Get a ResponseRecorder from the pool.
	rr = recorderPool.Get().(*ResponseRecorder)
	rr.Reset(w)
	return rr, true
}

// ReleaseResponseRecorder releases the ResponseRecorder back to the pool.
//
// A caller that put w on a Context with SetResponseWriter puts the previous
// writer back before calling this. The released recorder is reused by another
// request, and a handler registered outside the caller (an access log, an
// error handler) still reads the Context's writer after the chain returns:
//
//	previous := ctx.ResponseWriter()
//	if rr, ok := recorder.AcquireResponseRecorder(previous); ok {
//		ctx.SetResponseWriter(rr)
//		defer func() {
//			ctx.SetResponseWriter(previous)
//			recorder.ReleaseResponseRecorder(rr)
//		}()
//	}
func ReleaseResponseRecorder(w *ResponseRecorder) {
	w.release()
	recorderPool.Put(w)
}

// Clone returns a copy of the ResponseRecorder that is safe to use after the
// original has gone back to the pool.
//
// Note(@kataras): the body bytes must be COPIED, not aliased.
//
// bytes.NewBuffer(rr.Body.Bytes()) hands the new buffer the SAME backing array.
// The recorder returns to its sync.Pool as soon as the handler chain ends, and
// the next request's Reset writes straight into that array - so an async
// consumer still reading the clone (accesslog's Async mode is exactly this)
// sees another client's response body. It is also a plain data race.
func (rr *ResponseRecorder) Clone() context.ResponseWriter {
	return &ResponseRecorder{
		ResponseWriter: rr.ResponseWriter.Clone(),
		Body:           bytes.NewBuffer(slices.Clone(rr.Body.Bytes())),
		Code:           rr.Code,
		Headers:        rr.Headers.Clone(),
	}
}

// Reset resets the ResponseRecorder for reuse.
func (rr *ResponseRecorder) Reset(w context.ResponseWriter) {
	rr.ResponseWriter = w
	if rr.Body != nil {
		rr.Body.Reset()
	} else {
		rr.Body = bytes.NewBuffer(nil)
	}
	rr.Code = w.StatusCode()
	rr.Headers = make(http.Header)
	rr.result = nil
	rr.disabled = false
	rr.mediaTypeChecked = false
}

// release writes the captured response to the underlying ResponseWriter.
// Release should only be called once per request.
func (rr *ResponseRecorder) release() {
	if rr.disabled {
		// Disable already forwarded everything captured and the handler has been
		// writing straight through since. Replaying here would repeat the body
		// and stamp a stale status code over a response already on the wire.
		return
	}

	// Write headers.
	h := rr.ResponseWriter.Header()
	maps.Copy(h, rr.Headers)
	rr.ResponseWriter.WriteHeader(rr.Code)
	// Write body, when there is one. A write of zero bytes still sends the
	// status line and the headers, and after that a handler registered outside
	// the recorder's owner (rest/errors.Handler answering a bare failure
	// status, say) can add a body but no Content-Type for it. With nothing
	// recorded the status and the headers are handed over and the response
	// stays open; the router sends them when the request ends.
	if body := rr.Body.Bytes(); len(body) > 0 {
		rr.ResponseWriter.Write(body)
	}
}

// Disable steps this recorder out of the chain and returns the writer it
// wrapped, implementing context.ResponseWriterDisabler.
//
// It first hands over everything captured so far - headers, status code, body -
// in the order release would have, so calling it mid-response is lossless. From
// then on the recorder is a passthrough and release does nothing.
//
// The cost is the recording itself: a disabled recorder has no body to report,
// so accesslog logs a streaming response without one. That is the correct trade.
// A response body you can only produce by buffering the stream is a response the
// client never received as a stream.
func (rr *ResponseRecorder) Disable() context.ResponseWriter {
	underline := rr.ResponseWriter
	if rr.disabled || underline == nil {
		return underline
	}

	rr.disabled = true

	maps.Copy(underline.Header(), rr.Headers)
	underline.WriteHeader(rr.Code)
	if body := rr.Body.Bytes(); len(body) > 0 {
		underline.Write(body)
	}
	rr.Body.Reset()
	rr.result = nil

	return underline
}

// checkMediaType disables recording the first time it runs if the handler has
// chosen a streaming Content-Type.
//
// This is the safety net for a handler that writes text/event-stream by hand
// rather than through the sse package. It matters more here than in
// middleware/compress, because a recorder is installed by default: accesslog
// acquires one whenever it reads the response body, and the builder's
// LogRequests(true) turns that on.
func (rr *ResponseRecorder) checkMediaType() {
	if rr.mediaTypeChecked {
		return
	}
	rr.mediaTypeChecked = true

	if rr.ResponseWriter == nil {
		return
	}

	mediaType := context.TrimHeaderValue(rr.Headers.Get(context.ContentTypeHeaderKey))
	if _, isStreaming := streamingMediaTypes[mediaType]; isStreaming {
		rr.Disable()
	}
}

// Written returns the number of bytes recorded for the response body.
//
// With nothing recorded it reports what the wrapped writer does, so the two
// states a caller has to tell apart keep their meaning through a recorder:
// context.NoWritten when the response is untouched, and
// context.StatusCodeWritten or a byte count when the status line or a body
// already went out underneath (a flush before the recorder was installed, or
// a chunked flush through it). It used to report 0 in every one of these
// cases, which reads as "status line sent, no body yet".
func (rr *ResponseRecorder) Written() int {
	if rr.disabled {
		return rr.ResponseWriter.Written()
	}

	if n := rr.Body.Len(); n > 0 {
		return n
	}

	return rr.ResponseWriter.Written()
}

// StatusCode returns the status code of the response.
func (rr *ResponseRecorder) StatusCode() int {
	if rr.disabled {
		return rr.ResponseWriter.StatusCode()
	}

	return rr.Code
}

// Write writes the response body to the buffer.
func (rr *ResponseRecorder) Write(b []byte) (int, error) {
	rr.checkMediaType()

	if rr.disabled {
		return rr.ResponseWriter.Write(b)
	}

	return rr.Body.Write(b)
}

// WriteHeader records the status code. It is not forwarded to the
// underlying ResponseWriter until the recorder is released.
func (rr *ResponseRecorder) WriteHeader(statusCode int) {
	if rr.disabled {
		rr.ResponseWriter.WriteHeader(statusCode)
		return
	}

	rr.Code = statusCode
}

// Header returns the response headers.
//
// A disabled recorder returns the wrapped writer's header map, so a header set
// after it stepped aside still reaches the client.
func (rr *ResponseRecorder) Header() http.Header {
	if rr.disabled {
		return rr.ResponseWriter.Header()
	}

	return rr.Headers
}

// SetBody sets the response body.
func (rr *ResponseRecorder) SetBody(body []byte) {
	rr.Body.Reset()
	rr.Body.Write(body)
}

// GetBody returns the response body.
func (rr *ResponseRecorder) GetBody() []byte {
	if rr == nil {
		return nil
	}

	// Copy bytes to prevent the caller from modifying the response body.
	return append([]byte(nil), rr.Body.Bytes()...)
}

// Result returns the captured response as an http.Response
func (rr *ResponseRecorder) Result() *http.Response {
	if rr.result != nil {
		return rr.result
	}

	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: rr.Code,
		Header:     rr.Headers.Clone(),
	}
	rr.result = res
	if res.StatusCode == 0 {
		res.StatusCode = 200
	}
	res.Status = fmt.Sprintf("%03d %s", res.StatusCode, http.StatusText(res.StatusCode))
	if rr.Body != nil {
		res.Body = io.NopCloser(bytes.NewReader(rr.Body.Bytes()))
	} else {
		res.Body = http.NoBody
	}
	res.ContentLength = parseContentLength(res.Header.Get("Content-Length"))

	return res
}

func (rr *ResponseRecorder) Flush() {
	_ = rr.FlushError()
}

// FlushError forwards anything buffered and reports whether the flush reached
// the transport, implementing context.ResponseWriterFlushError.
func (rr *ResponseRecorder) FlushError() error {
	rr.checkMediaType()

	if rr.disabled {
		return context.FlushResponseWriter(rr.ResponseWriter)
	}

	// This fixes response recorder when chunked + Flush is used.
	//
	// Note(@kataras): the body is only Reset when it was actually forwarded.
	//
	// The reset used to run unconditionally while the forwarding above was gated
	// on "Transfer-Encoding: chunked", so any handler that called Flush without
	// that header - SSE, streaming JSON - silently lost everything buffered so
	// far. And httpcache, accesslog and servertiming all install a recorder
	// globally, so the handler had no way to know one was in the way.
	if rr.Headers.Get("Transfer-Encoding") == "chunked" {
		if rr.Written() == context.NoWritten {
			if len(rr.Headers) > 0 {
				h := rr.ResponseWriter.Header()
				// note: we don't reset the current underline's headers.
				maps.Copy(h, rr.Headers)
			}
		}

		if body := rr.Body.Bytes(); len(body) > 0 {
			rr.ResponseWriter.Write(body)
		}

		rr.Body.Reset()
	}

	return context.FlushResponseWriter(rr.ResponseWriter)
}

// ErrPushNotSupported is returned by the Push method to
// indicate that HTTP/2 Push support is not available.
var ErrPushNotSupported = errors.New("push feature is not supported by this ResponseWriter")

// Push implements the http.Pusher interface.
func (rr *ResponseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := context.UnwrapResponseWriter(rr.ResponseWriter).(http.Pusher); ok {
		if err := pusher.Push(target, opts); errors.Is(err, http.ErrNotSupported) {
			return ErrPushNotSupported
		}

		return nil
	}

	// Note(@kataras): do NOT reset the recording here.
	//
	// Reset clears Body, Code and Headers. A handler that opportunistically calls
	// Push and ignores the "not supported" error - which is the normal way to use
	// it over HTTP/1.1 - had everything it had written so far thrown away, and
	// the client received an empty body. Failing to push is not a reason to
	// discard the response.
	return ErrPushNotSupported
}

// parseContentLength trims whitespace from s and returns -1 if no value
// is set, or the value if it's >= 0.
//
// This a modified version of same function found in net/http/transfer.go. This
// one just ignores an invalid header.
func parseContentLength(cl string) int64 {
	cl = textproto.TrimString(cl)
	if cl == "" {
		return -1
	}
	n, err := strconv.ParseUint(cl, 10, 63)
	if err != nil {
		return -1
	}
	return int64(n)
}
