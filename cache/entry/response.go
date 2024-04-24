package entry

import (
	"io"
	"net/http"
)

// Response is the cached response will be send to the clients
// its fields set at runtime on each of the non-cached executions
// non-cached executions = first execution, and each time after
// cache expiration datetime passed.
type Response struct {
	// statusCode for the response cache handler.
	statusCode int
	// body is the contents will be served by the cache handler.
	body []byte
	// the total headers of the response, including content type.
	headers http.Header
}

// NewResponse returns a new cached Response.
func NewResponse(statusCode int, headers http.Header, body []byte) *Response {
	r := new(Response)

	r.SetStatusCode(statusCode)
	r.SetHeaders(headers)
	r.SetBody(body)

	return r
}

// SetStatusCode sets a valid status code.
func (r *Response) SetStatusCode(statusCode int) {
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}

	r.statusCode = statusCode
}

// StatusCode returns a valid status code.
func (r *Response) StatusCode() int {
	return r.statusCode
}

// ContentType returns a valid content type
// func (r *Response) ContentType() string {
// 	if r.headers == "" {
// 		r.contentType = "text/html; charset=utf-8"
// 	}
// 	return r.contentType
// }

// SetHeaders sets a clone of headers of the cached response.
func (r *Response) SetHeaders(h http.Header) {
	r.headers = h.Clone()
}

// Headers returns the total headers of the cached response.
func (r *Response) Headers() http.Header {
	return r.headers
}

// SetBody consumes "b" and sets the body of the cached response.
func (r *Response) SetBody(body []byte) {
	r.body = make([]byte, len(body))
	copy(r.body, body)
}

// Body returns contents will be served by the cache handler.
func (r *Response) Body() []byte {
	return r.body
}

// Read implements the io.Reader interface.
func (r *Response) Read(b []byte) (int, error) {
	if len(r.body) == 0 {
		return 0, io.EOF
	}

	n := copy(b, r.body)
	r.body = r.body[n:]
	return n, nil
}

// Bytes returns a copy of the cached response body.
func (r *Response) Bytes() []byte {
	return append([]byte(nil), r.body...)
}
