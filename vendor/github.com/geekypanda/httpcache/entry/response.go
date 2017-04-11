package entry

// Response is the cached response will be send to the clients
// its fields setted at runtime on each of the non-cached executions
// non-cached executions = first execution, and each time after
// cache expiration datetime passed
type Response struct {
	// statusCode for the response cache handler
	statusCode int
	// contentType for the response cache handler
	contentType string
	// body is the contents will be served by the cache handler
	body []byte
}

// StatusCode returns a valid status code
func (r *Response) StatusCode() int {
	if r.statusCode <= 0 {
		r.statusCode = 200
	}
	return r.statusCode
}

// ContentType returns a valid content type
func (r *Response) ContentType() string {
	if r.contentType == "" {
		r.contentType = "text/html; charset=utf-8"
	}
	return r.contentType
}

// Body returns contents will be served by the cache handler
func (r *Response) Body() []byte {
	return r.body
}
