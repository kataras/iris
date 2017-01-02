package iris

import (
	"bufio"
	"net"
	"net/http"
	"sync"

	"github.com/kataras/go-errors"
	"github.com/kataras/go-fs"
	"github.com/klauspost/compress/gzip"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	http.Flusher
	gzipWriter *gzip.Writer
}

var gzpool = sync.Pool{New: func() interface{} { return &gzipResponseWriter{} }}

func acquireGzipResponseWriter(underline http.ResponseWriter) *gzipResponseWriter {
	w := gzpool.Get().(*gzipResponseWriter)
	w.ResponseWriter = underline
	w.gzipWriter = fs.AcquireGzipWriter(w.ResponseWriter)
	return w
}

func releaseGzipResponseWriter(w *gzipResponseWriter) {
	fs.ReleaseGzipWriter(w.gzipWriter)
	gzpool.Put(w)
}

// Write compresses and writes that data to the underline response writer
func (w *gzipResponseWriter) Write(contents []byte) (int, error) {
	return w.gzipWriter.Write(contents)
}

var rpool = sync.Pool{New: func() interface{} { return &ResponseWriter{} }}

func acquireResponseWriter(underline http.ResponseWriter) *ResponseWriter {
	w := rpool.Get().(*ResponseWriter)
	w.ResponseWriter = underline
	w.headers = underline.Header()
	return w
}

func releaseResponseWriter(w *ResponseWriter) {
	w.headers = nil
	w.ResponseWriter = nil
	w.statusCode = 0
	w.beforeFlush = nil
	w.ResetBody()
	rpool.Put(w)
}

// A ResponseWriter interface is used by an HTTP handler to
// construct an HTTP response.
//
// A ResponseWriter may not be used after the Handler.ServeHTTP method
// has returned.
type ResponseWriter struct {
	// yes only one callback, we need simplicity here because on EmitError the beforeFlush events should NOT be cleared
	// but the response is cleared.
	// Sometimes is useful to keep the event,
	// so we keep one func only and let the user decide when he/she wants to override it with an empty func before the EmitError (context's behavior)
	beforeFlush func()
	http.ResponseWriter
	// these three fields are setted on flushBody which runs only once on the end of the handler execution.
	// this helps the performance on multi-write and keep tracks the body, status code and headers in order to run each transaction
	// on its own
	chunks     []byte      // keep track of the body in order to be resetable and useful inside custom transactions
	statusCode int         // the saved status code which will be used from the cache service
	headers    http.Header // the saved headers
}

// Header returns the header map that will be sent by
// WriteHeader. Changing the header after a call to
// WriteHeader (or Write) has no effect unless the modified
// headers were declared as trailers by setting the
// "Trailer" header before the call to WriteHeader (see example).
// To suppress implicit response headers, set their value to nil.
func (w *ResponseWriter) Header() http.Header {
	return w.headers
}

// StatusCode returns the status code header value
func (w *ResponseWriter) StatusCode() int {
	return w.statusCode
}

// Adds the contents to the body reply, it writes the contents temporarily
// to a value in order to be flushed at the end of the request,
// this method give us the opportunity to reset the body if needed.
//
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
func (w *ResponseWriter) Write(contents []byte) (int, error) {
	w.chunks = append(w.chunks, contents...)
	return len(w.chunks), nil
}

// Body returns the body tracked from the writer so far
// do not use this for edit.
func (w *ResponseWriter) Body() []byte {
	return w.chunks
}

// SetBodyString overrides the body and sets it to a string value
func (w *ResponseWriter) SetBodyString(s string) {
	w.chunks = []byte(s)
}

// SetBody overrides the body and sets it to a slice of bytes value
func (w *ResponseWriter) SetBody(b []byte) {
	w.chunks = b
}

// ResetBody resets the response body
func (w *ResponseWriter) ResetBody() {
	w.chunks = w.chunks[0:0]
}

// ResetHeaders clears the temp headers
func (w *ResponseWriter) ResetHeaders() {
	// original response writer's headers are empty.
	w.headers = w.ResponseWriter.Header()
}

// Reset resets the response body, headers and the status code header
func (w *ResponseWriter) Reset() {
	w.ResetHeaders()
	w.statusCode = 0
	w.ResetBody()
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// ContentType returns the content type, if not setted returns empty string
func (w *ResponseWriter) ContentType() string {
	return w.headers.Get(contentType)
}

// SetContentType sets the content type header
func (w *ResponseWriter) SetContentType(cType string) {
	w.headers.Set(contentType, cType)
}

var errHijackNotSupported = errors.New("Hijack is not supported to this response writer!")

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
func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, isHijacker := w.ResponseWriter.(http.Hijacker); isHijacker {
		return h.Hijack()
	}

	return nil, nil, errHijackNotSupported
}

// SetBeforeFlush registers the unique callback which called exactly before the response is flushed to the client
func (w *ResponseWriter) SetBeforeFlush(cb func()) {
	w.beforeFlush = cb
}

// flushResponse the full body, headers and status code to the underline response writer
// called automatically at the end of each request, see ReleaseCtx
func (w *ResponseWriter) flushResponse() {

	if w.beforeFlush != nil {
		w.beforeFlush()
	}

	if w.statusCode > 0 {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}

	if w.headers != nil {
		for k, values := range w.headers {
			for i := range values {
				w.ResponseWriter.Header().Add(k, values[i])
			}
		}
	}

	if len(w.chunks) > 0 {
		w.ResponseWriter.Write(w.chunks)
	}
}

// Flush sends any buffered data to the client.
func (w *ResponseWriter) Flush() {
	w.flushResponse()
	// The Flusher interface is implemented by ResponseWriters that allow
	// an HTTP handler to flush buffered data to the client.
	//
	// The default HTTP/1.x and HTTP/2 ResponseWriter implementations
	// support Flusher, but ResponseWriter wrappers may not. Handlers
	// should always test for this ability at runtime.
	//
	// Note that even for ResponseWriters that support Flush,
	// if the client is connected through an HTTP proxy,
	// the buffered data may not reach the client until the response
	// completes.
	if fl, isFlusher := w.ResponseWriter.(http.Flusher); isFlusher {
		fl.Flush()
	}
}

// clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseWriter
func (w *ResponseWriter) clone() *ResponseWriter {
	wc := &ResponseWriter{}
	wc.ResponseWriter = w.ResponseWriter
	wc.statusCode = w.statusCode
	wc.headers = w.headers
	wc.chunks = w.chunks[0:]
	wc.beforeFlush = w.beforeFlush
	return wc
}

// writeTo writes a response writer (temp: status code, headers and body) to another response writer
func (w *ResponseWriter) writeTo(to *ResponseWriter) {
	// set the status code, failure status code are first class
	if w.statusCode > 0 {
		to.statusCode = w.statusCode
	}

	// append the headers
	if w.headers != nil {
		for k, values := range w.headers {
			for _, v := range values {
				if to.headers.Get(v) == "" {
					to.headers.Add(k, v)
				}
			}
		}

	}

	// append the body
	if len(w.chunks) > 0 {
		to.Write(w.chunks)
	}

	if w.beforeFlush != nil {
		to.SetBeforeFlush(w.beforeFlush)
	}
}
