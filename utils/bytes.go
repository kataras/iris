package utils

import (
	"bytes"
	"encoding/gob"
)

// SerializeBytes serializa bytes using gob encoder and returns them
func SerializeBytes(m interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(m)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

// DeserializeBytes converts the bytes to an object using gob decoder
func DeserializeBytes(b []byte, m interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	return dec.Decode(m) //no reference here otherwise doesn't work because of go remote object
}
