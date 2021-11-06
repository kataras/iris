package jsonx

import "bytes"

var (
	quoteLiteral    = '"'
	emptyQuoteBytes = []byte(`""`)
	nullLiteral     = []byte("null")
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
