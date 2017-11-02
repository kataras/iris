package encoding

import (
	"errors"
	"io"
	"io/ioutil"
	"reflect"
)

// UnmarshalerFunc is the Unmarshaler compatible type.
//
// See 'unmarshalBody' for more.
type UnmarshalerFunc func(data []byte, v interface{}) error

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type.
func UnmarshalBody(body io.Reader, v interface{}, unmarshaler UnmarshalerFunc) error {
	if body == nil {
		return errors.New("unmarshal: empty body")
	}

	rawData, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	// check if v is already a pointer, if yes then pass as it's
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		return unmarshaler(rawData, v)
	}
	// finally, if the v doesn't contains a self-body decoder and it's not a pointer
	// use the custom unmarshaler to bind the body
	return unmarshaler(rawData, &v)
}
