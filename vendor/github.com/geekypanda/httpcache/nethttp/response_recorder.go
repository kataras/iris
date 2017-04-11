package nethttp

import (
	"net/http"
	"sync"
)

var rpool = sync.Pool{}

// AcquireResponseRecorder returns a ResponseRecorder
func AcquireResponseRecorder(underline http.ResponseWriter) *ResponseRecorder {
	v := rpool.Get()
	var res *ResponseRecorder
	if v != nil {
		res = v.(*ResponseRecorder)
	} else {
		res = &ResponseRecorder{}
	}
	res.underline = underline
	return res
}

// ReleaseResponseRecorder releases a ResponseRecorder which has been previously received by AcquireResponseRecorder
func ReleaseResponseRecorder(res *ResponseRecorder) {
	res.underline = nil
	res.statusCode = 0
	res.chunks = res.chunks[0:0]
	rpool.Put(res)
}

// ResponseRecorder is used by httpcache to be able to get the Body and the StatusCode of a request handler
type ResponseRecorder struct {
	underline  http.ResponseWriter
	chunks     [][]byte // 2d because .Write can be called more than one time in the same handler and we want to cache all of them
	statusCode int      // the saved status code which will be used from the cache service
}

// Body joins the chunks to one []byte slice, this is the full body
func (res *ResponseRecorder) Body() []byte {
	var body []byte
	for i := range res.chunks {
		body = append(body, res.chunks[i]...)
	}
	return body
}

// ContentType returns the header's value of "Content-Type"
func (res *ResponseRecorder) ContentType() string {
	return res.Header().Get("Content-Type")
}

// StatusCode returns the status code, if not given then returns 200
// but doesn't changes the existing behavior
func (res *ResponseRecorder) StatusCode() int {
	if res.statusCode == 0 {
		return 200
	}
	return res.statusCode
}

// Header returns the header map that will be sent by
// WriteHeader. Changing the header after a call to
// WriteHeader (or Write) has no effect unless the modified
// headers were declared as trailers by setting the
// "Trailer" header before the call to WriteHeader (see example).
// To suppress implicit response headers, set their value to nil.
func (res *ResponseRecorder) Header() http.Header {
	return res.underline.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
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
func (res *ResponseRecorder) Write(contents []byte) (int, error) {
	if res.statusCode == 0 { // if not setted set it here
		res.WriteHeader(http.StatusOK)
	}
	res.chunks = append(res.chunks, contents)
	return res.underline.Write(contents)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (res *ResponseRecorder) WriteHeader(statusCode int) {
	if res.statusCode == 0 { // set it only if not setted already, we don't want logs about multiple sends
		res.statusCode = statusCode
		res.underline.WriteHeader(statusCode)
	}

}
