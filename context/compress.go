package context

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2" // snappy output but likely faster decompression.
)

// The available builtin compression algorithms.
const (
	GZIP    = "gzip"
	DEFLATE = "deflate"
	BROTLI  = "br"
	SNAPPY  = "snappy"
	S2      = "s2"
)

// IDENTITY no transformation whatsoever.
const IDENTITY = "identity"

var (
	// ErrResponseNotCompressed returned from AcquireCompressResponseWriter
	// when response's Content-Type header is missing due to golang/go/issues/31753 or
	// when accept-encoding is empty. The caller should fallback to the original response writer.
	ErrResponseNotCompressed = errors.New("compress: response will not be compressed")
	// ErrRequestNotCompressed returned from NewCompressReader
	// when request is not compressed.
	ErrRequestNotCompressed = errors.New("compress: request is not compressed")
	// ErrNotSupportedCompression returned from
	// AcquireCompressResponseWriter, NewCompressWriter and NewCompressReader
	// when the request's Accept-Encoding was not found in the server's supported
	// compression algorithms. Check that error with `errors.Is`.
	ErrNotSupportedCompression = errors.New("compress: unsupported compression")
)

// AllEncodings is a slice of default content encodings.
// See `AcquireCompressResponseWriter`.
var AllEncodings = []string{GZIP, DEFLATE, BROTLI, SNAPPY}

// GetEncoding extracts the best available encoding from the request.
func GetEncoding(r *http.Request, offers []string) (string, error) {
	acceptEncoding := r.Header[AcceptEncodingHeaderKey]

	if len(acceptEncoding) == 0 {
		return "", ErrResponseNotCompressed
	}

	encoding := negotiateAcceptHeader(acceptEncoding, offers, IDENTITY)
	if encoding == "" {
		return "", fmt.Errorf("%w: %s", ErrNotSupportedCompression, encoding)
	}

	return encoding, nil
}

type (
	noOpWriter struct{}

	noOpReadCloser struct {
		io.Reader
	}
)

var (
	_ io.ReadCloser = (*noOpReadCloser)(nil)
	_ io.Writer     = (*noOpWriter)(nil)
)

func (w *noOpWriter) Write(p []byte) (int, error) { return 0, nil }

func (r *noOpReadCloser) Close() error {
	return nil
}

// CompressWriter is an interface which all compress writers should implement.
type CompressWriter interface {
	io.WriteCloser
	// All known implementations contain `Flush` and `Reset`  methods,
	// so we wanna declare them upfront.
	Flush() error
	Reset(io.Writer)
}

// NewCompressWriter returns a CompressWriter of "w" based on the given "encoding".
func NewCompressWriter(w io.Writer, encoding string, level int) (cw CompressWriter, err error) {
	switch encoding {
	case GZIP:
		cw, err = gzip.NewWriterLevel(w, level)
	case DEFLATE: // -1 default level, same for gzip.
		cw, err = flate.NewWriter(w, level)
	case BROTLI: // 6 default level.
		if level == -1 {
			level = 6
		}
		cw = brotli.NewWriterLevel(w, level)
	case SNAPPY:
		cw = snappy.NewBufferedWriter(w)
	case S2:
		cw = s2.NewWriter(w)
	default:
		// Throw if "identity" is given. As this is not acceptable on "Content-Encoding" header.
		// Only Accept-Encoding (client) can use that; it means, no transformation whatsoever.
		err = ErrNotSupportedCompression
	}

	return
}

// CompressReader is a structure which wraps a compressed reader.
// It is used for determination across common request body and a compressed one.
type CompressReader struct {
	io.ReadCloser

	// We need this to reset the body to its original state, if requested.
	Src io.ReadCloser
	// Encoding is the compression alogirthm is used to decompress and read the data.
	Encoding string
}

// NewCompressReader returns a new "compressReader" wrapper of "src".
// It returns `ErrRequestNotCompressed` if client's request data are not compressed
// or `ErrNotSupportedCompression` if server missing the decompression algorithm.
// Note: on server-side the request body (src) will be closed automaticaly.
func NewCompressReader(src io.Reader, encoding string) (*CompressReader, error) {
	if encoding == "" || src == nil {
		return nil, ErrRequestNotCompressed
	}

	var (
		rc  io.ReadCloser
		err error
	)

	switch encoding {
	case GZIP:
		rc, err = gzip.NewReader(src)
	case DEFLATE:
		rc = flate.NewReader(src)
	case BROTLI:
		rc = &noOpReadCloser{brotli.NewReader(src)}
	case SNAPPY:
		rc = &noOpReadCloser{snappy.NewReader(src)}
	case S2:
		rc = &noOpReadCloser{s2.NewReader(src)}
	default:
		err = ErrNotSupportedCompression
	}

	if err != nil {
		return nil, err
	}

	srcReadCloser, ok := src.(io.ReadCloser)
	if !ok {
		srcReadCloser = &noOpReadCloser{src}
	}

	return &CompressReader{
		ReadCloser: rc,
		Src:        srcReadCloser,
		Encoding:   encoding,
	}, nil
}

var compressWritersPool = sync.Pool{New: func() interface{} { return &CompressResponseWriter{} }}

// AddCompressHeaders just adds the headers "Vary" to "Accept-Encoding"
// and "Content-Encoding" to the given encoding.
func AddCompressHeaders(h http.Header, encoding string) {
	h.Set(VaryHeaderKey, AcceptEncodingHeaderKey)
	h.Set(ContentEncodingHeaderKey, encoding)
}

// CompressResponseWriter is a compressed data http.ResponseWriter.
type CompressResponseWriter struct {
	CompressWriter
	ResponseWriter

	http.Hijacker

	Disabled bool
	Encoding string
	Level    int
}

var _ ResponseWriter = (*CompressResponseWriter)(nil)

// AcquireCompressResponseWriter returns a CompressResponseWriter from the pool.
// It accepts an Iris response writer, a net/http request value and
// the level of compression (use -1 for default compression level).
//
// It returns the best candidate among "gzip", "defate", "br", "snappy" and "s2"
// based on the request's "Accept-Encoding" header value.
func AcquireCompressResponseWriter(w ResponseWriter, r *http.Request, level int) (*CompressResponseWriter, error) {
	encoding, err := GetEncoding(r, AllEncodings)
	if err != nil {
		return nil, err
	}

	v := compressWritersPool.Get().(*CompressResponseWriter)

	if h, ok := w.(http.Hijacker); ok {
		v.Hijacker = h
	} else {
		v.Hijacker = nil
	}

	// The Naive() should be used to check for Pusher,
	// as examples explicitly says, so don't do it:
	// if p, ok := w.Naive().(http.Pusher); ok {
	// 	v.Pusher = p
	// } else {
	// 	v.Pusher = nil
	// }

	v.ResponseWriter = w
	v.Disabled = false
	if level == -1 && encoding == BROTLI {
		level = 6
	}

	/*
		// Writer exists, encoding matching and it's valid because it has a non nil encWriter;
		// just reset to reduce allocations.
		if v.Encoding == encoding && v.Level == level && v.CompressWriter != nil {
			v.CompressWriter.Reset(w)
			return v, nil
		}
	*/

	v.Encoding = encoding

	v.Level = level
	encWriter, err := NewCompressWriter(w, encoding, level)
	if err != nil {
		return nil, err
	}

	v.CompressWriter = encWriter

	AddCompressHeaders(w.Header(), encoding)
	return v, nil
}

func releaseCompressResponseWriter(w *CompressResponseWriter) {
	compressWritersPool.Put(w)
}

// FlushResponse flushes any data, closes the underline compress writer
// and writes the status code.
// Called automatically before `EndResponse`.
func (w *CompressResponseWriter) FlushResponse() {
	w.FlushHeaders()

	/* this should NEVER happen, see `context.CompressWriter` method.
	if rec, ok := w.ResponseWriter.(*ResponseRecorder); ok {
		// Usecase: record, then compression.
		w.CompressWriter.Close() // flushes and closes.
		rec.FlushResponse()
		return
	}
	*/

	// write the status, after header set and before any flushed content sent.
	w.ResponseWriter.FlushResponse()

	if w.IsHijacked() {
		// net/http docs:
		// It becomes the caller's responsibility to manage
		// and close the connection.
		return
	}

	w.CompressWriter.Close() // flushes and closes.
}

// FlushHeaders deletes the encoding headers if
// the compressed writer was disabled otherwise
// removes the content-length so next callers can re-calculate the correct length.
func (w *CompressResponseWriter) FlushHeaders() {
	if w.Disabled {
		w.Header().Del(VaryHeaderKey)
		w.Header().Del(ContentEncodingHeaderKey)
		w.CompressWriter.Reset(&noOpWriter{})
	} else {
		w.ResponseWriter.Header().Del(ContentLengthHeaderKey)
	}
}

// EndResponse reeases the writers.
func (w *CompressResponseWriter) EndResponse() {
	w.ResponseWriter.EndResponse()
	releaseCompressResponseWriter(w)
}

func (w *CompressResponseWriter) Write(p []byte) (int, error) {
	if w.Disabled {
		// If disabled or the content-type is empty the response will not be compressed (golang/go/issues/31753).
		return w.ResponseWriter.Write(p)
	}

	if w.Header().Get(ContentTypeHeaderKey) == "" {
		w.Header().Set(ContentTypeHeaderKey, http.DetectContentType(p))
	}

	return w.CompressWriter.Write(p)
}

// Flush sends any buffered data to the client.
// Can be called manually.
func (w *CompressResponseWriter) Flush() {
	// if w.Disabled {
	// 	w.Header().Del(VaryHeaderKey)
	// 	w.Header().Del(ContentEncodingHeaderKey)
	// } else {
	// 	w.encWriter.Flush()
	// }

	if !w.Disabled {
		w.CompressWriter.Flush()
	}

	w.ResponseWriter.Flush()
}

// WriteTo writes the "p" to "dest" Writer using the compression that this compress writer was made of.
func (w *CompressResponseWriter) WriteTo(dest io.Writer, p []byte) (int, error) {
	if w.Disabled {
		return dest.Write(p)
	}

	cw, err := NewCompressWriter(dest, w.Encoding, w.Level)
	if err != nil {
		return 0, err
	}
	n, err := cw.Write(p)
	cw.Close()
	return n, err
}

// Reset implements the ResponseWriterReseter interface.
func (w *CompressResponseWriter) Reset() bool {
	if w.Disabled {
		// If it's disabled then the underline one is responsible.
		rs, ok := w.ResponseWriter.(ResponseWriterReseter)
		return ok && rs.Reset()
	}

	w.CompressWriter.Reset(w.ResponseWriter)
	return true
}
