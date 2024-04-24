package context

import (
	"math"
	"sync/atomic"
)

// Counter is the shared counter instances between Iris applications of the same process.
var Counter = NewGlobalCounter() // it's not used anywhere, currently but it's here.

// NewGlobalCounter returns a fresh instance of a global counter.
// End developers can use it as a helper for their applications.
func NewGlobalCounter() *GlobalCounter {
	return &GlobalCounter{Max: math.MaxUint64}
}

// GlobalCounter is a counter which
// atomically increments until Max.
type GlobalCounter struct {
	value uint64
	Max   uint64
}

// Increment increments the Value.
// The value cannot exceed the Max one.
// It uses Compare and Swap with the atomic package.
//
// Returns the new number value.
func (c *GlobalCounter) Increment() (newValue uint64) {
	for {
		prev := atomic.LoadUint64(&c.value)
		newValue = prev + 1

		if newValue >= c.Max {
			newValue = 0
		}

		if atomic.CompareAndSwapUint64(&c.value, prev, newValue) {
			break
		}
	}

	return
}

// Get returns the current counter without incrementing.
func (c *GlobalCounter) Get() uint64 {
	return atomic.LoadUint64(&c.value)
}
