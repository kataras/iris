package context

import (
	"fmt"
	"net/http"
	"sync"
)

// Recorder the middleware to enable response writer recording ( ResponseWriter -> ResponseRecorder)
var Recorder = func(ctx Context) {
	ctx.Record()
	ctx.Next()
}

var rrpool = sync.Pool{New: func() interface{} { return &ResponseRecorder{} }}

// AcquireResponseRecorder returns a new *AcquireResponseRecorder from the pool.
// Releasing is done automatically when request and response is done.
func AcquireResponseRecorder() *ResponseRecorder {
	return rrpool.Get().(*ResponseRecorder)
}

func releaseResponseRecorder(w *ResponseRecorder) {
	rrpool.Put(w)
}

// A ResponseRecorder is used mostly by context's transactions
// in order to record and change if needed the body, status code and headers.
//
// Developers are not limited to manually ask to record a response.
// To turn on the recorder from a Handler,
// rec := context.Recorder()
type ResponseRecorder struct {
	ResponseWriter
	// keep track of the body in order to be
	// resetable and useful inside custom transactions
	chunks []byte
	// the saved headers
	headers http.Header
}

var _ ResponseWriter = (*ResponseRecorder)(nil)

// Naive returns the simple, underline and original http.ResponseWriter
// that backends this response writer.
func (w *ResponseRecorder) Naive() http.ResponseWriter {
	return w.ResponseWriter.Naive()
}

// BeginRecord accepts its parent ResponseWriter and
// prepares itself, the response recorder, to record and send response to the client.
func (w *ResponseRecorder) BeginRecord(underline ResponseWriter) {
	w.ResponseWriter = underline
	w.headers = underline.Header()
	w.ResetBody()
}

// EndResponse is auto-called when the whole client's request is done,
// releases the response recorder and its underline ResponseWriter.
func (w *ResponseRecorder) EndResponse() {
	releaseResponseRecorder(w)
	w.ResponseWriter.EndResponse()
}

// Write Adds the contents to the body reply, it writes the contents temporarily
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
	// Remember that we should not return all the written length within `Write`:
	// see https://github.com/kataras/iris/pull/931
	return len(contents), nil
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered.
func (w *ResponseRecorder) Writef(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered
func (w *ResponseRecorder) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
}

// SetBody overrides the body and sets it to a slice of bytes value.
func (w *ResponseRecorder) SetBody(b []byte) {
	w.chunks = b
}

// SetBodyString overrides the body and sets it to a string value.
func (w *ResponseRecorder) SetBodyString(s string) {
	w.SetBody([]byte(s))
}

// Body returns the body tracked from the writer so far
// do not use this for edit.
func (w *ResponseRecorder) Body() []byte {
	return w.chunks
}

// ResetBody resets the response body.
func (w *ResponseRecorder) ResetBody() {
	w.chunks = w.chunks[0:0]
}

// ResetHeaders sets the headers to the underline's response writer's headers, may empty.
func (w *ResponseRecorder) ResetHeaders() {
	w.headers = w.ResponseWriter.Header()
}

// ClearHeaders clears all headers, both temp and underline's response writer.
func (w *ResponseRecorder) ClearHeaders() {
	w.headers = http.Header{}
	h := w.ResponseWriter.Header()
	for k := range h {
		h[k] = nil
	}
}

// Reset resets the response body, headers and the status code header.
func (w *ResponseRecorder) Reset() {
	w.ClearHeaders()
	w.WriteHeader(defaultStatusCode)
	w.ResetBody()
}

// FlushResponse the full body, headers and status code to the underline response writer
// called automatically at the end of each request.
func (w *ResponseRecorder) FlushResponse() {
	// copy the headers to the underline response writer
	if w.headers != nil {
		h := w.ResponseWriter.Header()

		for k, values := range w.headers {
			h[k] = nil
			for i := range values {
				h.Add(k, values[i])
			}
		}
	}

	// NOTE: before the ResponseWriter.Write in order to:
	// set the given status code even if the body is empty.
	w.ResponseWriter.FlushResponse()

	if len(w.chunks) > 0 {
		// ignore error
		w.ResponseWriter.Write(w.chunks)
	}
}

// Clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder
func (w *ResponseRecorder) Clone() ResponseWriter {
	wc := &ResponseRecorder{}
	wc.headers = w.headers
	wc.chunks = w.chunks[0:]
	if resW, ok := w.ResponseWriter.(*responseWriter); ok {
		wc.ResponseWriter = &(*resW) // clone it
	} else { // else just copy, may pointer, developer can change its behavior
		wc.ResponseWriter = w.ResponseWriter
	}

	return wc
}

// WriteTo writes a response writer (temp: status code, headers and body) to another response writer
func (w *ResponseRecorder) WriteTo(res ResponseWriter) {

	if to, ok := res.(*ResponseRecorder); ok {

		// set the status code, to is first ( probably an error? (context.StatusCodeNotSuccessful, defaults to < 200 || >= 400).
		if statusCode := w.ResponseWriter.StatusCode(); statusCode == defaultStatusCode {
			to.WriteHeader(statusCode)
		}

		if beforeFlush := w.ResponseWriter.GetBeforeFlush(); beforeFlush != nil {
			// if to had a before flush, lets combine them
			if to.GetBeforeFlush() != nil {
				nextBeforeFlush := beforeFlush
				prevBeforeFlush := to.GetBeforeFlush()
				to.SetBeforeFlush(func() {
					prevBeforeFlush()
					nextBeforeFlush()
				})
			} else {
				to.SetBeforeFlush(w.ResponseWriter.GetBeforeFlush())
			}
		}

		// if "to" is *responseWriter and it never written before (if -1),
		// set the "w"'s written length.
		if resW, ok := to.ResponseWriter.(*responseWriter); ok {
			if resW.Written() != StatusCodeWritten {
				resW.written = w.ResponseWriter.Written()
			}
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

// Flush sends any buffered data to the client.
func (w *ResponseRecorder) Flush() {
	w.ResponseWriter.Flush()
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
	w.FlushResponse()
	err := w.ResponseWriter.Push(target, opts)
	// NOTE: we have to reset them even if the push failed.
	w.ResetBody()
	w.ResetHeaders()

	return err
}
