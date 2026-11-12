package closer

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewCloser tests the creation of new closers with different function types.
func TestNewCloser(t *testing.T) {
	// Test creation with an io.Closer
	mockCloser := new(mockIoCloser)
	c := newCloser(mockCloser)
	if c.closerValue != mockCloser {
		t.Errorf("newCloser did not correctly assign io.Closer")
	}

	// Test creation with a function with no arguments and no return value
	fn := func() {}
	c = newCloser(fn)
	if c.fn == nil {
		t.Errorf("newCloser did not correctly assign func()")
	}

	// Test creation with a function with error return
	fnWithError := func() error { return nil }
	c = newCloser(fnWithError)
	if c.fnWithError == nil {
		t.Errorf("newCloser did not correctly assign func() error")
	}

	// Test creation with a function with context argument
	fnWithContext := func(ctx context.Context) {}
	c = newCloser(fnWithContext)
	if c.fnWithContext == nil {
		t.Errorf("newCloser did not correctly assign func(context.Context)")
	}

	// Test creation with a function with context argument and error return
	fnWithContextAndError := func(ctx context.Context) error { return nil }
	c = newCloser(fnWithContextAndError)
	if c.fnWithContextAndError == nil {
		t.Errorf("newCloser did not correctly assign func(context.Context) error")
	}
}

// TestCloser_Close tests the Close method of the closer type.
func TestCloser_Close(t *testing.T) {
	// Test closing with an io.Closer
	mockCloser := new(mockIoCloser)
	c := newCloser(mockCloser)
	err := c.Close()
	if err != nil {
		t.Errorf("Close returned an error for io.Closer")
	}
	if !mockCloser.closed {
		t.Errorf("Close did not call Close on the io.Closer")
	}

	// Test closing with a function that has no arguments and no return value
	called := false
	fn := func() { called = true }
	c = newCloser(fn)
	err = c.Close()
	if err != nil {
		t.Errorf("Close returned an error for func()")
	}
	if !called {
		t.Errorf("Close did not call the function for func()")
	}

	// Test closing with a function that returns an error
	expectedErr := errors.New("error")
	fnWithError := func() error { return expectedErr }
	c = newCloser(fnWithError)
	err = c.Close()
	if err != expectedErr {
		t.Errorf("Close did not return the expected error for func() error")
	}

	// Test closing with a function that takes a context
	calledWithContext := false
	fnWithContext := func(ctx context.Context) { calledWithContext = true }
	c = newCloser(fnWithContext)
	err = c.Close()
	if err != nil {
		t.Errorf("Close returned an error for func(context.Context)")
	}
	if !calledWithContext {
		t.Errorf("Close did not call the function for func(context.Context)")
	}

	// Test closing with a function that takes a context and returns an error
	fnWithContextAndError := func(ctx context.Context) error { return expectedErr }
	c = newCloser(fnWithContextAndError)
	err = c.Close()
	if err != expectedErr {
		t.Errorf("Close did not return the expected error for func(context.Context) error")
	}
}

// TestCloser_CloseWithContext tests the CloseWithContext method of the closer type.
func TestCloser_CloseWithContext(t *testing.T) {
	// Test closing with a context that times out
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	time.Sleep(time.Second) // Ensure the context times out.

	fn := func(context.Context) error {
		select {
		case <-ctx.Done():
			// Return the context's error, which could be DeadlineExceeded if it timed out.
			return ctx.Err()
		default:
			return nil
		}
	}
	c := newCloser(fn)
	err := c.CloseWithContext(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("CloseWithContext did not return context.DeadlineExceeded for timed-out context, got %v", err)
	}
}

// mockIoCloser is a mock implementation of io.Closer for testing.
type mockIoCloser struct {
	closed bool
}

func (m *mockIoCloser) Close() error {
	m.closed = true
	return nil
}

// TestClosers_CloseWithContext_JoinsErrors verifies that errors from multiple
// closers are accumulated via errors.Join, so callers can match every
// underlying error individually with errors.Is.
func TestClosers_CloseWithContext_JoinsErrors(t *testing.T) {
	errA := errors.New("err A")
	errB := errors.New("err B")
	errC := errors.New("err C")

	var cs Closers
	cs.AddCloser(func() error { return errA })
	cs.AddCloser(func() error { return nil }) // success between failures
	cs.AddCloser(func() error { return errB })
	cs.AddCloser(func() error { return errC })

	joined := cs.CloseWithContext(context.Background())
	if joined == nil {
		t.Fatal("CloseWithContext returned nil, expected joined error")
	}
	if !errors.Is(joined, errA) || !errors.Is(joined, errB) || !errors.Is(joined, errC) {
		t.Fatalf("joined error does not match all underlying errors: %v", joined)
	}
}

// TestClosers_PrependCloser_PreservesOrder verifies that PrependCloser inserts
// at the head of the slice so the closer runs before previously-registered ones.
func TestClosers_PrependCloser_PreservesOrder(t *testing.T) {
	var order []string
	var cs Closers
	cs.AddCloser(func() { order = append(order, "first-added") })
	cs.AddCloser(func() { order = append(order, "second-added") })
	cs.PrependCloser(func() { order = append(order, "prepended") })

	if err := cs.CloseWithContext(context.Background()); err != nil {
		t.Fatalf("CloseWithContext returned unexpected error: %v", err)
	}

	want := []string{"prepended", "first-added", "second-added"}
	if len(order) != len(want) {
		t.Fatalf("call order length: got %d, want %d (%v)", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("call order: got %v, want %v", order, want)
		}
	}
}
