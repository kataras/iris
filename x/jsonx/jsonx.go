package jsonx

import (
	"bytes"
	"errors"
)

var (
	quoteLiteral    = '"'
	emptyQuoteBytes = []byte(`""`)
	nullLiteral     = []byte("null")

	// ErrInvalid is returned when the value is invalid.
	ErrInvalid = errors.New("invalid")
)

func isNull(b []byte) bool {
	return len(b) == 0 || bytes.Equal(b, nullLiteral)
}

func trimQuotesFunc(r rune) bool {
	return r == quoteLiteral
}

func trimQuotes(b []byte) []byte {
	return bytes.TrimFunc(b, trimQuotesFunc)
}
