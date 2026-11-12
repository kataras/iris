package encoding

import (
	"bytes"
	"encoding/gob"
	"io"
	"time"
)

// why?
// on the future we may change how these encoders/decoders
// and we may need different method for store and other for entry.

func init() {
	gob.Register(time.Time{})
	gob.Register(Entries[any]{})
	gob.Register(Entry[any]{})
	gob.Register(StringEntries{})
	gob.Register(StringEntry{})
}

// GobEncode accepts a store and writes
// as series of bytes to the "w" writer.
func GobEncode[T any](store Entries[T], w io.Writer) error {
	enc := gob.NewEncoder(w)
	err := enc.Encode(store)
	return err
}

// GobSerialize same as GobEncode but it returns
// the bytes using a temp buffer.
func GobSerialize[T any](store Entries[T]) ([]byte, error) {
	w := new(bytes.Buffer)
	err := GobEncode(store, w)
	return w.Bytes(), err
}

// GobEncodeEntry accepts an entry and writes
// as series of bytes to the "w" writer.
func GobEncodeEntry[T any](entry Entry[T], w io.Writer) error {
	enc := gob.NewEncoder(w)
	err := enc.Encode(entry)
	return err
}

// GobSerializeEntry same as GobEncodeEntry but it returns
// the bytes using a temp buffer.
func GobSerializeEntry[T any](entry Entry[T]) ([]byte, error) {
	w := new(bytes.Buffer)
	err := GobEncodeEntry(entry, w)
	return w.Bytes(), err
}

// GobDecode accepts a series of bytes and returns
// the store.
func GobDecode[T any](b []byte) (store Entries[T], err error) {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	// no reference because of:
	// gob: decoding into local type *memstore.Entries, received remote type Entry
	err = dec.Decode(&store)
	return
}

// GobDecodeEntry accepts a series of bytes and returns
// the entry.
func GobDecodeEntry[T any](b []byte) (entry Entry[T], err error) {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	err = dec.Decode(&entry)
	return
}
