package context

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

// ResponseWriter interface is used by the context to serve an HTTP handler to
// construct an HTTP response.
//
// Note: Only this ResponseWriter is an interface in order to be able
// for developers to change the response writer of the Context via `context.ResetResponseWriter`.
// The rest of the response writers implementations (ResponseRecorder & CompressResponseWriter)
// are coupled to the internal ResponseWriter implementation(*responseWriter).
//
// A ResponseWriter may not be used after the Handler
// has returned.
type ResponseWriter interface {
	http.ResponseWriter

	// Naive returns the simple, underline and original http.ResponseWriter
	// that backends this response writer.
	Naive() http.ResponseWriter
	// SetWriter sets the underline http.ResponseWriter
	// that this responseWriter should write on.
	SetWriter(underline http.ResponseWriter)
	// BeginResponse receives an http.ResponseWriter
	// and initialize or reset the response writer's field's values.
	BeginResponse(http.ResponseWriter)
	// EndResponse is the last function which is called right before the server sent the final response.
	//
	// Here is the place which we can make the last checks or do a cleanup.
	EndResponse()

	// IsHijacked reports whether this response writer's connection is hijacked.
	IsHijacked() bool

	// StatusCode returns the status code header value.
	StatusCode() int

	// Written should returns the total length of bytes that were being written to the client.
	// In addition iris provides some variables to help low-level actions:
	// NoWritten, means that nothing were written yet and the response writer is still live.
	// StatusCodeWritten, means that status code was written but no other bytes are written to the client, response writer may closed.
	// > 0 means that the reply was written and it's the total number of bytes were written.
	Written() int

	// SetWritten sets manually a value for written, it can be
	// NoWritten(-1) or StatusCodeWritten(0), > 0 means body length which is useless here.
	SetWritten(int)

	// SetBeforeFlush registers the unique callback which called exactly before the response is flushed to the client.
	SetBeforeFlush(cb func())
	// GetBeforeFlush returns (not execute) the before flush callback, or nil if not set by SetBeforeFlush.
	GetBeforeFlush() func()
	// FlushResponse should be called only once before EndResponse.
	// it tries to send the status code if not sent already
	// and calls the  before flush callback, if any.
	//
	// FlushResponse can be called before EndResponse, but it should
	// be the last call of this response writer.
	FlushResponse()

	// clone returns a clone of this response writer
	// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder.
	Clone() ResponseWriter

	// CopyTo writes a response writer (temp: status code, headers and body) to another response writer
	CopyTo(ResponseWriter)

	// Flusher indicates if `Flush` is supported by the client.
	//
	// The default HTTP/1.x and HTTP/2 ResponseWriter implementations
	// support Flusher, but ResponseWriter wrappers may not. Handlers
	// should always test for this ability at runtime.
	//
	// Note that even for ResponseWriters that support Flush,
	// if the client is connected through an HTTP proxy,
	// the buffered data may not reach the client until the response
	// completes.
	Flusher() (http.Flusher, bool)
	// Flush sends any buffered data to the client.
	Flush() // required by compress writer.
}

// ResponseWriterBodyReseter can be implemented by
// response writers that supports response body overriding
// (e.g. recorder and compressed).
type ResponseWriterBodyReseter interface {
	// ResetBody should reset the body and reports back if it could reset successfully.
	ResetBody()
}

// ResponseWriterDisabler can be implemented
// by response writers that can be disabled and restored to their previous state
// (e.g. compressed).
type ResponseWriterDisabler interface {
	// Disable should disable this type of response writer and fallback to the default one.
	Disable()
}

// ResponseWriterReseter can be implemented
// by response writers that can clear the whole response
// so a new handler can write into this from the beginning.
// E.g. recorder, compressed (full) and common writer (status code and headers).
type ResponseWriterReseter interface {
	// Reset should reset the whole response and reports
	// whether it could reset successfully.
	Reset() bool
}

// ResponseWriterWriteTo can be implemented
// by response writers that needs a special
// encoding before writing to their buffers.
// E.g. a custom recorder that wraps a custom compressed one.
//
// Not used by the framework itself.
type ResponseWriterWriteTo interface {
	WriteTo(dest io.Writer, p []byte)
}

//  +------------------------------------------------------------+
//  | Response Writer Implementation                             |
//  +------------------------------------------------------------+

var rpool = sync.Pool{New: func() interface{} { return &responseWriter{} }}

// AcquireResponseWriter returns a new *ResponseWriter from the pool.
// Releasing is done automatically when request and response is done.
func AcquireResponseWriter() ResponseWriter {
	return rpool.Get().(*responseWriter)
}

func releaseResponseWriter(w ResponseWriter) {
	rpool.Put(w)
}

// ResponseWriter is the basic response writer,
// it writes directly to the underline http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter

	statusCode int // the saved status code which will be used from the cache service
	// statusCodeSent bool // reply header has been (logically) written | no needed any more as we have a variable to catch total len of written bytes
	written int // the total size of bytes were written
	// yes only one callback, we need simplicity here because on FireStatusCode the beforeFlush events should NOT be cleared
	// but the response is cleared.
	// Sometimes is useful to keep the event,
	// so we keep one func only and let the user decide when he/she wants to override it with an empty func before the FireStatusCode (context's behavior)
	beforeFlush func()
}

var _ ResponseWriter = (*responseWriter)(nil)

const (
	defaultStatusCode = http.StatusOK
	// NoWritten !=-1 => when nothing written before
	NoWritten = -1
	// StatusCodeWritten != 0 =>  when only status code written
	StatusCodeWritten = 0
)

// Naive returns the simple, underline and original http.ResponseWriter
// that backends this response writer.
func (w *responseWriter) Naive() http.ResponseWriter {
	return w.ResponseWriter
}

// BeginResponse receives an http.ResponseWriter
// and initialize or reset the response writer's field's values.
func (w *responseWriter) BeginResponse(underline http.ResponseWriter) {
	w.beforeFlush = nil
	w.written = NoWritten
	w.statusCode = defaultStatusCode
	w.SetWriter(underline)
}

// SetWriter sets the underline http.ResponseWriter
// that this responseWriter should write on.
func (w *responseWriter) SetWriter(underline http.ResponseWriter) {
	w.ResponseWriter = underline
}

// EndResponse is the last function which is called right before the server sent the final response.
//
// Here is the place which we can make the last checks or do a cleanup.
func (w *responseWriter) EndResponse() {
	releaseResponseWriter(w)
}

// Reset clears headers, sets the status code to 200
// and clears the cached body.
//
// Implements the `ResponseWriterReseter`.
func (w *responseWriter) Reset() bool {
	if w.written > 0 {
		return false // if already written we can't reset this type of response writer.
	}

	h := w.Header()
	for k := range h {
		h[k] = nil
	}

	w.written = NoWritten
	w.statusCode = defaultStatusCode
	return true
}

// SetWritten sets manually a value for written, it can be
// NoWritten(-1) or StatusCodeWritten(0), > 0 means body length which is useless here.
func (w *responseWriter) SetWritten(n int) {
	if n >= NoWritten && n <= StatusCodeWritten {
		w.written = n
	}
}

// Written should returns the total length of bytes that were being written to the client.
// In addition iris provides some variables to help low-level actions:
// NoWritten, means that nothing were written yet and the response writer is still live.
// StatusCodeWritten, means that status code were written but no other bytes are written to the client, response writer may closed.
// > 0 means that the reply was written and it's the total number of bytes were written.
func (w *responseWriter) Written() int {
	return w.written
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *responseWriter) tryWriteHeader() {
	if w.written == NoWritten { // before write, once.
		w.written = StatusCodeWritten
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
}

// IsHijacked reports whether this response writer's connection is hijacked.
func (w *responseWriter) IsHijacked() bool {
	// Note:
	// A zero-byte `ResponseWriter.Write` on a hijacked connection will
	// return `http.ErrHijacked` without any other side effects.
	_, err := w.ResponseWriter.Write(nil)
	return err == http.ErrHijacked
}

// Write writes to the client
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (w *responseWriter) Write(contents []byte) (int, error) {
	w.tryWriteHeader()
	n, err := w.ResponseWriter.Write(contents)
	w.written += n
	return n, err
}

// StatusCode returns the status code header value
func (w *responseWriter) StatusCode() int {
	return w.statusCode
}

func (w *responseWriter) GetBeforeFlush() func() {
	return w.beforeFlush
}

// SetBeforeFlush registers the unique callback which called exactly before the response is flushed to the client
func (w *responseWriter) SetBeforeFlush(cb func()) {
	w.beforeFlush = cb
}

func (w *responseWriter) FlushResponse() {
	if w.beforeFlush != nil {
		w.beforeFlush()
	}

	w.tryWriteHeader()
}

// Clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder.
func (w *responseWriter) Clone() ResponseWriter {
	wc := &responseWriter{}
	wc.ResponseWriter = w.ResponseWriter
	wc.statusCode = w.statusCode
	wc.beforeFlush = w.beforeFlush
	wc.written = w.written
	return wc
}

// CopyTo writes a response writer (temp: status code, headers and body) to another response writer.
func (w *responseWriter) CopyTo(to ResponseWriter) {
	// set the status code, failure status code are first class
	if w.statusCode >= 400 {
		to.WriteHeader(w.statusCode)
	}

	// append the headers
	for k, values := range w.Header() {
		for _, v := range values {
			if to.Header().Get(v) == "" {
				to.Header().Add(k, v)
			}
		}
	}
	// the body is not copied, this writer doesn't support recording
}

// ErrHijackNotSupported is returned by the Hijack method to
// indicate that Hijack feature is not available.
var ErrHijackNotSupported = errors.New("hijack is not supported by this ResponseWriter")

// Hijack lets the caller take over the connection.
// After a call to Hijack(), the HTTP server library
// will not do anything else with the connection.
//
// It becomes the caller's responsibility to manage
// and close the connection.
//
// The returned net.Conn may have read or write deadlines
// already set, depending on the configuration of the
// Server. It is the caller's responsibility to set
// or clear those deadlines as needed.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, isHijacker := w.ResponseWriter.(http.Hijacker); isHijacker {
		w.written = StatusCodeWritten
		return h.Hijack()
	}

	return nil, nil, ErrHijackNotSupported
}

// Flusher indicates if `Flush` is supported by the client.
//
// The default HTTP/1.x and HTTP/2 ResponseWriter implementations
// support Flusher, but ResponseWriter wrappers may not. Handlers
// should always test for this ability at runtime.
//
// Note that even for ResponseWriters that support Flush,
// if the client is connected through an HTTP proxy,
// the buffered data may not reach the client until the response
// completes.
func (w *responseWriter) Flusher() (http.Flusher, bool) {
	flusher, canFlush := w.ResponseWriter.(http.Flusher)
	return flusher, canFlush
}

// Flush sends any buffered data to the client.
func (w *responseWriter) Flush() {
	if flusher, ok := w.Flusher(); ok {
		// Flow: WriteHeader -> Flush -> Write -> Write -> Write....
		w.tryWriteHeader()

		flusher.Flush()
	}
}
