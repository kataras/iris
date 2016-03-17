package iris

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// ResponseHandler is the type of handler for response writer
type ResponseHandler func(ResponseWriter)

// ResponseMiddleware is a slice of ResponseHandler
// think it like a last-to-execute middleware for the context
type ResponseMiddleware []ResponseHandler

// A ResponseWriter interface is used by an HTTP handler to
// construct an HTTP response.
//
// A ResponseWriter may not be used after the Handler.ServeHTTP method
// has returned.
type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	Written() bool
	Size() int
	PreWrite(ResponseMiddleware)
	apply(res http.ResponseWriter)
	WriteString(message string) (s int, err error)
}

//implement the ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	status     int
	size       int
	middleware ResponseMiddleware
}

// NewResponseWriter returns a new ResponseWriter which is just a wrapper for a http.ResponseWriter
func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	return &responseWriter{rw, 0, 0, nil}
}

//the important staff, this is the register of the pre write handlers
func (res *responseWriter) PreWrite(m ResponseMiddleware) {
	//append them from last to first
	res.middleware = append(m, res.middleware...)
}

func (res *responseWriter) clear() {
	res.size = 0
	res.status = 0
	res.middleware = res.middleware[0:0]
	res.ResponseWriter = nil
}

func (res *responseWriter) apply(underlineResponseWriter http.ResponseWriter) {
	res.size = 0
	res.status = 0
	res.ResponseWriter = underlineResponseWriter

}

// ForceWriteHeader runs the responseWriter's middleware and after write the header
func (res *responseWriter) ForceWriteHeader() {
	mlen := len(res.middleware)
	if res.middleware != nil {
		for i := 0; i < mlen; i++ {
			res.middleware[i](res)
		}
	}
	res.WriteHeader(res.status)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (res *responseWriter) WriteHeader(status int) {
	res.status = status
}

// Written checks if status already written
func (res *responseWriter) Written() bool {
	return res.status != 0
}

//implement the http.ResponseWriter

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK)
// before writing the data.  If the Header does not contain a
// Content-Type line, Write adds a Content-Type set to the result of passing
// the initial 512 bytes of written data to DetectContentType.
func (res *responseWriter) Write(b []byte) (int, error) {
	//if headers not setted we assume that's it's 200
	if !res.Written() {
		res.ForceWriteHeader()
	}
	//write to the underline http.ResponseWriter
	size, err := res.ResponseWriter.Write(b)
	res.size += size
	return size, err
}

func (res *responseWriter) WriteString(message string) (s int, err error) {
	res.ForceWriteHeader()
	s, err = io.WriteString(res.ResponseWriter, message)
	res.size += s
	return
}

func (res *responseWriter) Status() int {
	return res.status
}

func (res *responseWriter) Size() int {
	return res.size
}

func (res *responseWriter) CloseNotify() <-chan bool {
	return res.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (res *responseWriter) Flush() {
	res.ResponseWriter.(http.Flusher).Flush()
}

// Implements the http.Hijacker interface
func (res *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if res.size < 0 {
		res.size = 0
	}
	return res.ResponseWriter.(http.Hijacker).Hijack()
}
