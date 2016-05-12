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

// BufferPool implements a pool of bytes.Buffers in the form of a bounded channel.
// Pulled from the github.com/oxtoacart/bpool package (Apache licensed).
type BufferPool struct {
	c chan *bytes.Buffer
}

// NewBufferPool creates a new BufferPool bounded to the given size.
func NewBufferPool(size int) (bp *BufferPool) {
	return &BufferPool{
		c: make(chan *bytes.Buffer, size),
	}
}

// Get gets a Buffer from the BufferPool, or creates a new one if none are
// available in the pool.
func (bp *BufferPool) Get() (b *bytes.Buffer) {
	select {
	case b = <-bp.c:
	// reuse existing buffer
	default:
		// create new buffer
		b = bytes.NewBuffer([]byte{})
	}
	return
}

// Put returns the given Buffer to the BufferPool.
func (bp *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()
	select {
	case bp.c <- b:
	default: // Discard the buffer if the pool is full.
	}
}
