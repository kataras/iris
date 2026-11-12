package compress

import (
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2" // snappy output but likely faster decompression.
	"github.com/klauspost/compress/zstd"
)

type noOpWriter struct{}

var _ io.Writer = (*noOpWriter)(nil)

func (w *noOpWriter) Write(p []byte) (int, error) { return 0, nil }

// Writer is an interface which all compress writers should implement.
type Writer interface {
	io.WriteCloser
	// All known implementations contain `Flush` and `Reset`  methods,
	// so we wanna declare them upfront.
	Flush() error
	Reset(io.Writer)
}

// NewWriter returns a CompressWriter of "w" based on the given "encoding".
func NewWriter(w io.Writer, encoding string, level int) (cw Writer, err error) {
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
	case ZSTD: // 3 default level (zstd CLI scale, see EncoderLevelFromZstd).
		if level == -1 {
			level = 3
		}
		// Single goroutine: the encoder lives for one response,
		// the default GOMAXPROCS workers are not worth spawning per request.
		cw, err = zstd.NewWriter(w,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
			zstd.WithEncoderConcurrency(1))
	default: // should never happen, as it's already checked by "shouldCompress" on response_writer.
		// Throw if "identity" is given. As this is not acceptable on "Content-Encoding" header.
		// Only Accept-Encoding (client) can use that; it means, no transformation whatsoever.
		err = fmt.Errorf("%w: %s", ErrNotSupportedCompression, encoding)
	}

	return
}
