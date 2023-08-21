package context

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"sync"
)

// Recorder the middleware to enable response writer recording ( ResponseWriter -> ResponseRecorder)
var Recorder = func(ctx *Context) {
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

// A ResponseRecorder is used mostly for testing
// in order to record and modify, if necessary, the body and status code and headers.
//
// See `context.Recorder“ method too.
type ResponseRecorder struct {
	ResponseWriter

	// keep track of the body written.
	chunks []byte
	// the saved headers
	headers http.Header

	result *http.Response
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
	w.headers = underline.Header().Clone()
	w.result = nil
	w.ResetBody()
}

// EndResponse is auto-called when the whole client's request is done,
// releases the response recorder and its underline ResponseWriter.
func (w *ResponseRecorder) EndResponse() {
	w.ResponseWriter.EndResponse()
	releaseResponseRecorder(w)
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

// Header returns the temporary header map that, on flush response,
// will be sent by the underline's ResponseWriter's WriteHeader method.
func (w *ResponseRecorder) Header() http.Header {
	return w.headers
}

// SetBody overrides the body and sets it to a slice of bytes value.
func (w *ResponseRecorder) SetBody(b []byte) {
	w.chunks = b
}

// SetBodyString overrides the body and sets it to a string value.
func (w *ResponseRecorder) SetBodyString(s string) {
	w.SetBody([]byte(s))
}

// Body returns the body tracked from the writer so far,
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
	w.headers = w.ResponseWriter.Header().Clone()
}

// ClearHeaders clears all headers, both temp and underline's response writer.
func (w *ResponseRecorder) ClearHeaders() {
	w.headers = http.Header{}
	h := w.ResponseWriter.Header()
	for k := range h {
		delete(h, k)
	}
}

// Reset clears headers, sets the status code to 200
// and clears the cached body.
//
// - Use ResetBody() and ResetHeaders() instead to keep compression after reseting.
//
// - Use Reset() & ResponseRecorder.ResponseWriter.(*context.CompressResponseWriter).Disabled = true
// to set a new body without compression when the previous handler was iris.Compression.
//
// Implements the `ResponseWriterReseter`.
func (w *ResponseRecorder) Reset() bool {
	w.ClearHeaders()
	w.WriteHeader(defaultStatusCode)
	w.ResetBody()
	return true
}

// FlushResponse the full body, headers and status code to the underline response writer
// called automatically at the end of each request.
func (w *ResponseRecorder) FlushResponse() {
	// copy the headers to the underline response writer
	if w.headers != nil {
		h := w.ResponseWriter.Header()
		// note: we don't reset the current underline's headers.
		for k, v := range w.headers {
			h[k] = v
		}
	}

	cw, mustWriteToClose := w.ResponseWriter.(*CompressResponseWriter)
	if mustWriteToClose { // see #1569#issuecomment-664003098
		cw.FlushHeaders()
	} else {
		// NOTE: before the ResponseWriter.Write in order to:
		// set the given status code even if the body is empty.
		w.ResponseWriter.FlushResponse()
	}

	if len(w.chunks) > 0 {
		// ignore error
		w.ResponseWriter.Write(w.chunks)
	}

	if mustWriteToClose {
		cw.ResponseWriter.FlushResponse()
		cw.CompressWriter.Close()
	}
}

// Clone returns a clone of this response writer
// it copies the header, status code, headers and the beforeFlush finally  returns a new ResponseRecorder
func (w *ResponseRecorder) Clone() ResponseWriter {
	wc := &ResponseRecorder{}

	// copy headers.
	wc.headers = w.headers.Clone()

	// copy body.
	chunksCopy := make([]byte, len(w.chunks))
	copy(chunksCopy, w.chunks)
	wc.chunks = chunksCopy

	if resW, ok := w.ResponseWriter.(*responseWriter); ok {
		wc.ResponseWriter = &responseWriter{
			ResponseWriter: resW.ResponseWriter,
			statusCode:     resW.statusCode,
			written:        resW.written,
			beforeFlush:    resW.beforeFlush,
		} // clone it
	} else { // else just copy, may pointer, developer can change its behavior
		wc.ResponseWriter = w.ResponseWriter
	}

	return wc
}

// CopyTo writes a response writer (temp: status code, headers and body) to another response writer
func (w *ResponseRecorder) CopyTo(res ResponseWriter) {
	if to, ok := res.(*ResponseRecorder); ok {

		// set the status code, to is first ( probably an error? (context.StatusCodeNotSuccessful, defaults to >=400).
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
		for k, values := range w.headers {
			for _, v := range values {
				if to.headers.Get(v) == "" {
					to.headers.Add(k, v)
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
	// This fixes response recorder when chunked + Flush is used.
	if w.headers.Get("Transfer-Encoding") == "chunked" {
		if w.Written() == NoWritten {
			if len(w.headers) > 0 {
				h := w.ResponseWriter.Header()
				// note: we don't reset the current underline's headers.
				for k, v := range w.headers {
					h[k] = v
				}
			}
		}

		if len(w.chunks) > 0 {
			w.ResponseWriter.Write(w.chunks)
		}
	}

	w.ResponseWriter.Flush()
	w.ResetBody()
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
func (w *ResponseRecorder) Push(target string, opts *http.PushOptions) (err error) {
	w.FlushResponse()

	if pusher, ok := w.ResponseWriter.Naive().(http.Pusher); ok {
		err = pusher.Push(target, opts)
		if err != nil && err.Error() == http.ErrNotSupported.ErrorString {
			return ErrPushNotSupported
		}
	}

	// NOTE: we have to reset them even if the push failed.
	w.ResetBody()
	w.ResetHeaders()

	return ErrPushNotSupported
}

// Result returns the response generated by the handler.
// It does set all provided headers.
//
// Result must only be called after the handler has finished running.
func (w *ResponseRecorder) Result() *http.Response { // a modified copy of net/http/httptest
	if w.result != nil {
		return w.result
	}

	headers := w.headers.Clone()

	// for k, v := range w.ResponseWriter.Header() {
	// 	headers[k] = v
	// }
	/*
		dateFound := false
		for k := range headers {
			if strings.ToLower(k) == "date" {
				dateFound = true
				break
			}
		}

		if !dateFound {
			headers["Date"] = []string{time.Now().Format(http.TimeFormat)}
		}
	*/

	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: w.StatusCode(),
		Header:     headers,
	}
	if res.StatusCode == 0 {
		res.StatusCode = 200
	}
	res.Status = fmt.Sprintf("%03d %s", res.StatusCode, http.StatusText(res.StatusCode))
	if w.chunks != nil {
		res.Body = io.NopCloser(bytes.NewReader(w.chunks))
	} else {
		res.Body = http.NoBody
	}
	res.ContentLength = parseContentLength(res.Header.Get("Content-Length"))

	w.result = res
	return res
}

// copy of net/http/httptest
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
