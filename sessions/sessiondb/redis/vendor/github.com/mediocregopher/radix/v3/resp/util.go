package resp

import (
	"io"
)

// LenReader adds an additional method to io.Reader, returning how many bytes
// are left till be read until an io.EOF is reached.
type LenReader interface {
	io.Reader
	Len() int64
}

type lenReader struct {
	r io.Reader
	l int64
}

// NewLenReader wraps an existing io.Reader whose length is known so that it
// implements LenReader
func NewLenReader(r io.Reader, l int64) LenReader {
	return &lenReader{r: r, l: l}
}

func (lr *lenReader) Read(b []byte) (int, error) {
	n, err := lr.r.Read(b)
	lr.l -= int64(n)
	return n, err
}

func (lr *lenReader) Len() int64 {
	return lr.l
}
