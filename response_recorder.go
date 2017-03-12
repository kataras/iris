package iris

import (
	"fmt"
	"net/http"
	"sync"
)

// Recorder the middleware to enable response writer recording ( *responseWriter -> *ResponseRecorder)
var Recorder = HandlerFunc(func(ctx *Context) {
	ctx.Record()
	ctx.Next()
})

var rrpool = sync.Pool{New: func() interface{} { return &ResponseRecorder{} }}

func acquireResponseRecorder(underline *responseWriter) *ResponseRecorder {
	w := rrpool.Get().(*ResponseRecorder)
	w.responseWriter = underline
	w.headers = underline.Header()
	w.ResetBody()
	return w
}

func releaseResponseRecorder(w *ResponseRecorder) {
	if w.responseWriter != nil {
		releaseResponseWriter(w.responseWriter)
	}
	rrpool.Put(w)
}

// A ResponseRecorder is used mostly by context's transactions
// in order to record and change if needed the body, status code and headers.
//
// You are NOT limited to use that too:
// just call context.ResponseWriter.Recorder()/Record() and
// response writer will act like context.ResponseWriter.(*iris.ResponseRecorder)
type ResponseRecorder struct {
	*responseWriter
	// these three fields are setted on flushBody which runs only once on the end of the handler execution.
	// this helps the performance on multi-write and keep tracks the body, status code and headers in order to run each transaction
	// on its own
	chunks  []byte      // keep track of the body in order to be resetable and useful inside custom transactions
	headers http.Header // the saved headers
}

var _ ResponseWriter = &ResponseRecorder{}

// Header returns the header map that will be sent by
// WriteHeader. Changing the header after a call to
// WriteHeader (or Write) has no effect unless the modified
// headers were declared as trailers by setting the
// "Trailer" header before the call to WriteHeader (see example).
// To suppress implicit response headers, set their value to nil.
func (w *ResponseRecorder) Header() http.Header {
	return w.headers
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered
func (w *ResponseRecorder) Writef(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered
func (w *ResponseRecorder) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
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
func (w *ResponseRecorder) Write(contents []byte) (int, error) {
	w.chunks = append(w.chunks, contents...)
	return len(w.chunks), nil
}

// Body returns the body tracked from the writer so far
// do not use this for edit.
func (w *ResponseRecorder) Body() []byte {
	return w.chunks
}

// SetBodyString overrides the body and sets it to a string value
func (w *ResponseRecorder) SetBodyString(s string) {
	w.chunks = []byte(s)
}

// SetBody overrides the body and sets it to a slice of bytes value
func (w *ResponseRecorder) SetBody(b []byte) {
	w.chunks = b
}

// ResetBody resets the response body
func (w *ResponseRecorder) ResetBody() {
	w.chunks = w.chunks[0:0]
}

// ResetHeaders clears the temp headers
func (w *ResponseRecorder) ResetHeaders() {
	// original response writer's headers are empty.
	w.headers = w.responseWriter.Header()
}

// ReseAllHeaders clears all headers, both temp and underline's response writer
func (w *ResponseRecorder) ReseAllHeaders() {
	w.headers = http.Header{}
	h := w.responseWriter.Header()
	for k := range h {
		h[k] = nil
	}
}

// Reset resets the response body, headers and the status code header
func (w *ResponseRecorder) Reset() {
	w.ReseAllHeaders()
	w.statusCode = StatusOK
	w.ResetBody()
}

// flushResponse the full body, headers and status code to the underline response writer
// called automatically at the end of each request, see ReleaseCtx
func (w *ResponseRecorder) flushResponse() {
	if w.headers != nil {
		h := w.responseWriter.Header()

		for k, values := range w.headers {
			h[k] = nil
			for i := range values {
				h.Add(k, values[i])
			}
		}
	}

	// NOTE: before the responseWriter.Writer in order to:
	// 1. execute the beforeFlush if != nil
	// 2. set the status code before the .Write method overides that
	w.responseWriter.flushResponse()

	if len(w.chunks) > 0 {
		// ignore error
		w.responseWriter.Write(w.chunks)
	}
}

// Flush sends any buffered data to the client.
func (w *ResponseRecorder) Flush() {
	w.responseWriter.Flush()
	w.ResetBody()
}

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
func (w *ResponseRecorder) Push(target string, opts *http.PushOptions) error {
	w.flushResponse()
	err := w.responseWriter.Push(target, opts)
	// NOTE: we have to reset them even if the push failed.
	w.ResetBody()
	w.ResetHeaders()

	return err
}

// clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder
func (w *ResponseRecorder) clone() ResponseWriter {
	wc := &ResponseRecorder{}
	wc.headers = w.headers
	wc.chunks = w.chunks[0:]
	wc.responseWriter = &(*w.responseWriter) // w.responseWriter.clone().(*responseWriter) //
	return wc
}

// writeTo writes a response writer (temp: status code, headers and body) to another response writer
func (w *ResponseRecorder) writeTo(res ResponseWriter) {

	if to, ok := res.(*ResponseRecorder); ok {

		// set the status code, to is first ( probably an error >=400)
		if w.statusCode == StatusOK {
			to.statusCode = w.statusCode
		}

		if w.beforeFlush != nil {
			// if to had a before flush, lets combine them
			if to.beforeFlush != nil {
				nextBeforeFlush := w.beforeFlush
				prevBeforeFlush := to.beforeFlush
				to.beforeFlush = func() {
					prevBeforeFlush()
					nextBeforeFlush()
				}
			} else {
				to.beforeFlush = w.beforeFlush
			}
		}

		if !to.statusCodeSent {
			to.statusCodeSent = w.statusCodeSent
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
			// ignore error
			to.Write(w.chunks)
		}

	}
}

func (w *ResponseRecorder) releaseMe() {
	releaseResponseRecorder(w)
}
