package iris

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/kataras/go-errors"
	"github.com/klauspost/compress/gzip"
)

type gzipResponseWriter struct {
	ResponseWriter
	gzipWriter *gzip.Writer // it contains the underline writer too
	chunks     []byte
	disabled   bool
}

var gzpool = sync.Pool{New: func() interface{} { return &gzipResponseWriter{} }}

func acquireGzipResponseWriter(underline ResponseWriter) *gzipResponseWriter {
	w := gzpool.Get().(*gzipResponseWriter)
	w.ResponseWriter = underline
	w.gzipWriter = acquireGzipWriter(w.ResponseWriter)
	w.chunks = w.chunks[0:0]
	w.disabled = false
	return w
}

func releaseGzipResponseWriter(w *gzipResponseWriter) {
	releaseGzipWriter(w.gzipWriter)
	gzpool.Put(w)
}

// Write compresses and writes that data to the underline response writer
func (w *gzipResponseWriter) Write(contents []byte) (int, error) {
	// save the contents to serve them (only gzip data here)
	w.chunks = append(w.chunks, contents...)
	return len(w.chunks), nil
}

func (w *gzipResponseWriter) flushResponse() {
	if w.disabled {
		w.ResponseWriter.Write(w.chunks)
	} else {
		// if it's not disable write all chunks gzip compressed
		w.gzipWriter.Write(w.chunks) // it writes to the underline responseWriter (look acquireGzipResponseWriter)
	}
	w.ResponseWriter.flushResponse()
}

func (w *gzipResponseWriter) ResetBody() {
	w.chunks = w.chunks[0:0]
}

func (w *gzipResponseWriter) Disable() {
	w.disabled = true
}

func (w *gzipResponseWriter) releaseMe() {
	releaseGzipResponseWriter(w)
	w.ResponseWriter.releaseMe()
}

var rpool = sync.Pool{New: func() interface{} { return &responseWriter{statusCode: StatusOK} }}

func acquireResponseWriter(underline http.ResponseWriter) *responseWriter {
	w := rpool.Get().(*responseWriter)
	w.ResponseWriter = underline
	return w
}

func releaseResponseWriter(w *responseWriter) {
	w.statusCodeSent = false
	w.beforeFlush = nil
	w.statusCode = StatusOK
	rpool.Put(w)
}

// ResponseWriter interface is used by the context to serve an HTTP handler to
// construct an HTTP response.
//
// A ResponseWriter may not be used after the Handler.ServeHTTP method
// has returned.
type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	http.Hijacker
	http.CloseNotifier
	// breaks go 1.7 as well as the *PushOptions.
	// New users should upgrade to 1.8 if they want to use Iris.
	http.Pusher

	Writef(format string, a ...interface{}) (n int, err error)
	WriteString(s string) (n int, err error)
	SetContentType(cType string)
	ContentType() string
	StatusCode() int
	SetBeforeFlush(cb func())
	flushResponse()
	clone() ResponseWriter
	writeTo(ResponseWriter)
	releaseMe()
}

// responseWriter is the basic response writer,
// it writes directly to the underline http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode     int  // the saved status code which will be used from the cache service
	statusCodeSent bool // reply header has been (logically) written
	// yes only one callback, we need simplicity here because on EmitError the beforeFlush events should NOT be cleared
	// but the response is cleared.
	// Sometimes is useful to keep the event,
	// so we keep one func only and let the user decide when he/she wants to override it with an empty func before the EmitError (context's behavior)
	beforeFlush func()
}

var _ ResponseWriter = &responseWriter{}

// StatusCode returns the status code header value
func (w *responseWriter) StatusCode() int {
	return w.statusCode
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered
func (w *responseWriter) Writef(format string, a ...interface{}) (n int, err error) {
	w.tryWriteHeader()
	return fmt.Fprintf(w.ResponseWriter, format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered
func (w *responseWriter) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
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
	return w.ResponseWriter.Write(contents)
}

// prin to write na benei to write header
// meta to write den ginete edw
// prepei omws kai mono me WriteHeader kai xwris Write na pigenei to status code
// ara...wtf prepei na exw function flushStatusCode kai na elenxei an exei dw9ei status code na to kanei write aliws 200

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// SetBeforeFlush registers the unique callback which called exactly before the response is flushed to the client
func (w *responseWriter) SetBeforeFlush(cb func()) {
	w.beforeFlush = cb
}

func (w *responseWriter) flushResponse() {
	if w.beforeFlush != nil {
		w.beforeFlush()
	}
	w.tryWriteHeader()
}

func (w *responseWriter) tryWriteHeader() {
	if !w.statusCodeSent { // by write
		w.statusCodeSent = true
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
}

// ContentType returns the content type, if not setted returns empty string
func (w *responseWriter) ContentType() string {
	return w.ResponseWriter.Header().Get(contentType)
}

// SetContentType sets the content type header
func (w *responseWriter) SetContentType(cType string) {
	w.ResponseWriter.Header().Set(contentType, cType)
}

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
		w.statusCodeSent = true
		return h.Hijack()
	}

	return nil, nil, errors.New("hijack is not supported by this ResponseWriter")
}

// Flush sends any buffered data to the client.
func (w *responseWriter) Flush() {
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

// ErrPushNotSupported is returned by the Push method to
// indicate that HTTP/2 Push support is not available.
var ErrPushNotSupported = errors.New("push feature is not supported by this ResponseWriter")

// Push initiates an HTTP/2 server push. This constructs a synthetic
// request using the given target and options, serializes that request
// into a PUSH_PROMISE frame, then dispatches that request using the
// server's request handler. If opts is nil, default options are used.
//
// The target must either be an absolute path (like "/path") or an absolute
// URL that contains a valid host and the same scheme as the parent request.
// If the target is a path, it will inherit the scheme and host of the
// parent request.
//
// The HTTP/2 spec disallows recursive pushes and cross-authority pushes.
// Push may or may not detect these invalid pushes; however, invalid
// pushes will be detected and canceled by conforming clients.
//
// Handlers that wish to push URL X should call Push before sending any
// data that may trigger a request for URL X. This avoids a race where the
// client issues requests for X before receiving the PUSH_PROMISE for X.
//
// Push returns ErrPushNotSupported if the client has disabled push or if push
// is not supported on the underlying connection.
func (w *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, isPusher := w.ResponseWriter.(http.Pusher); isPusher {
		err := pusher.Push(target, opts)
		if err != nil && err.Error() == http.ErrNotSupported.ErrorString {
			return ErrPushNotSupported
		}
		return err
	}
	return ErrPushNotSupported
}

// CloseNotify returns a channel that receives at most a
// single value (true) when the client connection has gone
// away.
//
// CloseNotify may wait to notify until Request.Body has been
// fully read.
//
// After the Handler has returned, there is no guarantee
// that the channel receives a value.
//
// If the protocol is HTTP/1.1 and CloseNotify is called while
// processing an idempotent request (such a GET) while
// HTTP/1.1 pipelining is in use, the arrival of a subsequent
// pipelined request may cause a value to be sent on the
// returned channel. In practice HTTP/1.1 pipelining is not
// enabled in browsers and not seen often in the wild. If this
// is a problem, use HTTP/2 or only use CloseNotify on methods
// such as POST.
func (w *responseWriter) CloseNotify() <-chan bool {
	if notifier, supportsCloseNotify := w.ResponseWriter.(http.CloseNotifier); supportsCloseNotify {
		return notifier.CloseNotify()
	}
	ch := make(chan bool, 1)
	return ch
}

// clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder
func (w *responseWriter) clone() ResponseWriter {
	wc := &responseWriter{}
	wc.ResponseWriter = w.ResponseWriter
	wc.statusCode = w.statusCode
	wc.beforeFlush = w.beforeFlush
	wc.statusCodeSent = w.statusCodeSent
	return wc
}

// writeTo writes a response writer (temp: status code, headers and body) to another response writer
func (w *responseWriter) writeTo(to ResponseWriter) {
	// set the status code, failure status code are first class
	if w.statusCode >= 400 {
		to.WriteHeader(w.statusCode)
	}

	// append the headers
	if w.Header() != nil {
		for k, values := range w.Header() {
			for _, v := range values {
				if to.Header().Get(v) == "" {
					to.Header().Add(k, v)
				}
			}
		}

	}
	// the body is not copied, this writer doesn't supports recording
}

func (w *responseWriter) releaseMe() {
	releaseResponseWriter(w)
}
