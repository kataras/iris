package fasthttp

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
)

// Supported compression levels.
const (
	CompressNoCompression      = flate.NoCompression
	CompressBestSpeed          = flate.BestSpeed
	CompressBestCompression    = flate.BestCompression
	CompressDefaultCompression = flate.DefaultCompression
)

func acquireGzipReader(r io.Reader) (*gzip.Reader, error) {
	v := gzipReaderPool.Get()
	if v == nil {
		return gzip.NewReader(r)
	}
	zr := v.(*gzip.Reader)
	if err := zr.Reset(r); err != nil {
		return nil, err
	}
	return zr, nil
}

func releaseGzipReader(zr *gzip.Reader) {
	zr.Close()
	gzipReaderPool.Put(zr)
}

var gzipReaderPool sync.Pool

func acquireFlateReader(r io.Reader) (io.ReadCloser, error) {
	v := flateReaderPool.Get()
	if v == nil {
		zr, err := zlib.NewReader(r)
		if err != nil {
			return nil, err
		}
		return zr, nil
	}
	zr := v.(io.ReadCloser)
	if err := resetFlateReader(zr, r); err != nil {
		return nil, err
	}
	return zr, nil
}

func releaseFlateReader(zr io.ReadCloser) {
	zr.Close()
	flateReaderPool.Put(zr)
}

func resetFlateReader(zr io.ReadCloser, r io.Reader) error {
	zrr, ok := zr.(zlib.Resetter)
	if !ok {
		panic("BUG: zlib.Reader doesn't implement zlib.Resetter???")
	}
	return zrr.Reset(r, nil)
}

var flateReaderPool sync.Pool

func acquireGzipWriter(w io.Writer, level int) *gzipWriter {
	p := gzipWriterPoolMap[level]
	if p == nil {
		panic(fmt.Sprintf("BUG: unexpected compression level passed: %d. See compress/gzip for supported levels", level))
	}

	v := p.Get()
	if v == nil {
		zw, err := gzip.NewWriterLevel(w, level)
		if err != nil {
			panic(fmt.Sprintf("BUG: unexpected error from gzip.NewWriterLevel(%d): %s", level, err))
		}
		return &gzipWriter{
			Writer: zw,
			p:      p,
		}
	}
	zw := v.(*gzipWriter)
	zw.Reset(w)
	return zw
}

func releaseGzipWriter(zw *gzipWriter) {
	zw.Close()
	zw.p.Put(zw)
}

type gzipWriter struct {
	*gzip.Writer
	p *sync.Pool
}

var gzipWriterPoolMap = func() map[int]*sync.Pool {
	// Initialize pools for all the compression levels defined
	// in https://golang.org/pkg/compress/gzip/#pkg-constants .
	m := make(map[int]*sync.Pool, 11)
	m[-1] = &sync.Pool{}
	for i := 0; i < 10; i++ {
		m[i] = &sync.Pool{}
	}
	return m
}()

// AppendGzipBytesLevel appends gzipped src to dst using the given
// compression level and returns the resulting dst.
//
// Supported compression levels are:
//
//    * CompressNoCompression
//    * CompressBestSpeed
//    * CompressBestCompression
//    * CompressDefaultCompression
func AppendGzipBytesLevel(dst, src []byte, level int) []byte {
	w := &byteSliceWriter{dst}
	WriteGzipLevel(w, src, level)
	return w.b
}

// WriteGzipLevel writes gzipped p to w using the given compression level
// and returns the number of compressed bytes written to w.
//
// Supported compression levels are:
//
//    * CompressNoCompression
//    * CompressBestSpeed
//    * CompressBestCompression
//    * CompressDefaultCompression
func WriteGzipLevel(w io.Writer, p []byte, level int) (int, error) {
	zw := acquireGzipWriter(w, level)
	n, err := zw.Write(p)
	releaseGzipWriter(zw)
	return n, err
}

// WriteGzip writes gzipped p to w and returns the number of compressed
// bytes written to w.
func WriteGzip(w io.Writer, p []byte) (int, error) {
	return WriteGzipLevel(w, p, CompressDefaultCompression)
}

// AppendGzipBytes appends gzipped src to dst and returns the resulting dst.
func AppendGzipBytes(dst, src []byte) []byte {
	return AppendGzipBytesLevel(dst, src, CompressDefaultCompression)
}

// WriteGunzip writes ungzipped p to w and returns the number of uncompressed
// bytes written to w.
func WriteGunzip(w io.Writer, p []byte) (int, error) {
	r := &byteSliceReader{p}
	zr, err := acquireGzipReader(r)
	if err != nil {
		return 0, err
	}
	n, err := copyZeroAlloc(w, zr)
	releaseGzipReader(zr)
	nn := int(n)
	if int64(nn) != n {
		return 0, fmt.Errorf("too much data gunzipped: %d", n)
	}
	return nn, err
}

// WriteInflate writes inflated p to w and returns the number of uncompressed
// bytes written to w.
func WriteInflate(w io.Writer, p []byte) (int, error) {
	r := &byteSliceReader{p}
	zr, err := acquireFlateReader(r)
	if err != nil {
		return 0, err
	}
	n, err := copyZeroAlloc(w, zr)
	releaseFlateReader(zr)
	nn := int(n)
	if int64(nn) != n {
		return 0, fmt.Errorf("too much data inflated: %d", n)
	}
	return nn, err
}

// AppendGunzipBytes append gunzipped src to dst and returns the resulting dst.
func AppendGunzipBytes(dst, src []byte) ([]byte, error) {
	w := &byteSliceWriter{dst}
	_, err := WriteGunzip(w, src)
	return w.b, err
}

type byteSliceWriter struct {
	b []byte
}

func (w *byteSliceWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

type byteSliceReader struct {
	b []byte
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.b)
	r.b = r.b[n:]
	return n, nil
}

func acquireFlateWriter(w io.Writer, level int) *flateWriter {
	p := flateWriterPoolMap[level]
	if p == nil {
		panic(fmt.Sprintf("BUG: unexpected compression level passed: %d. See compress/flate for supported levels", level))
	}

	v := p.Get()
	if v == nil {
		zw, err := zlib.NewWriterLevel(w, level)
		if err != nil {
			panic(fmt.Sprintf("BUG: unexpected error in zlib.NewWriterLevel(%d): %s", level, err))
		}
		return &flateWriter{
			Writer: zw,
			p:      p,
		}
	}
	zw := v.(*flateWriter)
	zw.Reset(w)
	return zw
}

func releaseFlateWriter(zw *flateWriter) {
	zw.Close()
	zw.p.Put(zw)
}

type flateWriter struct {
	*zlib.Writer
	p *sync.Pool
}

var flateWriterPoolMap = func() map[int]*sync.Pool {
	// Initialize pools for all the compression levels defined
	// in https://golang.org/pkg/compress/flate/#pkg-constants .
	m := make(map[int]*sync.Pool, 11)
	m[-1] = &sync.Pool{}
	for i := 0; i < 10; i++ {
		m[i] = &sync.Pool{}
	}
	return m
}()

func isFileCompressible(f *os.File, minCompressRatio float64) bool {
	// Try compressing the first 4kb of of the file
	// and see if it can be compressed by more than
	// the given minCompressRatio.
	b := AcquireByteBuffer()
	zw := acquireGzipWriter(b, CompressDefaultCompression)
	lr := &io.LimitedReader{
		R: f,
		N: 4096,
	}
	_, err := copyZeroAlloc(zw, lr)
	releaseGzipWriter(zw)
	f.Seek(0, 0)
	if err != nil {
		return false
	}

	n := 4096 - lr.N
	zn := len(b.B)
	ReleaseByteBuffer(b)
	return float64(zn) < float64(n)*minCompressRatio
}
