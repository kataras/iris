// Package counter counts requests (or anything else) with an atomic,
// generic-typed counter.
//
// Its entry point is NewCounter[T](), which returns a *Counter[T] rather than a
// Handler because the value is also read back by the application; wrap it with
// RequestCounter[T](counter) to get the Handler that increments it per request.
// It is generic, so the numeric type cannot be an Options field.
package counter

import (
	"math"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/kataras/iris/v14/core/pkg/constraints"
)

// GlobalCounter is the shared counter instances between Iris applications of the same process.
var GlobalCounter = NewCounter[uint64]() // it's not used anywhere, currently but it's here.

// Number is the interface that wraps the basic numerical types.
type Number interface {
	constraints.Number
}

// Notification contains old and new values for the counter.
type Notification[T Number] struct {
	OldValue T
	NewValue T
}

// Counter is a counter which
// atomically increments until Max.
//
// The T can be any type that implements the Number interface,
// which includes all the numerical basic types
// (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64).
type Counter[T Number] struct {
	value atomic.Pointer[T]
	Min   T
	Max   T

	// Guards the lazy creation of notifications against a subscriber racing an
	// increment. Not on the counter value itself, which stays lock-free.
	mu            sync.Mutex
	notifications chan Notification[T]
}

// NewCounter returns a fresh instance of a counter of any basic numerical type.
// End developers can use it as a helper for their applications.
//
// Min and Max are derived from T's kind and width, so a Counter[uint8] saturates
// at 255 and a Counter[int64] at math.MaxInt64, on every platform. Defined types
// with a numeric underlying type (`type Hits uint32`) are supported too.
func NewCounter[T Number]() *Counter[T] {
	var (
		max       T
		min       T
		zeroValue T
	)

	// Written through reflection rather than by reinterpreting the address of an
	// untyped constant, which is how this went wrong before. `limit :=
	// math.MaxUint32` gives an *int*, so the old code copied the first bytes of
	// an int over a uint32 — correct only on a 64-bit little-endian machine, a
	// compile error on a 32-bit one, and silently garbage for float32, where
	// math.MaxFloat32 defaults to float64 and the low four bytes of its bit
	// pattern read back as roughly -3.7e19. It is the same class of bug as
	// kataras/iris#2608: a constant whose width is not the width you assumed.
	//
	// Deriving both bounds from Type().Bits() instead has no such assumption. It
	// runs once per counter, so reflection costs nothing that matters.
	high := reflect.ValueOf(&max).Elem()
	low := reflect.ValueOf(&min).Elem()

	switch kind := high.Kind(); kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// The lowest value of an n-bit two's complement integer is -1 shifted
		// left n-1 places, and the highest is its complement.
		lowest := int64(-1) << (high.Type().Bits() - 1)
		high.SetInt(^lowest)
		low.SetInt(lowest)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		high.SetUint(^uint64(0) >> (64 - high.Type().Bits()))
		low.SetUint(0)
	case reflect.Float32:
		high.SetFloat(math.MaxFloat32)
		low.SetFloat(0)
	case reflect.Float64:
		high.SetFloat(math.MaxFloat64)
		low.SetFloat(0)
	}

	// notifications stays nil until somebody calls Notifications. Creating it
	// here made trySendNotification fire on every single change, and because the
	// channel is unbuffered and nobody was reading it, each send parked a
	// goroutine forever. RequestCounter increments and decrements once per
	// request, so a server using this middleware leaked two goroutines per
	// request until it fell over.
	c := &Counter[T]{
		Max: max,
		Min: min,
	}
	c.value.Store(&zeroValue)
	return c
}

// Notifications returns the read-only channel that can be used to listen for counter changes.
func (c *Counter[T]) Notifications() <-chan Notification[T] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.notifications == nil { // enable only if requested.
		c.notifications = make(chan Notification[T]) // blocking with zero cap.
	}

	return c.notifications
}

// trySendNotification delivers a change to a subscriber, if there is one.
//
// The send is a goroutine on an unbuffered channel, so a caller that asked for
// Notifications and then stopped reading parks one goroutine per change. That
// is the subscriber's contract to keep: having asked for every change, it has to
// take them. A counter nobody subscribed to sends nothing and allocates nothing.
func (c *Counter[T]) trySendNotification(oldValue, newValue T) {
	c.mu.Lock()
	ch := c.notifications
	c.mu.Unlock()

	if ch == nil {
		return
	}

	go func() {
		ch <- Notification[T]{OldValue: oldValue, NewValue: newValue}
	}()
}

// Increment increments the Value.
// The value cannot exceed the Max one.
// It uses Compare and Swap with the atomic package.
//
// Returns the new number value.
func (c *Counter[T]) Increment() (newValue T) {
	for {
		ptr := c.value.Load()
		prev := *ptr
		if prev == c.Max {
			return c.Max
		}

		newValue = prev + 1
		if newValue > c.Max {
			return c.Max
		}

		// Swapped against the pointer this iteration read, not against whatever
		// is stored by the time the swap runs. Re-reading it inside the swap
		// would make every attempt succeed and lose concurrent updates: two
		// goroutines both read 5, both store 6, and one increment vanishes.
		if c.compareAndSwap(ptr, newValue) {
			c.trySendNotification(prev, newValue)
			break
		}
	}

	return
}

// Decrement decrements the Value.
// The value cannot be less than the Min one (0 for the unsigned integer and the
// float types, the most negative value of T for the signed integer types).
// It uses Compare and Swap with the atomic package.
//
// Returns the new number value.
func (c *Counter[T]) Decrement() (newValue T) {
	for {
		ptr := c.value.Load()
		prev := *ptr
		if prev == c.Min {
			return c.Min
		}

		newValue = prev - 1
		if newValue < c.Min {
			// Nothing was stored, so report the floor rather than the value that
			// was refused. Increment does the same with Max.
			return c.Min
		}

		if c.compareAndSwap(ptr, newValue) {
			c.trySendNotification(prev, newValue)
			break
		}
	}

	return
}

// Get returns the current counter without incrementing.
func (c *Counter[T]) Get() T {
	return *c.value.Load()
}

// Set sets the counter to a new value, unconditionally.
// It always returns true; the result is kept for backwards compatibility.
//
// Set does not consult Min or Max. Use Increment and Decrement for a value that
// has to stay inside the counter's bounds.
func (c *Counter[T]) Set(newValue T) bool {
	c.value.Store(&newValue)
	return true
}

// compareAndSwap stores newValue only if the counter still holds the exact
// pointer `old`. Every Set allocates, so a value that went 5 -> 6 -> 5 leaves a
// different pointer behind and the swap correctly fails: no ABA here.
func (c *Counter[T]) compareAndSwap(old *T, newValue T) bool {
	return c.value.CompareAndSwap(old, &newValue)
}
