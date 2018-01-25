package entry

import "net/http"

// Response is the cached response will be send to the clients
// its fields setted at runtime on each of the non-cached executions
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

// StatusCode returns a valid status code.
func (r *Response) StatusCode() int {
	if r.statusCode <= 0 {
		r.statusCode = 200
	}
	return r.statusCode
}

// ContentType returns a valid content type
// func (r *Response) ContentType() string {
// 	if r.headers == "" {
// 		r.contentType = "text/html; charset=utf-8"
// 	}
// 	return r.contentType
// }

// Headers returns the total headers of the cached response.
func (r *Response) Headers() http.Header {
	return r.headers
}

// Body returns contents will be served by the cache handler.
func (r *Response) Body() []byte {
	return r.body
}
