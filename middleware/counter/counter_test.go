package counter

import (
	"io"
	"math"
	"runtime"
	rpprof "runtime/pprof"
	"sync"
	"testing"
	"time"
)

func TestCounterInt(t *testing.T) {
	counter := NewCounter[int]()

	if counter.Max != math.MaxInt {
		t.Errorf("Expected Max to be %d, got %d", math.MaxInt, counter.Max)
	}

	counter.Increment()
	if counter.Get() != 1 {
		t.Errorf("Expected counter value to be 1, got %d", counter.Get())
	}

	counter.Decrement()
	if counter.Get() != 0 {
		t.Errorf("Expected counter value to be 0, got %d", counter.Get())
	}
}

func TestCounterUint64(t *testing.T) {
	counter := NewCounter[uint64]()

	if counter.Max != math.MaxUint64 {
		t.Errorf("Expected Max to be %d, got %d", uint64(math.MaxUint64), counter.Max)
	}

	counter.Increment()
	if counter.Get() != 1 {
		t.Errorf("Expected counter value to be 1, got %d", counter.Get())
	}

	counter.Decrement()
	if counter.Get() != 0 {
		t.Errorf("Expected counter value to be 0, got %d", counter.Get())
	}
}

func TestCounterFloat64(t *testing.T) {
	counter := NewCounter[float64]()

	if counter.Max != math.MaxFloat64 {
		t.Errorf("Expected Max to be %f, got %f", math.MaxFloat64, counter.Max)
	}

	counter.Increment()
	if counter.Get() != 1.0 {
		t.Errorf("Expected counter value to be 1.0, got %f", counter.Get())
	}

	counter.Decrement()
	if counter.Get() != 0.0 {
		t.Errorf("Expected counter value to be 0.0, got %f", counter.Get())
	}
}

func TestCounterMaxValue(t *testing.T) {
	counter := NewCounter[int8]()
	counter.value.Store(&counter.Max)

	counter.Increment()
	if counter.Get() != math.MaxInt8 {
		t.Errorf("Expected counter value to be %d, got %d", math.MaxInt8, counter.Get())
	}

	counter.Decrement()
	if counter.Get() != math.MaxInt8-1 {
		t.Errorf("Expected counter value to be %d, got %d", math.MaxInt8-1, counter.Get())
	}
}

// bounds asserts the Min and Max that NewCounter derives for T, and that a fresh
// counter still counts from zero.
//
// Every type in the Number set gets a case below, deliberately: the bugs this
// covers were only reachable through the types nobody had instantiated. Max for
// a uint32 was built by copying the first bytes of an int, which is a compile
// error on a 32-bit platform and the wrong end of the word on a big-endian one,
// and Max for a float32 was the low half of a float64's bit pattern.
func bounds[T Number](t *testing.T, wantMin, wantMax T) {
	t.Helper()

	c := NewCounter[T]()

	if c.Max != wantMax {
		t.Errorf("Max: got %v, want %v", c.Max, wantMax)
	}
	if c.Min != wantMin {
		t.Errorf("Min: got %v, want %v", c.Min, wantMin)
	}

	if got := c.Increment(); got != 1 {
		t.Errorf("Increment from zero: got %v, want 1", got)
	}
	if got := c.Decrement(); got != 0 {
		t.Errorf("Decrement back to zero: got %v, want 0", got)
	}
}

func TestCounterBounds(t *testing.T) {
	// Keep the explicit type arguments. The bounds are untyped constants, so
	// inference would default most of these rows to int or float64 and
	// silently test that type while the subtest name still claimed int8 or
	// float32. Only the int, uintptr and float64 rows would infer what they
	// say, and spelling those three differently from the other ten is how the
	// next reader talks themselves into dropping all thirteen.
	t.Run("int", func(t *testing.T) { bounds[int](t, math.MinInt, math.MaxInt) })
	t.Run("int8", func(t *testing.T) { bounds[int8](t, math.MinInt8, math.MaxInt8) })
	t.Run("int16", func(t *testing.T) { bounds[int16](t, math.MinInt16, math.MaxInt16) })
	t.Run("int32", func(t *testing.T) { bounds[int32](t, math.MinInt32, math.MaxInt32) })
	t.Run("int64", func(t *testing.T) { bounds[int64](t, math.MinInt64, math.MaxInt64) })
	t.Run("uint", func(t *testing.T) { bounds[uint](t, 0, math.MaxUint) })
	t.Run("uint8", func(t *testing.T) { bounds[uint8](t, 0, math.MaxUint8) })
	t.Run("uint16", func(t *testing.T) { bounds[uint16](t, 0, math.MaxUint16) })
	t.Run("uint32", func(t *testing.T) { bounds[uint32](t, 0, math.MaxUint32) })
	t.Run("uint64", func(t *testing.T) { bounds[uint64](t, 0, math.MaxUint64) })
	t.Run("uintptr", func(t *testing.T) { bounds[uintptr](t, 0, ^uintptr(0)) })
	t.Run("float32", func(t *testing.T) { bounds[float32](t, 0, math.MaxFloat32) })
	t.Run("float64", func(t *testing.T) { bounds[float64](t, 0, math.MaxFloat64) })
}

// requests is a defined type over uint32, which the Number constraint admits
// because it is written with ~.
type requests uint32

func TestCounterDefinedType(t *testing.T) {
	// A type switch on the concrete type matched only the predeclared types, so
	// a counter of a defined type used to come back with Max and Min both zero
	// and refuse to count past nothing. Deriving the bounds from the reflected
	// kind covers both.
	c := NewCounter[requests]()

	if want := requests(math.MaxUint32); c.Max != want {
		t.Errorf("Max: got %d, want %d", c.Max, want)
	}
	if c.Min != 0 {
		t.Errorf("Min: got %d, want 0", c.Min)
	}
	if got := c.Increment(); got != 1 {
		t.Errorf("Increment: got %d, want 1", got)
	}
}

func TestCounterSaturates(t *testing.T) {
	c := NewCounter[uint8]()

	c.Set(math.MaxUint8 - 1)
	if got := c.Increment(); got != math.MaxUint8 {
		t.Errorf("Increment to the ceiling: got %d, want %d", got, uint8(math.MaxUint8))
	}
	// Already at Max: it has to stay there rather than wrap to zero.
	if got := c.Increment(); got != math.MaxUint8 {
		t.Errorf("Increment at the ceiling: got %d, want %d", got, uint8(math.MaxUint8))
	}

	c.Set(0)
	if got := c.Decrement(); got != 0 {
		t.Errorf("Decrement at the floor: got %d, want 0", got)
	}
}

func TestCounterIncrementConcurrent(t *testing.T) {
	// Set used to load its own comparand, so the compare-and-swap always saw the
	// value it had just read and always succeeded. Two goroutines could both
	// read 5 and both store 6, and one increment disappeared.
	c := NewCounter[int]()

	const workers, each = 8, 1000

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range each {
				c.Increment()
			}
		})
	}
	wg.Wait()

	if got := c.Get(); got != workers*each {
		t.Errorf("lost updates under concurrency: got %d, want %d", got, workers*each)
	}
}

func TestCounterNoSubscriberSpawnsNothing(t *testing.T) {
	// RequestCounter increments and decrements once per request. While the
	// notification channel was created up front, every one of those parked a
	// goroutine on an unbuffered channel nobody was reading, so a server using
	// the middleware leaked two goroutines per request.
	//
	// Two detectors, because they see different things. The goroutineleak
	// profile only reports goroutines whose blocking is provably permanent:
	// parked on a sync object the GC can no longer reach. This test holds c,
	// and c holds the channel, so goroutines parked on it could in principle
	// still be freed by a reader - measured on go1.27.0, five hundred of them
	// report a count of zero. NumGoroutine therefore stays the primary signal
	// for the failure shape above, and the profile is a supplementary
	// zero-growth check for the variant where the counter itself has been
	// dropped. Its detection pass runs only inside WriteTo; Count without one
	// reads a stale (initially empty) profile.
	leaked := func() int {
		p := rpprof.Lookup("goroutineleak")
		if err := p.WriteTo(io.Discard, 0); err != nil {
			t.Fatalf("goroutineleak profile: %v", err)
		}
		return p.Count()
	}

	before := runtime.NumGoroutine()
	leakedBefore := leaked()

	c := NewCounter[uint64]()
	for range 500 {
		c.Increment()
		c.Decrement()
	}

	// Slack for whatever the test runtime is doing alongside us; the failure this
	// guards against is a thousand goroutines, not five.
	if after := runtime.NumGoroutine(); after > before+5 {
		t.Errorf("goroutines grew from %d to %d with nobody subscribed", before, after)
	}
	if leakedAfter := leaked(); leakedAfter > leakedBefore {
		t.Errorf("goroutineleak profile grew from %d to %d with nobody subscribed", leakedBefore, leakedAfter)
	}
}

func TestCounterNotifications(t *testing.T) {
	c := NewCounter[int]()

	// Subscribe before generating a change. A counter nobody subscribed to sends
	// nothing, which is what keeps RequestCounter from parking a goroutine on
	// every request.
	notifications := c.Notifications()

	// Define a timeout for receiving from the channel.
	timeout := time.Second

	// Increment the counter and test the notification.
	go func() {
		c.Increment()
	}()

	select {
	case notif := <-notifications:
		if notif.OldValue != 0 || notif.NewValue != 1 {
			t.Errorf("expected old value 0 and new value 1, got old value %d and new value %d", notif.OldValue, notif.NewValue)
		}
	case <-time.After(timeout):
		t.Errorf("increment notification timed out")
	}

	// Give the goroutine enough time to complete.
	time.Sleep(time.Millisecond * 10)

	// Decrement the counter and test the notification.
	go func() {
		c.Decrement()
	}()

	select {
	case notif := <-notifications:
		if notif.OldValue != 1 || notif.NewValue != 0 {
			t.Errorf("expected old value 1 and new value 0, got old value %d and new value %d", notif.OldValue, notif.NewValue)
		}
	case <-time.After(timeout):
		t.Errorf("decrement notification timed out")
	}
}

func TestCounterNotifications_SelectRoutine(t *testing.T) {
	c := NewCounter[int]()

	// Subscribe on this goroutine, so the channel exists before either the
	// reader below or the increment that follows it.
	notifications := c.Notifications()

	// Define a timeout for receiving from the channel.
	timeout := 2 * time.Second

	go func() {
		select {
		case notif := <-notifications:
			if notif.OldValue != 0 || notif.NewValue != 1 {
				t.Errorf("expected old value 0 and new value 1, got old value %d and new value %d", notif.OldValue, notif.NewValue)
			}
		case <-time.After(timeout):
			t.Errorf("increment notification timed out")
		}
	}()

	// Increment the counter and test the notification.
	c.Increment()

	// Give the goroutine enough time to complete.
	time.Sleep(time.Millisecond * 10)

	go func() {
		select {
		case notif := <-notifications:
			if notif.OldValue != 1 || notif.NewValue != 0 {
				t.Errorf("expected old value 1 and new value 0, got old value %d and new value %d", notif.OldValue, notif.NewValue)
			}
		case <-time.After(timeout):
			t.Errorf("decrement notification timed out")
		}
	}()

	// Decrement the counter and test the notification.
	c.Decrement()

	// Give the goroutine enough time to complete.
	time.Sleep(time.Millisecond * 10)
}
