package context

import (
	"io"
	"sync"

	"github.com/klauspost/compress/gzip"
)

// compressionPool is a wrapper of sync.Pool, to initialize a new compression writer pool
type compressionPool struct {
	sync.Pool
	Level int
}

//  +------------------------------------------------------------+
//  |GZIP raw io.writer, our gzip response writer will use that. |
//  +------------------------------------------------------------+

// default writer pool with Compressor's level setted to -1
var gzipPool = &compressionPool{Level: -1}

// acquireGzipWriter prepares a gzip writer and returns it.
//
// see releaseGzipWriter too.
func acquireGzipWriter(w io.Writer) *gzip.Writer {
	v := gzipPool.Get()
	if v == nil {
		gzipWriter, err := gzip.NewWriterLevel(w, gzipPool.Level)
		if err != nil {
			return nil
		}
		return gzipWriter
	}
	gzipWriter := v.(*gzip.Writer)
	gzipWriter.Reset(w)
	return gzipWriter
}

// releaseGzipWriter called when flush/close and put the gzip writer back to the pool.
//
// see acquireGzipWriter too.
func releaseGzipWriter(gzipWriter *gzip.Writer) {
	gzipWriter.Close()
	gzipPool.Put(gzipWriter)
}

// writeGzip writes a compressed form of p to the underlying io.Writer. The
// compressed bytes are not necessarily flushed until the Writer is closed.
func writeGzip(w io.Writer, b []byte) (int, error) {
	gzipWriter := acquireGzipWriter(w)
	n, err := gzipWriter.Write(b)
	releaseGzipWriter(gzipWriter)
	return n, err
}

var gzpool = sync.Pool{New: func() interface{} { return &GzipResponseWriter{} }}

// AcquireGzipResponseWriter returns a new *GzipResponseWriter from the pool.
// Releasing is done automatically when request and response is done.
func AcquireGzipResponseWriter() *GzipResponseWriter {
	w := gzpool.Get().(*GzipResponseWriter)
	return w
}

func releaseGzipResponseWriter(w *GzipResponseWriter) {
	releaseGzipWriter(w.gzipWriter)
	gzpool.Put(w)
}

// GzipResponseWriter is an upgraded response writer which writes compressed data to the underline ResponseWriter.
//
// It's a separate response writer because iris gives you the ability to "fallback" and "roll-back" the gzip encoding if something
// went wrong with the response, and write http errors in plain form instead.
type GzipResponseWriter struct {
	ResponseWriter
	gzipWriter *gzip.Writer
	chunks     []byte
	disabled   bool
}

var _ ResponseWriter = &GzipResponseWriter{}

// BeginGzipResponse accepts a ResponseWriter
// and prepares the new gzip response writer.
// It's being called per-handler, when caller decide
// to change the response writer type.
func (w *GzipResponseWriter) BeginGzipResponse(underline ResponseWriter) {
	w.ResponseWriter = underline
	w.gzipWriter = acquireGzipWriter(w.ResponseWriter)
	w.chunks = w.chunks[0:0]
	w.disabled = false
}

// EndResponse called right before the contents of this
// response writer are flushed to the client.
func (w *GzipResponseWriter) EndResponse() {
	releaseGzipResponseWriter(w)
	w.ResponseWriter.EndResponse()
}

// Write compresses and writes that data to the underline response writer
func (w *GzipResponseWriter) Write(contents []byte) (int, error) {
	// save the contents to serve them (only gzip data here)
	w.chunks = append(w.chunks, contents...)
	return len(w.chunks), nil
}

// FlushResponse validates the response headers in order to be compatible with the gzip written data
// and writes the data to the underline ResponseWriter.
func (w *GzipResponseWriter) FlushResponse() {
	if w.disabled {
		w.ResponseWriter.Write(w.chunks)
		// remove gzip headers: no need, we just add two of them if gzip was enabled, below
		// headers := w.ResponseWriter.Header()
		// headers[contentType] = nil
		// headers["X-Content-Type-Options"] = nil
		// headers[varyHeader] = nil
		// headers[contentEncodingHeader] = nil
		// headers[contentLength] = nil
	} else {
		// if it's not disable write all chunks gzip compressed with the correct response headers.
		w.ResponseWriter.Header().Add(varyHeaderKey, "Accept-Encoding")
		w.ResponseWriter.Header().Set(contentEncodingHeaderKey, "gzip")
		w.gzipWriter.Write(w.chunks) // it writes to the underline ResponseWriter.
	}
	w.ResponseWriter.FlushResponse()
}

// ResetBody resets the response body.
func (w *GzipResponseWriter) ResetBody() {
	w.chunks = w.chunks[0:0]
}

// Disable turns off the gzip compression for the next .Write's data,
// if called then the contents are being written in plain form.
func (w *GzipResponseWriter) Disable() {
	w.disabled = true
}
