package sessions

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
	"time"
)

func init() {
	gob.Register(time.Time{})
}

type (
	// Marshaler is the common marshaler interface, used by transcoder.
	Marshaler interface {
		Marshal(interface{}) ([]byte, error)
	}
	// Unmarshaler is the common unmarshaler interface, used by transcoder.
	Unmarshaler interface {
		Unmarshal([]byte, interface{}) error
	}
	// Transcoder is the interface that transcoders should implement, it includes just the `Marshaler` and the `Unmarshaler`.
	Transcoder interface {
		Marshaler
		Unmarshaler
	}
)

type (
	defaultTranscoder struct{}
	// GobTranscoder can be set to `DefaultTranscoder` to modify the database(s) transcoder.
	GobTranscoder struct{}
)

var (
	_ Transcoder = (*defaultTranscoder)(nil)
	_ Transcoder = (*GobTranscoder)(nil)

	// DefaultTranscoder is the default transcoder across databases (when `UseDatabase` is used).
	//
	// The default database's values encoder and decoder
	// calls the value's `Marshal/Unmarshal` methods (if any)
	// otherwise JSON is selected,
	// the JSON format can be stored to any database and
	// it supports both builtin language types(e.g. string, int) and custom struct values.
	// Also, and the most important, the values can be
	// retrieved/logged/monitored by a third-party program
	// written in any other language as well.
	//
	// You can change this behavior by registering a custom `Transcoder`.
	// Iris provides a `GobTranscoder` which is mostly suitable
	// if your session values are going to be custom Go structs.
	// Select this if you always retrieving values through Go.
	// Don't forget to initialize a call of gob.Register when necessary.
	// Read https://golang.org/pkg/encoding/gob/ for more.
	//
	// You can also implement your own `sessions.Transcoder` and use it,
	// i.e: a transcoder which will allow(on Marshal: return its byte representation and nil error)
	// or dissalow(on Marshal: return non nil error) certain types.
	//
	// sessions.DefaultTranscoder = sessions.GobTranscoder{}
	DefaultTranscoder Transcoder = defaultTranscoder{}
)

func (defaultTranscoder) Marshal(value interface{}) ([]byte, error) {
	if tr, ok := value.(Marshaler); ok {
		return tr.Marshal(value)
	}

	if jsonM, ok := value.(json.Marshaler); ok {
		return jsonM.MarshalJSON()
	}

	return json.Marshal(value)
}

func (defaultTranscoder) Unmarshal(b []byte, outPtr interface{}) error {
	if tr, ok := outPtr.(Unmarshaler); ok {
		return tr.Unmarshal(b, outPtr)
	}

	if jsonUM, ok := outPtr.(json.Unmarshaler); ok {
		return jsonUM.UnmarshalJSON(b)
	}

	return json.Unmarshal(b, outPtr)
}

// Marshal returns the gob encoding of "value".
func (GobTranscoder) Marshal(value interface{}) ([]byte, error) {
	var (
		w   = new(bytes.Buffer)
		enc = gob.NewEncoder(w)
		err error
	)

	if v, ok := value.(reflect.Value); ok {
		err = enc.EncodeValue(v)
	} else {
		err = enc.Encode(&value)
	}

	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// Unmarshal parses the gob-encoded data "b" and stores the result
// in the value pointed to by "outPtr".
func (GobTranscoder) Unmarshal(b []byte, outPtr interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))

	if v, ok := outPtr.(reflect.Value); ok {
		return dec.DecodeValue(v)
	}

	return dec.Decode(outPtr)
}
