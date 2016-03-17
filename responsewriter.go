package iris

import (
	"net/http"
)

type ResponseHandler func(ResponseWriter)

type ResponseMiddleware []ResponseHandler

type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	Written() bool
	Size() int
	PreWrite(ResponseMiddleware)
	apply(res http.ResponseWriter)
}

//implement the ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	status     int
	size       int
	middleware ResponseMiddleware
}

// Crea returns a new ResponseWriter that is wrapped from a http.ResponseWriter
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

func (res *responseWriter) ForceWriteHeader(status int) {

	res.status = status
	mlen := len(res.middleware) - 1
	if res.middleware != nil {
		for i := 0; i < mlen; i++ {
			res.middleware[i](res)
		}
	}
	res.size = 0
	res.ResponseWriter.WriteHeader(status)

}

func (res *responseWriter) WriteHeader(status int) {
	res.status = status
}

func (res *responseWriter) Written() bool {
	return res.status != 0
}

//implement the http.ResponseWriter

func (res *responseWriter) Write(b []byte) (int, error) {
	//if headers not setted we assume that's it's 200
	if !res.Written() {
		res.ForceWriteHeader(http.StatusOK)
	}
	//write to the underline http.ResponseWriter
	size, err := res.ResponseWriter.Write(b)
	res.size += size
	return size, err
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
