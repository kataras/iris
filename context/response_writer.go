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
// for developers to change the response writer of the Context via `Context.SetResponseWriter`.
// The rest of the response writers implementations (ResponseRecorder & CompressResponseWriter)
// are coupled to the internal ResponseWriter implementation(*DefaultResponseWriter).
//
// A ResponseWriter may not be used after the Handler
// has returned.
type ResponseWriter interface {
	http.ResponseWriter
	// http.Flusher
	// http.Hijacker

	// Unwrap returns the underlying http.ResponseWriter,
	// it's the original http.ResponseWriter that the first (*DefaultResponseWriter) Iris' response writer wraps.
	//
	// As of go 1.20 can work with http.NewResponseController.
	Unwrap() http.ResponseWriter

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

	// Flush sends any buffered data to the client.
	Flush()

	// Hijack lets the caller take over the connection.
	// After a call to Hijack the HTTP server library
	// will not do anything else with the connection.
	Hijack() (net.Conn, *bufio.ReadWriter, error)

	// Clone returns a copy of this response writer, carrying over the status code,
	// the total number of bytes written and the hijacked flag.
	// The copy wraps the same underlying http.ResponseWriter, therefore the headers
	// are shared with the original one, they are not copied.
	Clone() ResponseWriter
}

// ResponseWriterBodyResetter can be implemented by
// response writers that supports response body overriding
// (e.g. recorder and compressed).
type ResponseWriterBodyResetter interface {
	// ResetBody should reset the body. It reports nothing back, use the
	// ResponseWriterResetter.Reset method when the caller needs to know
	// whether the reset was successful.
	ResetBody()
}

// ResponseWriterDisabler is implemented by response writers that buffer or
// transform the response body and can step out of the chain before anything has
// been written to the client.
//
// It exists for streaming. A writer that is correct for an ordinary response can
// be fatal for a stream: middleware/compress buffers into a compressor, and
// middleware/recorder - which accesslog, httpcache and servertiming all install -
// buffers into a bytes.Buffer. Neither reaches the client on a Flush.
//
// See Context.DisableResponseBuffering, which walks the chain and calls this.
type ResponseWriterDisabler interface {
	// Disable makes this writer inert and returns the writer it wrapped, which
	// the caller installs in its place.
	//
	// It must be safe to call before the first byte, it must be idempotent, and
	// it must neutralise this writer's own deferred release, so that a compressor
	// footer or a replayed status code cannot land in the middle of a live
	// stream afterwards.
	Disable() ResponseWriter
}

// ResponseWriterFlushError can be implemented by response writers that report
// whether a flush actually reached the client.
//
// net/http's ResponseController probes this interface BEFORE http.Flusher, so
// implementing it turns a flush from "always succeeds" into the cheapest
// disconnect signal a long-lived response has. Only a real FIN/RST or a failing
// write cancels a request context, so a client that vanished without one - a
// closed lid, a dropped Wi-Fi link, an evicted NAT entry - otherwise goes
// unnoticed for as long as the kernel keeps retransmitting.
type ResponseWriterFlushError interface {
	FlushError() error
}

// ResponseWriterResetter can be implemented
// by response writers that can clear the whole response
// so a new handler can write into this from the beginning.
// E.g. recorder, compressed (full) and common writer (status code and headers).
type ResponseWriterResetter interface {
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

// ResponseWriterHeaderFlusher can be implemented
// by response writers that can flush the headers
// before the body is being written.
// It's used internally.
// See middleare/compress/response_writer.go
type ResponseWriterHeaderFlusher interface {
	FlushHeaders()
	Unwrap() ResponseWriter
	Close() error
}

//  +------------------------------------------------------------+
//  | Response Writer Implementation                             |
//  +------------------------------------------------------------+

// DefaultResponseWriter is the default response writer,
// it writes directly to the standard http.ResponseWriter.
type DefaultResponseWriter struct {
	http.ResponseWriter

	statusCode int // the saved status code which will be used from the cache service
	// statusCodeSent bool // reply header has been (logically) written | no needed any more as we have a variable to catch total len of written bytes
	written  int  // the total size of bytes were written.
	hijacked bool // reports whether Hijack was called on this writer.
}

var (
	_ ResponseWriter = (*DefaultResponseWriter)(nil)
	_ http.Flusher   = (*DefaultResponseWriter)(nil)
	_ http.Hijacker  = (*DefaultResponseWriter)(nil)
)

var responseWriterPool = sync.Pool{New: func() any { return &DefaultResponseWriter{} }}

// AcquireResponseWriter returns a new *DefaultResponseWriter from the pool.
// Releasing is done automatically when request and response is done.
func AcquireResponseWriter(w http.ResponseWriter) *DefaultResponseWriter {
	v := responseWriterPool.Get().(*DefaultResponseWriter)
	v.ResponseWriter = w
	v.written = NoWritten
	v.statusCode = defaultStatusCode
	v.hijacked = false

	return v
}

// ReleaseResponseWriter resets the response writer and returns it back to the pool.
func ReleaseResponseWriter(w *DefaultResponseWriter) {
	w.tryWriteHeader()

	w.ResponseWriter = nil
	responseWriterPool.Put(w)
}

const (
	defaultStatusCode = http.StatusOK
	// NoWritten (== -1) => when nothing written before
	NoWritten = -1
	// StatusCodeWritten (== 0) => when only status code written
	StatusCodeWritten = 0
)

type rwUnwrapper interface {
	Unwrap() http.ResponseWriter
}

// UnwrapResponseWriter unwraps all the wrappers and returns the simple, underlying and original http.ResponseWriter.
func UnwrapResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	rw := w
	for {
		if t, ok := rw.(rwUnwrapper); ok {
			rw = t.Unwrap()
			continue
		}

		return rw
	}
}

// Unwrap returns the simple, underlying and original http.ResponseWriter
// that backends this response writer.
func (w *DefaultResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Written should returns the total length of bytes that were being written to the client.
// In addition iris provides some variables to help low-level actions:
// NoWritten, means that nothing were written yet and the response writer is still live.
// StatusCodeWritten, means that status code were written but no other bytes are written to the client, response writer may closed.
// > 0 means that the reply was written and it's the total number of bytes were written.
func (w *DefaultResponseWriter) Written() int {
	return w.written
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
//
// A status code outside the valid 100-999 range is ignored, leaving the
// previous one in place. net/http panics on such a value, and it reaches here
// unchecked from Context.StatusCode, which handlers do call with numbers
// derived from user input or from an upstream response.
func (w *DefaultResponseWriter) WriteHeader(statusCode int) {
	if statusCode < 100 || statusCode > 999 {
		return
	}

	w.statusCode = statusCode
}

func (w *DefaultResponseWriter) tryWriteHeader() {
	if w.written == NoWritten { // before write, once.
		w.written = StatusCodeWritten
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
}

// StatusCode returns the status code header value
func (w *DefaultResponseWriter) StatusCode() int {
	return w.statusCode
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
func (w *DefaultResponseWriter) Write(contents []byte) (int, error) {
	w.tryWriteHeader()
	n, err := w.ResponseWriter.Write(contents)
	w.written += n
	return n, err
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
func (w *DefaultResponseWriter) Flusher() (http.Flusher, bool) {
	flusher, canFlush := w.ResponseWriter.(http.Flusher)
	return flusher, canFlush
}

// Flush sends any buffered data to the client.
//
// Use FlushError when the result matters, which on a long-lived response it
// does: a failing flush is the earliest sign that the client has gone.
func (w *DefaultResponseWriter) Flush() {
	_ = w.FlushError()
}

// FlushError sends any buffered data to the client and reports whether it
// reached the transport. It writes the status code first, exactly as Flush does.
//
// It follows the same preference order as http.ResponseController.Flush - a
// FlushError method, then http.Flusher, then one step down the Unwrap chain -
// without allocating a controller for every event of a stream. It returns an
// error matching errors.ErrUnsupported when nothing in the chain can flush,
// which is what happens under http.TimeoutHandler.
func (w *DefaultResponseWriter) FlushError() error {
	// Flow: WriteHeader -> Flush -> Write -> Write -> Write....
	w.tryWriteHeader() // keep it outside of the flusher check, because we may need to just flush the status code and headers.

	return FlushResponseWriter(w.ResponseWriter)
}

// FlushResponseWriter flushes w and reports whether the flush reached the
// transport, preferring a FlushError method over http.Flusher and unwrapping
// until it finds one. It is what ResponseWriter implementations that wrap
// another writer should delegate to.
func FlushResponseWriter(w http.ResponseWriter) error {
	for {
		switch t := w.(type) {
		case interface{ FlushError() error }:
			return t.FlushError()
		case http.Flusher:
			t.Flush()
			return nil
		case rwUnwrapper:
			w = t.Unwrap()
		default:
			return http.ErrNotSupported
		}
	}
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
func (w *DefaultResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, isHijacker := w.ResponseWriter.(http.Hijacker); isHijacker {
		w.written = StatusCodeWritten
		conn, rw, err := h.Hijack()
		if err == nil {
			w.hijacked = true
		}

		return conn, rw, err
	}

	return nil, nil, ErrHijackNotSupported
}

// IsHijacked reports whether this response writer's connection is hijacked.
func (w *DefaultResponseWriter) IsHijacked() bool {
	if w.hijacked {
		return true
	}

	if w.written == NoWritten {
		// Nothing has been sent yet, so the connection cannot be probed without
		// committing the response: a zero-byte Write on a live net/http writer
		// triggers the implicit WriteHeader(200) and flushes the headers, which
		// would silently discard any status code a later handler sets.
		return false
	}

	// The response is already committed, so the probe costs nothing. A zero-byte
	// Write on a hijacked connection returns http.ErrHijacked with no other side
	// effects; this catches a hijack performed on the underlying writer directly.
	_, err := w.ResponseWriter.Write(nil)
	return err == http.ErrHijacked
}

// Clone returns a copy of this response writer, carrying over the status code
// and the number of bytes written so far.
//
// The copy wraps the same underlying http.ResponseWriter, so it is a second
// handle on one connection rather than an independent response: writing to both
// interleaves their output, and neither is usable once the server has finished
// the request. Clone exists so a handler can hand a writer to a goroutine that
// outlives the handler; that goroutine must check Context.IsCanceled first.
func (w *DefaultResponseWriter) Clone() ResponseWriter {
	wc := &DefaultResponseWriter{}
	wc.ResponseWriter = w.ResponseWriter
	wc.statusCode = w.statusCode
	wc.written = w.written
	wc.hijacked = w.hijacked
	return wc
}
