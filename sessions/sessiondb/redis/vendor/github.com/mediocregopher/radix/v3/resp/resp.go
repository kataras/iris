// Package resp is an umbrella package which covers both the old RESP protocol
// (resp2) and the new one (resp3), allowing clients to choose which one they
// care to use
package resp

import (
	"bufio"
	"io"
)

// Marshaler is the interface implemented by types that can marshal themselves
// into valid RESP.
type Marshaler interface {
	MarshalRESP(io.Writer) error
}

// Unmarshaler is the interface implemented by types that can unmarshal a RESP
// description of themselves.
//
// Note that, unlike Marshaler, Unmarshaler _must_ take in a *bufio.Reader.
type Unmarshaler interface {
	UnmarshalRESP(*bufio.Reader) error
}
