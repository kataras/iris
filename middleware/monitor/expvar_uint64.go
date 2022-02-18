package monitor

import (
	"expvar"
	"strconv"
	"sync/atomic"
)

// Uint64 completes the expvar metric interface, holds an uint64 value.
type Uint64 struct {
	value uint64
}

// Set sets v to value.
func (v *Uint64) Set(value uint64) {
	atomic.StoreUint64(&v.value, value)
}

// Value returns the underline uint64 value.
func (v *Uint64) Value() uint64 {
	return atomic.LoadUint64(&v.value)
}

// String returns the text representation of the underline uint64 value.
func (v *Uint64) String() string {
	return strconv.FormatUint(atomic.LoadUint64(&v.value), 10)
}

func newUint64(name string) *Uint64 {
	v := new(Uint64)
	expvar.Publish(name, v)
	return v
}
