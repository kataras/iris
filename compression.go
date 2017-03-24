package iris

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
//  |                                                            |
//  |                      GZIP                                  |
//  |                                                            |
//  +------------------------------------------------------------+

// writes gzip compressed content to an underline io.Writer. It uses sync.Pool to reduce memory allocations.
// Better performance through klauspost/compress package which provides us a gzip.Writer which is faster than Go standard's gzip package's writer.

// These constants are copied from the standard flate package
// available Compressors
const (
	NoCompressionLevel       = 0
	BestSpeedLevel           = 1
	BestCompressionLevel     = 9
	DefaultCompressionLevel  = -1
	ConstantCompressionLevel = -2 // Does only Huffman encoding
)

// default writer pool with Compressor's level setted to DefaultCompressionLevel
var gzipPool = &compressionPool{Level: DefaultCompressionLevel}

// AcquireGzipWriter prepares a gzip writer and returns it
//
// see ReleaseGzipWriter
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

// ReleaseGzipWriter called when flush/close and put the gzip writer back to the pool
//
// see AcquireGzipWriter
func releaseGzipWriter(gzipWriter *gzip.Writer) {
	gzipWriter.Close()
	gzipPool.Put(gzipWriter)
}

// WriteGzip writes a compressed form of p to the underlying io.Writer. The
// compressed bytes are not necessarily flushed until the Writer is closed
func writeGzip(w io.Writer, b []byte) (int, error) {
	gzipWriter := acquireGzipWriter(w)
	n, err := gzipWriter.Write(b)
	releaseGzipWriter(gzipWriter)
	return n, err
}
