package signalhandler

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The closers of an interrupt signal receive a context bounded by the
// ShutdownOnInterruptTimeout of the running server, and Wait blocks until the
// last of them has returned.
//
// The package keeps its state in package-level variables and the signal is
// caught once per process, so this is one test that runs the whole sequence,
// from the state newProcess sets.
func TestOnSignal(t *testing.T) {
	exited := newProcess()

	// Wait returns at once before a signal.
	done := make(chan struct{})
	go func() { Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Wait blocked before a signal")
	}

	var (
		registered = make(chan time.Duration, 1) // the deadline of the context.
		drained    = make(chan struct{})
		slowDone   = make(chan struct{})
	)

	// A server registers its own closer with its timeout, and a second server
	// one with a smaller timeout: the larger of the two wins.
	RegisterWithTimeout(func(ctx context.Context) error {
		defer close(drained) // the server has shut down: this is where its Run is released.

		deadline, ok := ctx.Deadline()
		if !ok {
			registered <- 0
			return nil
		}
		registered <- time.Until(deadline).Round(time.Second)
		return nil
	}, 30*time.Second)
	RegisterWithTimeout(func() error { return nil }, 5*time.Second)

	// A closer that takes its time: Wait must not return before it is done.
	Register(func() error {
		time.Sleep(300 * time.Millisecond)
		close(slowDone)
		return nil
	})

	// What Run does: it gets to Wait when the server's own closer has run, which
	// is after the signal arrived and before the closers after it are done.
	waited := make(chan struct{})
	go func() {
		<-drained
		Wait()
		close(waited)
	}()

	go onSignal()

	select {
	case <-waited:
	case <-time.After(10 * time.Second):
		t.Fatal("Wait did not return after the closers")
	}

	select {
	case <-slowDone:
	default:
		t.Error("Wait returned while a closer was still running")
	}

	select {
	case got := <-registered:
		if want := 30 * time.Second; got != want {
			t.Errorf("expected the closers to get a context with a %s deadline, got %s", want, got)
		}
	default:
		t.Error("the closer did not run")
	}

	select {
	case code := <-exited:
		if code != 0 {
			t.Errorf("expected the process to exit with 0, got %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the process was not told to exit")
	}
}

// A closer that fails ends the process with status 1, and Wait is still
// blocked when it does. Released first, a main that ends with Run would end the
// process itself, with status 0 and before the error is printed.
func TestOnSignalError(t *testing.T) {
	exited := newProcess()

	Register(func() error { return errors.New("the closer failed") })

	waited := make(chan struct{})
	go func() {
		<-triggered
		Wait()
		close(waited)
	}()

	go onSignal()

	select {
	case code := <-exited:
		if code != 1 {
			t.Errorf("expected the process to exit with 1, got %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the process was not told to exit")
	}

	select {
	case <-waited:
		t.Error("Wait returned before the process was told to exit with 1")
	case <-time.After(100 * time.Millisecond):
	}

	// os.Exit does not return, so nothing releases Wait. The test does.
	close(finished)
	<-waited
}

// A zero or negative timeout sets no deadline, and neither does a process whose
// servers registered none.
func TestOnSignalNoDeadline(t *testing.T) {
	tests := []struct {
		name     string
		timeouts []time.Duration
	}{
		{"no server registered", nil},
		{"zero", []time.Duration{0}},
		{"negative", []time.Duration{-time.Second}},
		// A server that asked for no deadline is not cut by the limit of
		// another one, whichever of the two registered first.
		{"zero after a limit", []time.Duration{5 * time.Second, 0}},
		{"zero before a limit", []time.Duration{0, 5 * time.Second}},
		{"negative after a limit", []time.Duration{5 * time.Second, -time.Second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exited := newProcess()

			for _, d := range tt.timeouts {
				RegisterWithTimeout(func() error { return nil }, d)
			}

			hasDeadline := make(chan bool, 1)
			Register(func(ctx context.Context) {
				_, ok := ctx.Deadline()
				hasDeadline <- ok
			})

			go onSignal()

			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				t.Fatal("the process was not told to exit")
			}

			if <-hasDeadline {
				t.Error("expected the closers to get a context with no deadline")
			}
		})
	}
}

// With more than one server the largest timeout is the deadline, whichever of
// them registered first.
func TestOnSignalLargestTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeouts []time.Duration
	}{
		{"largest first", []time.Duration{30 * time.Second, 5 * time.Second}},
		{"largest last", []time.Duration{5 * time.Second, 30 * time.Second}},
		{"largest between", []time.Duration{5 * time.Second, 30 * time.Second, 10 * time.Second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exited := newProcess()

			for _, d := range tt.timeouts {
				RegisterWithTimeout(func() error { return nil }, d)
			}

			left := make(chan time.Duration, 1) // zero: no deadline.
			Register(func(ctx context.Context) {
				deadline, ok := ctx.Deadline()
				if !ok {
					left <- 0
					return
				}
				left <- time.Until(deadline).Round(time.Second)
			})

			go onSignal()

			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				t.Fatal("the process was not told to exit")
			}

			if got, want := <-left, 30*time.Second; got != want {
				t.Errorf("expected the closers to get a context with a %s deadline, got %s (zero: none)", want, got)
			}
		})
	}
}

// newProcess puts the package in the state of a process that has received no
// signal, so a test can run more than once (go test -count), and replaces exit.
// The channel carries the status the process was told to exit with.
func newProcess() <-chan int {
	mu.Lock()
	closers, timeout, unbounded = nil, 0, false
	triggered, finished = make(chan struct{}), make(chan struct{})
	mu.Unlock()

	exited := make(chan int, 1)
	exit = func(code int) { exited <- code }
	return exited
}
