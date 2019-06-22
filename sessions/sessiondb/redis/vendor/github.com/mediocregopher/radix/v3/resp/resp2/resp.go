// Package resp2 implements the original redis RESP protocol, a plaintext
// protocol which is also binary safe. Redis uses the RESP protocol to
// communicate with its clients, but there's nothing about the protocol which
// ties it to redis, it could be used for almost anything.
//
// See https://redis.io/topics/protocol for more details on the protocol.
package resp2

import (
	"bufio"
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"

	"github.com/mediocregopher/radix/v3/internal/bytesutil"

	"github.com/mediocregopher/radix/v3/resp"
)

var delim = []byte{'\r', '\n'}

// prefix enumerates the possible RESP types by enumerating the different
// prefixes a RESP message might start with.
type prefix []byte

// Enumeration of each of RESP's message types, each denoted by the prefix which
// is prepended to messages of that type.
//
// In order to determine the type of a message which is being written to a
// *bufio.Reader, without actually consuming it, one can use the Peek method and
// compare it against these values.
var (
	SimpleStringPrefix = []byte{'+'}
	ErrorPrefix        = []byte{'-'}
	IntPrefix          = []byte{':'}
	BulkStringPrefix   = []byte{'$'}
	ArrayPrefix        = []byte{'*'}
)

// String formats a prefix into a human-readable name for the type it denotes.
func (p prefix) String() string {
	pStr := string(p)
	switch pStr {
	case string(SimpleStringPrefix):
		return "simple-string"
	case string(ErrorPrefix):
		return "error"
	case string(IntPrefix):
		return "integer"
	case string(BulkStringPrefix):
		return "bulk-string"
	case string(ArrayPrefix):
		return "array"
	default:
		return pStr
	}
}

var (
	nilBulkString = []byte("$-1\r\n")
	nilArray      = []byte("*-1\r\n")
)

var bools = [][]byte{
	{'0'},
	{'1'},
}

////////////////////////////////////////////////////////////////////////////////

func assertBufferedPrefix(br *bufio.Reader, pref prefix) error {
	b, err := br.Peek(len(pref))
	if err != nil {
		return err
	} else if !bytes.Equal(b, []byte(pref)) {
		return fmt.Errorf("expected prefix %q, got %q", pref.String(), prefix(b).String())
	}
	_, err = br.Discard(len(pref))
	return err
}

////////////////////////////////////////////////////////////////////////////////

// SimpleString represents the simple string type in the RESP protocol
type SimpleString struct {
	S string
}

// MarshalRESP implements the Marshaler method
func (ss SimpleString) MarshalRESP(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, SimpleStringPrefix...)
	*scratch = append(*scratch, ss.S...)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (ss *SimpleString) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, SimpleStringPrefix); err != nil {
		return err
	}
	b, err := bytesutil.BufferedBytesDelim(br)
	if err != nil {
		return err
	}

	ss.S = string(b)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// Error represents an error type in the RESP protocol. Note that this only
// represents an actual error message being read/written on the stream, it is
// separate from network or parsing errors. An E value of nil is equivalent to
// an empty error string
type Error struct {
	E error
}

func (e Error) Error() string {
	return e.E.Error()
}

// MarshalRESP implements the Marshaler method
func (e Error) MarshalRESP(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, ErrorPrefix...)
	if e.E != nil {
		*scratch = append(*scratch, e.E.Error()...)
	}
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (e *Error) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, ErrorPrefix); err != nil {
		return err
	}
	b, err := bytesutil.BufferedBytesDelim(br)
	e.E = errors.New(string(b))
	return err
}

////////////////////////////////////////////////////////////////////////////////

// Int represents an int type in the RESP protocol
type Int struct {
	I int64
}

// MarshalRESP implements the Marshaler method
func (i Int) MarshalRESP(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, IntPrefix...)
	*scratch = strconv.AppendInt(*scratch, i.I, 10)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (i *Int) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, IntPrefix); err != nil {
		return err
	}
	n, err := bytesutil.BufferedIntDelim(br)
	i.I = n
	return err
}

////////////////////////////////////////////////////////////////////////////////

// BulkStringBytes represents the bulk string type in the RESP protocol using a
// go byte slice. A B value of nil indicates the nil bulk string message, versus
// a B value of []byte{} which indicates a bulk string of length 0.
type BulkStringBytes struct {
	B []byte

	// If true then this won't marshal the nil RESP value when B is nil, it will
	// marshal as an empty string instead
	MarshalNotNil bool
}

// MarshalRESP implements the Marshaler method
func (b BulkStringBytes) MarshalRESP(w io.Writer) error {
	if b.B == nil && !b.MarshalNotNil {
		_, err := w.Write(nilBulkString)
		return err
	}
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, BulkStringPrefix...)
	*scratch = strconv.AppendInt(*scratch, int64(len(b.B)), 10)
	*scratch = append(*scratch, delim...)
	*scratch = append(*scratch, b.B...)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (b *BulkStringBytes) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, BulkStringPrefix); err != nil {
		return err
	}
	n, err := bytesutil.BufferedIntDelim(br)
	nn := int(n)
	if err != nil {
		return err
	} else if n == -1 {
		b.B = nil
		return nil
	} else {
		b.B = bytesutil.Expand(b.B, nn)
		if b.B == nil {
			b.B = []byte{}
		}
	}

	if _, err := io.ReadFull(br, b.B); err != nil {
		return err
	} else if _, err := bytesutil.BufferedBytesDelim(br); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// BulkString represents the bulk string type in the RESP protocol using a go
// string.
type BulkString struct {
	S string
}

// MarshalRESP implements the Marshaler method
func (b BulkString) MarshalRESP(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, BulkStringPrefix...)
	*scratch = strconv.AppendInt(*scratch, int64(len(b.S)), 10)
	*scratch = append(*scratch, delim...)
	*scratch = append(*scratch, b.S...)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method. This treats a Nil bulk
// string message as empty string.
func (b *BulkString) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, BulkStringPrefix); err != nil {
		return err
	}
	n, err := bytesutil.BufferedIntDelim(br)
	if err != nil {
		return err
	} else if n == -1 {
		b.S = ""
		return nil
	}

	scratch := bytesutil.GetBytes()
	defer bytesutil.PutBytes(scratch)
	*scratch = bytesutil.Expand(*scratch, int(n))

	if _, err := io.ReadFull(br, *scratch); err != nil {
		return err
	} else if _, err := bytesutil.BufferedBytesDelim(br); err != nil {
		return err
	}

	b.S = string(*scratch)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// BulkReader is like BulkString, but it only supports marshalling and will use
// the given LenReader to do so. If LR is nil then the nil bulk string RESP will
// be written
type BulkReader struct {
	LR resp.LenReader
}

// MarshalRESP implements the Marshaler method
func (b BulkReader) MarshalRESP(w io.Writer) error {
	if b.LR == nil {
		_, err := w.Write(nilBulkString)
		return err
	}

	l := b.LR.Len()
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, BulkStringPrefix...)
	*scratch = strconv.AppendInt(*scratch, l, 10)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	if err != nil {
		return err
	}

	if _, err := io.CopyN(w, b.LR, l); err != nil {
		return err
	} else if _, err := w.Write(delim); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// ArrayHeader represents the header sent preceding array elements in the RESP
// protocol. It does not actually encompass any elements itself, it only
// declares how many elements will come after it.
//
// An N of -1 may also be used to indicate a nil response, as per the RESP spec
type ArrayHeader struct {
	N int
}

// MarshalRESP implements the Marshaler method
func (ah ArrayHeader) MarshalRESP(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, ArrayPrefix...)
	*scratch = strconv.AppendInt(*scratch, int64(ah.N), 10)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (ah *ArrayHeader) UnmarshalRESP(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, ArrayPrefix); err != nil {
		return err
	}
	n, err := bytesutil.BufferedIntDelim(br)
	ah.N = int(n)
	return err
}

////////////////////////////////////////////////////////////////////////////////

// Array represents an array of RESP elements which will be marshaled as a RESP
// array. If A is Nil then a Nil RESP will be marshaled.
type Array struct {
	A []resp.Marshaler
}

// MarshalRESP implements the Marshaler method
func (a Array) MarshalRESP(w io.Writer) error {
	ah := ArrayHeader{N: len(a.A)}
	if a.A == nil {
		ah.N = -1
	}

	if err := ah.MarshalRESP(w); err != nil {
		return err
	}
	for _, el := range a.A {
		if err := el.MarshalRESP(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// Any represents any primitive go type, such as integers, floats, strings,
// bools, etc... It also includes encoding.Text(Un)Marshalers and
// encoding.(Un)BinaryMarshalers. It will _not_ marshal resp.Marshalers.
//
// Most things will be treated as bulk strings, except for those that have their
// own corresponding type in the RESP protocol (e.g. ints). strings and []bytes
// will always be encoded as bulk strings, never simple strings.
//
// Arrays and slices will be treated as RESP arrays, and their values will be
// treated as if also wrapped in an Any struct. Maps will be similarly treated,
// but they will be flattened into arrays of their alternating keys/values
// first.
//
// When using UnmarshalRESP the value of I must be a pointer or nil. If it is
// nil then the RESP value will be read and discarded.
//
// If an error type is read in the UnmarshalRESP method then a resp2.Error will
// be returned with that error, and the value of I won't be touched.
type Any struct {
	I interface{}

	// If true then the MarshalRESP method will marshal all non-array types as
	// bulk strings. This primarily effects integers and errors.
	MarshalBulkString bool

	// If true then no array headers will be sent when MarshalRESP is called.
	// For I values which are non-arrays this means no behavior change. For
	// arrays and embedded arrays it means only the array elements will be
	// written, and an ArrayHeader must have been manually marshalled
	// beforehand.
	MarshalNoArrayHeaders bool
}

func (a Any) cp(i interface{}) Any {
	a.I = i
	return a
}

var byteSliceT = reflect.TypeOf([]byte{})

// NumElems returns the number of non-array elements which would be marshalled
// based on I. For example:
//
//	Any{I: "foo"}.NumElems() == 1
//	Any{I: []string{}}.NumElems() == 0
//	Any{I: []string{"foo"}}.NumElems() == 1
//	Any{I: []string{"foo", "bar"}}.NumElems() == 2
//	Any{I: [][]string{{"foo"}, {"bar", "baz"}, {}}}.NumElems() == 3
//
func (a Any) NumElems() int {
	return numElems(reflect.ValueOf(a.I))
}

var (
	lenReaderT               = reflect.TypeOf(new(resp.LenReader)).Elem()
	encodingTextMarshalerT   = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
	encodingBinaryMarshalerT = reflect.TypeOf(new(encoding.BinaryMarshaler)).Elem()
)

func numElems(vv reflect.Value) int {
	if !vv.IsValid() {
		return 1
	}

	tt := vv.Type()
	switch {
	case tt.Implements(lenReaderT):
		return 1
	case tt.Implements(encodingTextMarshalerT):
		return 1
	case tt.Implements(encodingBinaryMarshalerT):
		return 1
	}

	switch vv.Kind() {
	case reflect.Ptr:
		return numElems(reflect.Indirect(vv))
	case reflect.Slice, reflect.Array:
		// TODO does []rune need extra support here?
		if vv.Type() == byteSliceT {
			return 1
		}

		l := vv.Len()
		var c int
		for i := 0; i < l; i++ {
			c += numElems(vv.Index(i))
		}
		return c

	case reflect.Map:
		kkv := vv.MapKeys()
		var c int
		for _, kv := range kkv {
			c += numElems(kv)
			c += numElems(vv.MapIndex(kv))
		}
		return c

	case reflect.Interface:
		return numElems(vv.Elem())

	case reflect.Struct:
		return numElemsStruct(vv, true)

	default:
		return 1
	}
}

// this is separated out of numElems because marshalStruct is only given the
// reflect.Value and needs to know the numElems, so it wouldn't make sense to
// recast to an interface{} to pass into NumElems, it would just get turned into
// a reflect.Value again.
func numElemsStruct(vv reflect.Value, flat bool) int {
	tt := vv.Type()
	l := vv.NumField()
	var c int
	for i := 0; i < l; i++ {
		ft, fv := tt.Field(i), vv.Field(i)
		if ft.Anonymous {
			if fv = reflect.Indirect(fv); fv.IsValid() { // fv isn't nil
				c += numElemsStruct(fv, flat)
			}
			continue
		} else if ft.PkgPath != "" || ft.Tag.Get("redis") == "-" {
			continue // continue
		}

		c++ // for the key
		if flat {
			c += numElems(fv)
		} else {
			c++
		}
	}
	return c
}

// MarshalRESP implements the Marshaler method
func (a Any) MarshalRESP(w io.Writer) error {
	marshalBulk := func(b []byte) error {
		bs := BulkStringBytes{B: b, MarshalNotNil: a.MarshalBulkString}
		return bs.MarshalRESP(w)
	}

	switch at := a.I.(type) {
	case []byte:
		return marshalBulk(at)
	case string:
		if at == "" {
			// special case, we never want string to be nil, but appending empty
			// string to a nil []byte would still be a nil bulk string
			return BulkStringBytes{MarshalNotNil: true}.MarshalRESP(w)
		}
		scratch := bytesutil.GetBytes()
		defer bytesutil.PutBytes(scratch)
		*scratch = append(*scratch, at...)
		return marshalBulk(*scratch)
	case bool:
		b := bools[0]
		if at {
			b = bools[1]
		}
		return marshalBulk(b)
	case float32:
		scratch := bytesutil.GetBytes()
		defer bytesutil.PutBytes(scratch)
		*scratch = strconv.AppendFloat(*scratch, float64(at), 'f', -1, 32)
		return marshalBulk(*scratch)
	case float64:
		scratch := bytesutil.GetBytes()
		defer bytesutil.PutBytes(scratch)
		*scratch = strconv.AppendFloat(*scratch, at, 'f', -1, 64)
		return marshalBulk(*scratch)
	case nil:
		return marshalBulk(nil)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		at64 := bytesutil.AnyIntToInt64(at)
		if a.MarshalBulkString {
			scratch := bytesutil.GetBytes()
			defer bytesutil.PutBytes(scratch)
			*scratch = strconv.AppendInt(*scratch, at64, 10)
			return marshalBulk(*scratch)
		}
		return Int{I: at64}.MarshalRESP(w)
	case error:
		if a.MarshalBulkString {
			scratch := bytesutil.GetBytes()
			defer bytesutil.PutBytes(scratch)
			*scratch = append(*scratch, at.Error()...)
			return marshalBulk(*scratch)
		}
		return Error{E: at}.MarshalRESP(w)
	case resp.LenReader:
		return BulkReader{LR: at}.MarshalRESP(w)
	case encoding.TextMarshaler:
		b, err := at.MarshalText()
		if err != nil {
			return err
		}
		return marshalBulk(b)
	case encoding.BinaryMarshaler:
		b, err := at.MarshalBinary()
		if err != nil {
			return err
		}
		return marshalBulk(b)
	}

	// now we use.... reflection! duhduhduuuuh....
	vv := reflect.ValueOf(a.I)

	// if it's a pointer we de-reference and try the pointed to value directly
	if vv.Kind() == reflect.Ptr {
		return a.cp(reflect.Indirect(vv).Interface()).MarshalRESP(w)
	}

	// some helper functions
	var err error
	arrHeader := func(l int) {
		if a.MarshalNoArrayHeaders || err != nil {
			return
		}
		err = ArrayHeader{N: l}.MarshalRESP(w)
	}
	arrVal := func(v interface{}) {
		if err != nil {
			return
		}
		err = a.cp(v).MarshalRESP(w)
	}

	switch vv.Kind() {
	case reflect.Slice, reflect.Array:
		if vv.IsNil() && !a.MarshalNoArrayHeaders {
			_, err := w.Write(nilArray)
			return err
		}
		l := vv.Len()
		arrHeader(l)
		for i := 0; i < l; i++ {
			arrVal(vv.Index(i).Interface())
		}

	case reflect.Map:
		if vv.IsNil() && !a.MarshalNoArrayHeaders {
			_, err := w.Write(nilArray)
			return err
		}
		kkv := vv.MapKeys()
		arrHeader(len(kkv) * 2)
		for _, kv := range kkv {
			arrVal(kv.Interface())
			arrVal(vv.MapIndex(kv).Interface())
		}

	case reflect.Struct:
		return a.marshalStruct(w, vv, false)

	default:
		return fmt.Errorf("could not marshal value of type %T", a.I)
	}

	return err
}

func (a Any) marshalStruct(w io.Writer, vv reflect.Value, inline bool) error {
	var err error
	if !a.MarshalNoArrayHeaders && !inline {
		numElems := numElemsStruct(vv, a.MarshalNoArrayHeaders)
		if err = (ArrayHeader{N: numElems}).MarshalRESP(w); err != nil {
			return err
		}
	}

	tt := vv.Type()
	l := vv.NumField()
	for i := 0; i < l; i++ {
		ft, fv := tt.Field(i), vv.Field(i)
		tag := ft.Tag.Get("redis")
		if ft.Anonymous {
			if fv = reflect.Indirect(fv); !fv.IsValid() { // fv is nil
				continue
			} else if err := a.marshalStruct(w, fv, true); err != nil {
				return err
			}
			continue
		} else if ft.PkgPath != "" || tag == "-" {
			continue // unexported
		}

		keyName := ft.Name
		if tag != "" {
			keyName = tag
		}
		if err := (BulkString{S: keyName}).MarshalRESP(w); err != nil {
			return err
		} else if err := a.cp(fv.Interface()).MarshalRESP(w); err != nil {
			return err
		}
	}
	return nil
}

func saneDefault(prefix byte) interface{} {
	// we don't handle ErrorPrefix because that always returns an error and
	// doesn't touch I
	switch prefix {
	case ArrayPrefix[0]:
		ii := make([]interface{}, 8)
		return &ii
	case BulkStringPrefix[0]:
		bb := make([]byte, 16)
		return &bb
	case SimpleStringPrefix[0]:
		return new(string)
	case IntPrefix[0]:
		return new(int64)
	}
	panic("should never get here")
}

// We use pools for these even though they only get used within
// Any.UnmarshalRESP because of how often they get used. Any return from redis
// which has a simple string or bulk string (the vast majority of them) is going
// to go through one of these.
var (
	// RawMessage.UnmarshalInto also uses these
	byteReaderPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewReader(nil)
		},
	}
	bufioReaderPool = sync.Pool{
		New: func() interface{} {
			return bufio.NewReader(nil)
		},
	}
)

// UnmarshalRESP implements the Unmarshaler method
func (a Any) UnmarshalRESP(br *bufio.Reader) error {
	// if I is itself an Unmarshaler just hit that directly
	if u, ok := a.I.(resp.Unmarshaler); ok {
		return u.UnmarshalRESP(br)
	}

	b, err := br.Peek(1)
	if err != nil {
		return err
	}
	prefix := b[0]

	// This is a super special case that _must_ be handled before we actually
	// read from the reader. If an *interface{} is given we instead unmarshal
	// into a default (created based on the type of th message), then set the
	// *interface{} to that
	if ai, ok := a.I.(*interface{}); ok {
		innerA := Any{I: saneDefault(prefix)}
		if err := innerA.UnmarshalRESP(br); err != nil {
			return err
		}
		*ai = reflect.ValueOf(innerA.I).Elem().Interface()
		return nil
	}

	br.Discard(1)
	b, err = bytesutil.BufferedBytesDelim(br)
	if err != nil {
		return err
	}

	switch prefix {
	case ErrorPrefix[0]:
		return Error{E: errors.New(string(b))}
	case ArrayPrefix[0]:
		l, err := bytesutil.ParseInt(b)
		if err != nil {
			return err
		} else if l == -1 {
			return a.unmarshalNil()
		}
		return a.unmarshalArray(br, l)
	case BulkStringPrefix[0]:
		l, err := bytesutil.ParseInt(b) // fuck DRY
		if err != nil {
			return err
		} else if l == -1 {
			return a.unmarshalNil()
		}
		if err := a.unmarshalSingle(br, int(l)); err != nil {
			return err
		}
		_, err = br.Discard(2)
		return err
	case SimpleStringPrefix[0], IntPrefix[0]:
		reader := byteReaderPool.Get().(*bytes.Reader)
		reader.Reset(b)
		err := a.unmarshalSingle(reader, reader.Len())
		byteReaderPool.Put(reader)
		return err
	default:
		return fmt.Errorf("unknown type prefix %q", b[0])
	}
}

func (a Any) unmarshalSingle(body io.Reader, n int) error {
	var (
		err error
		i   int64
		ui  uint64
	)

	switch ai := a.I.(type) {
	case nil:
		// just read it and do nothing
		err = bytesutil.ReadNDiscard(body, n)
	case *string:
		scratch := bytesutil.GetBytes()
		*scratch, err = bytesutil.ReadNAppend(body, *scratch, n)
		*ai = string(*scratch)
		bytesutil.PutBytes(scratch)
	case *[]byte:
		*ai, err = bytesutil.ReadNAppend(body, (*ai)[:0], n)
	case *bool:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = ui > 0
	case *int:
		i, err = bytesutil.ReadInt(body, n)
		*ai = int(i)
	case *int8:
		i, err = bytesutil.ReadInt(body, n)
		*ai = int8(i)
	case *int16:
		i, err = bytesutil.ReadInt(body, n)
		*ai = int16(i)
	case *int32:
		i, err = bytesutil.ReadInt(body, n)
		*ai = int32(i)
	case *int64:
		i, err = bytesutil.ReadInt(body, n)
		*ai = i
	case *uint:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = uint(ui)
	case *uint8:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = uint8(ui)
	case *uint16:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = uint16(ui)
	case *uint32:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = uint32(ui)
	case *uint64:
		ui, err = bytesutil.ReadUint(body, n)
		*ai = ui
	case *float32:
		var f float64
		f, err = bytesutil.ReadFloat(body, 32, n)
		*ai = float32(f)
	case *float64:
		*ai, err = bytesutil.ReadFloat(body, 64, n)
	case io.Writer:
		_, err = io.CopyN(ai, body, int64(n))
	case encoding.TextUnmarshaler:
		scratch := bytesutil.GetBytes()
		if *scratch, err = bytesutil.ReadNAppend(body, *scratch, n); err != nil {
			break
		}
		err = ai.UnmarshalText(*scratch)
		bytesutil.PutBytes(scratch)
	case encoding.BinaryUnmarshaler:
		scratch := bytesutil.GetBytes()
		if *scratch, err = bytesutil.ReadNAppend(body, *scratch, n); err != nil {
			break
		}
		err = ai.UnmarshalBinary(*scratch)
		bytesutil.PutBytes(scratch)
	default:
		return fmt.Errorf("can't unmarshal into %T", a.I)
	}

	return err
}

func (a Any) unmarshalNil() error {
	vv := reflect.ValueOf(a.I)
	if vv.Kind() != reflect.Ptr || !vv.Elem().CanSet() {
		// If the type in I can't be set then just ignore it. This is kind of
		// weird but it's what encoding/json does in the same circumstance
		return nil
	}

	vve := vv.Elem()
	vve.Set(reflect.Zero(vve.Type()))
	return nil
}

func (a Any) unmarshalArray(br *bufio.Reader, l int64) error {
	if a.I == nil {
		return a.discardArray(br, l)
	}

	size := int(l)
	v := reflect.ValueOf(a.I)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("can't unmarshal into %T", a.I)
	}
	v = reflect.Indirect(v)

	switch v.Kind() {
	case reflect.Slice:
		if size > v.Cap() || v.IsNil() {
			newV := reflect.MakeSlice(v.Type(), size, size)
			// we copy only because there might be some preset values in there
			// already that we're intended to decode into,
			// e.g.  []interface{}{int8(0), ""}
			reflect.Copy(newV, v)
			v.Set(newV)
		} else if size != v.Len() {
			v.SetLen(size)
		}

		for i := 0; i < size; i++ {
			ai := Any{I: v.Index(i).Addr().Interface()}
			if err := ai.UnmarshalRESP(br); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		if size%2 != 0 {
			return errors.New("cannot decode redis array with odd number of elements into map")
		} else if v.IsNil() {
			v.Set(reflect.MakeMapWithSize(v.Type(), size/2))
		}

		var kvs reflect.Value
		if size > 0 && canShareReflectValue(v.Type().Key()) {
			kvs = reflect.New(v.Type().Key())
		}

		var vvs reflect.Value
		if size > 0 && canShareReflectValue(v.Type().Elem()) {
			vvs = reflect.New(v.Type().Elem())
		}

		for i := 0; i < size; i += 2 {
			kv := kvs
			if !kv.IsValid() {
				kv = reflect.New(v.Type().Key())
			}
			if err := (Any{I: kv.Interface()}).UnmarshalRESP(br); err != nil {
				return err
			}

			vv := vvs
			if !vv.IsValid() {
				vv = reflect.New(v.Type().Elem())
			}
			if err := (Any{I: vv.Interface()}).UnmarshalRESP(br); err != nil {
				return err
			}

			v.SetMapIndex(kv.Elem(), vv.Elem())
		}
		return nil

	case reflect.Struct:
		if size%2 != 0 {
			return errors.New("cannot decode redis array with odd number of elements into struct")
		}

		structFields := getStructFields(v.Type())
		var field BulkStringBytes

		for i := 0; i < size; i += 2 {
			if err := field.UnmarshalRESP(br); err != nil {
				return err
			}

			var vv reflect.Value
			structField, ok := structFields[string(field.B)] // no allocation, since Go 1.3
			if ok {
				vv = getStructField(v, structField.indices)
			}

			if !ok || !vv.IsValid() {
				// discard the value
				if err := (Any{}).UnmarshalRESP(br); err != nil {
					return err
				}
				continue
			}

			if err := (Any{I: vv.Interface()}).UnmarshalRESP(br); err != nil {
				return err
			}
		}

		return nil

	default:
		return fmt.Errorf("cannot decode redis array into %v", v.Type())
	}
}

func canShareReflectValue(ty reflect.Type) bool {
	switch ty.Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

type structField struct {
	name    string
	fromTag bool // from a tag overwrites a field name
	indices []int
}

// encoding/json uses a similar pattern for unmarshaling into structs
var structFieldsCache sync.Map // aka map[reflect.Type]map[string]structField

func getStructFields(t reflect.Type) map[string]structField {
	if mV, ok := structFieldsCache.Load(t); ok {
		return mV.(map[string]structField)
	}

	getIndices := func(parents []int, i int) []int {
		indices := make([]int, len(parents), len(parents)+1)
		copy(indices, parents)
		indices = append(indices, i)
		return indices
	}

	m := map[string]structField{}

	var populateFrom func(reflect.Type, []int)
	populateFrom = func(t reflect.Type, parents []int) {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		l := t.NumField()

		// first get all fields which aren't embedded structs
		for i := 0; i < l; i++ {
			ft := t.Field(i)
			if ft.Anonymous || ft.PkgPath != "" {
				continue
			}

			key, fromTag := ft.Name, false
			if tag := ft.Tag.Get("redis"); tag != "" && tag != "-" {
				key, fromTag = tag, true
			}
			if m[key].fromTag {
				continue
			}
			m[key] = structField{
				name:    key,
				fromTag: fromTag,
				indices: getIndices(parents, i),
			}
		}

		// then find all embedded structs and descend into them
		for i := 0; i < l; i++ {
			ft := t.Field(i)
			if !ft.Anonymous {
				continue
			}
			populateFrom(ft.Type, getIndices(parents, i))
		}
	}

	populateFrom(t, []int{})
	structFieldsCache.LoadOrStore(t, m)
	return m
}

// v must be setable. Always returns a Kind() == reflect.Ptr, unless it returns
// the zero Value, which means a setable value couldn't be gotten.
func getStructField(v reflect.Value, ii []int) reflect.Value {
	if len(ii) == 0 {
		return v.Addr()
	}
	i, ii := ii[0], ii[1:]

	iv := v.Field(i)
	if iv.Kind() == reflect.Ptr && iv.IsNil() {
		// If the field is a pointer to an unexported type then it won't be
		// settable, though if the user pre-sets the value it will be (I think).
		if !iv.CanSet() {
			return reflect.Value{}
		}
		iv.Set(reflect.New(iv.Type().Elem()))
	}
	iv = reflect.Indirect(iv)

	return getStructField(iv, ii)
}

func (a Any) discardArray(br *bufio.Reader, l int64) error {
	for i := 0; i < int(l); i++ {
		if err := (Any{}).UnmarshalRESP(br); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// RawMessage is a Marshaler/Unmarshaler which will capture the exact raw bytes
// of a RESP message. When Marshaling the exact bytes of the RawMessage will be
// written as-is. When Unmarshaling the bytes of a single RESP message will be
// read into the RawMessage's bytes.
type RawMessage []byte

// MarshalRESP implements the Marshaler method
func (rm RawMessage) MarshalRESP(w io.Writer) error {
	_, err := w.Write(rm)
	return err
}

// UnmarshalRESP implements the Unmarshaler method
func (rm *RawMessage) UnmarshalRESP(br *bufio.Reader) error {
	*rm = (*rm)[:0]
	return rm.unmarshal(br)
}

func (rm *RawMessage) unmarshal(br *bufio.Reader) error {
	b, err := br.ReadSlice('\n')
	if err != nil {
		return err
	}
	*rm = append(*rm, b...)

	if len(b) < 3 {
		return errors.New("malformed data read")
	}
	body := b[1 : len(b)-2]

	switch b[0] {
	case ArrayPrefix[0]:
		l, err := bytesutil.ParseInt(body)
		if err != nil {
			return err
		} else if l == -1 {
			return nil
		}
		for i := 0; i < int(l); i++ {
			if err := rm.unmarshal(br); err != nil {
				return err
			}
		}
		return nil
	case BulkStringPrefix[0]:
		l, err := bytesutil.ParseInt(body) // fuck DRY
		if err != nil {
			return err
		} else if l == -1 {
			return nil
		}
		*rm, err = bytesutil.ReadNAppend(br, *rm, int(l+2))
		return err
	case ErrorPrefix[0], SimpleStringPrefix[0], IntPrefix[0]:
		return nil
	default:
		return fmt.Errorf("unknown type prefix %q", b[0])
	}
}

// UnmarshalInto is a shortcut for wrapping this RawMessage in a *bufio.Reader
// and passing that into the given Unmarshaler's UnmarshalRESP method. Any error
// from calling UnmarshalRESP is returned, and the RawMessage is unaffected in
// all cases.
func (rm RawMessage) UnmarshalInto(u resp.Unmarshaler) error {
	r := byteReaderPool.Get().(*bytes.Reader)
	r.Reset(rm)
	br := bufioReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	err := u.UnmarshalRESP(br)
	bufioReaderPool.Put(br)
	byteReaderPool.Put(r)
	return err
}

// IsNil returns true if the contents of RawMessage are one of the nil values.
func (rm RawMessage) IsNil() bool {
	return bytes.Equal(rm, nilBulkString) || bytes.Equal(rm, nilArray)
}
