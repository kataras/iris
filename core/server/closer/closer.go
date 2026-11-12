package closer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"
)

type (
	// Closer is the interface that wraps the CloseWithContext method.
	Closer interface {
		// CloseWithContext calls the closer with the given context.
		CloseWithContext(context.Context) error
		// Close() error
	}

	// Closers is a collection of closers.
	// It is used to store all the closers that need to be called when the server is terminated.
	Closers []Closer
)

// AddCloser adds a closer to the server.
// The closer will be called when the server is closed.
// The closer can be a function that takes no arguments, a function that returns an error,
// a function that takes a context, a function that takes a context and returns an error or,
// a value that implements the io.Closer or this package's Closer interface.
func (cs *Closers) AddCloser(v any) {
	if v == nil {
		return
	}

	closer := newCloser(v)
	// signalhandler.Register(closer.Close) // register each closer to the signal handler too.

	*cs = append(*cs, closer)
}

// PrependCloser adds a closer to the server but at the beginning of the closers list.
func (cs *Closers) PrependCloser(v any) {
	if v == nil {
		return
	}

	closer := newCloser(v)
	// signalhandler.Register(closer.Close) // register each closer to the signal handler too.

	*cs = slices.Insert(*cs, 0, Closer(closer))
}

// CloseWithContext calls the CloseWithContext method on all closers.
// Errors from individual closers are accumulated with errors.Join so callers
// can match each underlying error via errors.Is / errors.As.
func (cs *Closers) CloseWithContext(ctx context.Context) (joinedErr error) {
	for _, closer := range *cs {
		if closeErr := closer.CloseWithContext(ctx); closeErr != nil {
			joinedErr = errors.Join(joinedErr, closeErr)
		}
	}

	return
}

// closer is a helper type that wraps a function to be called when the server is closed.
type closer struct {
	closerValue            io.Closer
	closerWithContextValue Closer
	fn                     func()
	fnWithError            func() error
	fnWithContext          func(context.Context)
	fnWithContextAndError  func(context.Context) error
}

// newCloser wraps value into a *closer that can be invoked at shutdown.
// The supported concrete types are:
//   - *closer (returned as-is)
//   - io.Closer
//   - Closer (this package's context-aware variant)
//   - func()
//   - func() error
//   - func(context.Context)
//   - func(context.Context) error
//
// Any other type triggers a panic at registration time so the misuse is
// caught immediately instead of at server shutdown.
// See Server.AddCloser for more information.
func newCloser(value any) *closer {
	switch v := value.(type) {
	case *closer:
		return v // just in case.
	case io.Closer:
		return &closer{closerValue: v}
	case Closer:
		return &closer{closerWithContextValue: v}
	case func():
		return &closer{fn: v}
	case func() error:
		return &closer{fnWithError: v}
	case func(context.Context):
		return &closer{fnWithContext: v}
	case func(context.Context) error:
		return &closer{fnWithContextAndError: v}
	}

	panic(fmt.Errorf("server: newCloser expects a func() or func() error or func(context.Context) or func(context.Context) error or an io.Closer but got %T", value))
}

// Close calls the registered function.
// It completes the io.Closer interface.
func (c *closer) Close() error {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second) // give each job 1 second to finish, as this is meant to "forcely" call the closers.
	defer cancelFn()

	return c.CloseWithContext(ctx)
}

// CloseWithContext calls the registered closer.
func (c *closer) CloseWithContext(ctx context.Context) error {
	if c.closerValue != nil {
		return c.closerValue.Close()
	}

	if c.closerWithContextValue != nil {
		return c.closerWithContextValue.CloseWithContext(ctx)
	}

	if c.fn != nil {
		c.fn()
		return nil
	}

	if c.fnWithError != nil {
		return c.fnWithError()
	}

	// No need, let the custom closers handle the passed context.
	// Check if the context is done.
	// select {
	// case <-ctx.Done():
	// 	// Return the context's error, which could be DeadlineExceeded if it timed out.
	// 	return ctx.Err()
	// default:
	if c.fnWithContext != nil {
		c.fnWithContext(ctx)
		return nil
	}

	if c.fnWithContextAndError != nil {
		return c.fnWithContextAndError(ctx)
	}

	return nil
}
