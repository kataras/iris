// Package bytesutil provides utility functions for working with bytes and byte streams that are useful when
// working with the RESP protocol.
package bytesutil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// AnyIntToInt64 converts a value of any of Go's integer types (signed and unsigned) into a signed int64.
//
// If m is not of one of Go's built in integer types the call will panic.
func AnyIntToInt64(m interface{}) int64 {
	switch mt := m.(type) {
	case int:
		return int64(mt)
	case int8:
		return int64(mt)
	case int16:
		return int64(mt)
	case int32:
		return int64(mt)
	case int64:
		return mt
	case uint:
		return int64(mt)
	case uint8:
		return int64(mt)
	case uint16:
		return int64(mt)
	case uint32:
		return int64(mt)
	case uint64:
		return int64(mt)
	}
	panic(fmt.Sprintf("anyIntToInt64 got bad arg: %#v", m))
}

var bytePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 64)
		return &b
	},
}

// GetBytes returns a non-nil pointer to a byte slice from a pool of byte slices.
//
// The returned byte slice should be put back into the pool using PutBytes after usage.
func GetBytes() *[]byte {
	return bytePool.Get().(*[]byte)
}

// PutBytes puts the given byte slice pointer into a pool that can be accessed via GetBytes.
//
// After calling PutBytes the given pointer and byte slice must not be accessed anymore.
func PutBytes(b *[]byte) {
	*b = (*b)[:0]
	bytePool.Put(b)
}

// ParseInt is a specialized version of strconv.ParseInt that parses a base-10 encoded signed integer from a []byte.
//
// This can be used to avoid allocating a string, since strconv.ParseInt only takes a string.
func ParseInt(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, errors.New("empty slice given to parseInt")
	}

	var neg bool
	if b[0] == '-' || b[0] == '+' {
		neg = b[0] == '-'
		b = b[1:]
	}

	n, err := ParseUint(b)
	if err != nil {
		return 0, err
	}

	if neg {
		return -int64(n), nil
	}

	return int64(n), nil
}

// ParseUint is a specialized version of strconv.ParseUint that parses a base-10 encoded integer from a []byte.
//
// This can be used to avoid allocating a string, since strconv.ParseUint only takes a string.
func ParseUint(b []byte) (uint64, error) {
	if len(b) == 0 {
		return 0, errors.New("empty slice given to parseUint")
	}

	var n uint64

	for i, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid character %c at position %d in parseUint", c, i)
		}

		n *= 10
		n += uint64(c - '0')
	}

	return n, nil
}

// Expand expands the given byte slice to exactly n bytes.
//
// If cap(b) < n, a new slice will be allocated and filled with the bytes from b.
func Expand(b []byte, n int) []byte {
	if cap(b) < n {
		nb := make([]byte, n)
		copy(nb, b)
		return nb
	}
	return b[:n]
}

// BufferedBytesDelim reads a line from br and checks that the line ends with \r\n, returning the line without \r\n.
func BufferedBytesDelim(br *bufio.Reader) ([]byte, error) {
	b, err := br.ReadSlice('\n')
	if err != nil {
		return nil, err
	} else if len(b) < 2 || b[len(b)-2] != '\r' {
		return nil, fmt.Errorf("malformed resp %q", b)
	}
	return b[:len(b)-2], err
}

// BufferedIntDelim reads the current line from br as an integer.
func BufferedIntDelim(br *bufio.Reader) (int64, error) {
	b, err := BufferedBytesDelim(br)
	if err != nil {
		return 0, err
	}
	return ParseInt(b)
}

// ReadNAppend appends exactly n bytes from r into b.
func ReadNAppend(r io.Reader, b []byte, n int) ([]byte, error) {
	if n == 0 {
		return b, nil
	}
	m := len(b)
	b = Expand(b, len(b)+n)
	_, err := io.ReadFull(r, b[m:])
	return b, err
}

// ReadNDicard discards exactly n bytes from r.
func ReadNDiscard(r io.Reader, n int) error {
	type discarder interface {
		Discard(int) (int, error)
	}

	if n == 0 {
		return nil
	}

	switch v := r.(type) {
	case discarder:
		_, err := v.Discard(n)
		return err
	case io.Seeker:
		_, err := v.Seek(int64(n), io.SeekCurrent)
		return err
	}

	scratch := GetBytes()
	defer PutBytes(scratch)
	*scratch = (*scratch)[:cap(*scratch)]
	if len(*scratch) < n {
		*scratch = make([]byte, 8192)
	}

	for {
		buf := *scratch
		if len(buf) > n {
			buf = buf[:n]
		}
		nr, err := r.Read(buf)
		n -= nr
		if n == 0 || err != nil {
			return err
		}
	}
}

// ReadInt reads the next n bytes from r as a signed 64 bit integer.
func ReadInt(r io.Reader, n int) (int64, error) {
	scratch := GetBytes()
	defer PutBytes(scratch)

	var err error
	if *scratch, err = ReadNAppend(r, *scratch, n); err != nil {
		return 0, err
	}
	return ParseInt(*scratch)
}

// ReadUint reads the next n bytes from r as an unsigned 64 bit integer.
func ReadUint(r io.Reader, n int) (uint64, error) {
	scratch := GetBytes()
	defer PutBytes(scratch)

	var err error
	if *scratch, err = ReadNAppend(r, *scratch, n); err != nil {
		return 0, err
	}
	return ParseUint(*scratch)
}

// ReadFloat reads the next n bytes from r as a 64 bit floating point number with the given precision.
func ReadFloat(r io.Reader, precision, n int) (float64, error) {
	scratch := GetBytes()
	defer PutBytes(scratch)

	var err error
	if *scratch, err = ReadNAppend(r, *scratch, n); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(string(*scratch), precision)
}
