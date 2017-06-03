// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hex implements hexadecimal encoding and decoding. It's taken almost
// verbatim from golang/encoding/hex, however it allows using a different set
// of encoding characters than the standard 0-F.
package hex

import (
	"errors"
	"fmt"
)

// DefaultEncoding behaves just like golang/encoding/hex.
var DefaultEncoding = NewEncoding("0123456789abcdef")

// An Encoding that uses a specific table of encoding characters.
type Encoding struct {
	hextable string
}

// NewEncoding constructs an Encoding using the given hextable.
func NewEncoding(hextable string) *Encoding {
	return &Encoding{hextable}
}

// EncodedLen returns the length of an encoding of n source bytes.
func EncodedLen(n int) int { return n * 2 }

// Encode encodes src into EncodedLen(len(src))
// bytes of dst.  As a convenience, it returns the number
// of bytes written to dst, but this value is always EncodedLen(len(src)).
// Encode implements hexadecimal encoding.
func (e *Encoding) Encode(dst, src []byte) int {
	for i, v := range src {
		dst[i*2] = e.hextable[v>>4]
		dst[i*2+1] = e.hextable[v&0x0f]
	}

	return len(src) * 2
}

// ErrLength results from decoding an odd length slice.
var ErrLength = errors.New("encoding/hex: odd length hex string")

// InvalidByteError values describe errors resulting from an invalid byte in a hex string.
type InvalidByteError byte

func (e InvalidByteError) Error() string {
	return fmt.Sprintf("encoding/hex: invalid byte: %#U", rune(e))
}

func DecodedLen(x int) int { return x / 2 }

// Decode decodes src into DecodedLen(len(src)) bytes, returning the actual
// number of bytes written to dst.
//
// If Decode encounters invalid input, it returns an error describing the failure.
func (e *Encoding) Decode(dst, src []byte) (int, error) {
	if len(src)%2 == 1 {
		return 0, ErrLength
	}

	for i := 0; i < len(src)/2; i++ {
		a, ok := e.fromHexChar(src[i*2])
		if !ok {
			return 0, InvalidByteError(src[i*2])
		}
		b, ok := e.fromHexChar(src[i*2+1])
		if !ok {
			return 0, InvalidByteError(src[i*2+1])
		}
		dst[i] = (a << 4) | b
	}

	return len(src) / 2, nil
}

// fromHexChar converts a hex character into its value and a success flag.
func (e *Encoding) fromHexChar(c byte) (byte, bool) {
	for i, ch := range []byte(e.hextable) {
		if ch == c {
			return byte(i), true
		}
	}

	return 0, false
}

// EncodeToString returns the hexadecimal encoding of src.
func (e *Encoding) EncodeToString(src []byte) string {
	dst := make([]byte, EncodedLen(len(src)))
	e.Encode(dst, src)
	return string(dst)
}

// DecodeString returns the bytes represented by the hexadecimal string s.
func (e *Encoding) DecodeString(s string) ([]byte, error) {
	src := []byte(s)
	dst := make([]byte, DecodedLen(len(src)))
	_, err := e.Decode(dst, src)
	if err != nil {
		return nil, err
	}
	return dst, nil
}
