package fs

import (
	"io"
	"sync"

	"github.com/klauspost/compress/gzip"
)

// writes gzip compressed content to an underline io.Writer. It uses sync.Pool to reduce memory allocations.
// Better performance through klauspost/compress package which provides us a gzip.Writer which is faster than Go standard's gzip package's writer.

// These constants are copied from the standard flate package
// available Compressors
const (
	NoCompression       = 0
	BestSpeed           = 1
	BestCompression     = 9
	DefaultCompression  = -1
	ConstantCompression = -2 // Does only Huffman encoding
)

// GzipPool is a wrapper of sync.Pool, to initialize a new gzip writer pool, just create a new instance of this iteral, GzipPool{}
type GzipPool struct {
	sync.Pool
	Level int
}

// NewGzipPool returns a new gzip writer pool, ready to use
func NewGzipPool(Level int) *GzipPool {
	return &GzipPool{Level: Level}
}

// DefaultGzipPool returns a new writer pool with Compressor's level setted to DefaultCompression
func DefaultGzipPool() *GzipPool {
	return NewGzipPool(DefaultCompression)
}

// default writer pool with Compressor's level setted to DefaultCompression
var defaultGzipWriterPool = DefaultGzipPool()

// AcquireGzipWriter prepares a gzip writer and returns it
//
// see ReleaseGzipWriter
func AcquireGzipWriter(w io.Writer) *gzip.Writer {
	return defaultGzipWriterPool.AcquireGzipWriter(w)
}

// AcquireGzipWriter prepares a gzip writer and returns it
//
// see ReleaseGzipWriter
func (p *GzipPool) AcquireGzipWriter(w io.Writer) *gzip.Writer {
	v := p.Get()
	if v == nil {
		gzipWriter, err := gzip.NewWriterLevel(w, p.Level)
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
func ReleaseGzipWriter(gzipWriter *gzip.Writer) {
	defaultGzipWriterPool.ReleaseGzipWriter(gzipWriter)
}

// ReleaseGzipWriter called when flush/close and put the gzip writer back to the pool
//
// see AcquireGzipWriter
func (p *GzipPool) ReleaseGzipWriter(gzipWriter *gzip.Writer) {
	gzipWriter.Close()
	p.Put(gzipWriter)
}

// WriteGzip writes a compressed form of p to the underlying io.Writer. The
// compressed bytes are not necessarily flushed until the Writer is closed
func WriteGzip(w io.Writer, b []byte) (int, error) {
	return defaultGzipWriterPool.WriteGzip(w, b)
}

// WriteGzip writes a compressed form of p to the underlying io.Writer. The
// compressed bytes are not necessarily flushed until the Writer is closed
func (p *GzipPool) WriteGzip(w io.Writer, b []byte) (int, error) {
	gzipWriter := p.AcquireGzipWriter(w)
	n, err := gzipWriter.Write(b)
	p.ReleaseGzipWriter(gzipWriter)
	return n, err
}
