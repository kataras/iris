package interpol

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type runeWriter interface {
	io.Writer
	WriteRune(r rune) (int, error)
}

func templateReader(opts *Options) io.RuneReader {
	if rr, ok := opts.Template.(io.RuneReader); ok {
		return rr
	}
	return bufio.NewReaderSize(opts.Template, utf8.UTFMax)
}

func outputWriter(opts *Options) runeWriter {
	if rw, ok := opts.Output.(runeWriter); ok {
		return rw
	}
	return &simpleRuneWriter{w: opts.Output}
}

type simpleRuneWriter struct {
	runeEncoder
	w io.Writer
}

func (rw *simpleRuneWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *simpleRuneWriter) WriteRune(r rune) (int, error) {
	return rw.w.Write(rw.encode(r))
}

type runeEncoder struct {
	b [utf8.UTFMax]byte
}

func (re *runeEncoder) encode(r rune) []byte {
	if r < utf8.RuneSelf {
		re.b[0] = byte(r)
		return re.b[:1]
	}
	n := utf8.EncodeRune(re.b[:], r)
	return re.b[:n]
}
