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

func (res *responseWriter) WriteHeader(status int) {
	res.status = status
	mlen := len(res.middleware) - 1
	if res.middleware != nil {
		for i := 0; i < mlen; i++ {
			res.middleware[i](res)
		}
	}

}

func (res *responseWriter) Written() bool {
	return res.status != 0
}

//implement the http.ResponseWriter

func (res *responseWriter) Write(b []byte) (int, error) {
	if !res.Written() {
		//if the write header not called then we assume that the status will be 200
		res.WriteHeader(http.StatusOK)
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
	flusher, ok := res.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}
