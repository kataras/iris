package compress

import (
	"fmt"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2" // snappy output but likely faster decompression.
	"github.com/klauspost/compress/zstd"
)

// NewRequestReader returns a new "Reader" wrapper of the request body.
// The "Reader" is used to decompress the request body data.
// The "Reader" should be closed after use.
func NewRequestReader(r *http.Request) (*Reader, bool, error) {
	encoding, ok, err := GetContentEncoding(r)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	reader, err := NewReader(r.Body, encoding)
	return reader, err == nil, err
}

type noOpReadCloser struct {
	io.Reader
}

var _ io.ReadCloser = (*noOpReadCloser)(nil)

func (r *noOpReadCloser) Close() error {
	return nil
}

// Reader is a structure which wraps a compressed reader.
// It is used for determination across common request body and a compressed one.
type Reader struct {
	io.ReadCloser

	// We need this to reset the body to its original state, if requested.
	Src io.ReadCloser
	// Encoding is the compression alogirthm is used to decompress and read the data.
	Encoding string
}

// NewReader returns a new "Reader" wrapper of "src".
// It returns a wrapped `ErrNotSupportedCompression` if the "encoding" is empty or the server
// is missing its decompression algorithm, check it with errors.Is.
// Note: on server-side the request body (src) will be closed automaticaly.
func NewReader(src io.Reader, encoding string) (*Reader, error) {
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
	case ZSTD:
		var dec *zstd.Decoder
		// Synchronous decoding, no decoder goroutines per request.
		dec, err = zstd.NewReader(src, zstd.WithDecoderConcurrency(1))
		if err == nil {
			// Close must reach the decoder to release its resources,
			// so no noOpReadCloser here.
			rc = dec.IOReadCloser()
		}
	default: // should never happen, as it's already checked by "shouldCompress".
		err = fmt.Errorf("%w: %s", ErrNotSupportedCompression, encoding)
	}

	if err != nil {
		return nil, err
	}

	srcReadCloser, ok := src.(io.ReadCloser)
	if !ok {
		srcReadCloser = &noOpReadCloser{src}
	}

	return &Reader{
		ReadCloser: rc,
		Src:        srcReadCloser,
		Encoding:   encoding,
	}, nil
}
