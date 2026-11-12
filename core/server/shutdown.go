package server

import (
	"context"
)

// shutdownSignalKeyType is unexported so nothing outside this package can put a
// value under it, which keeps ShutdownSignal's answer trustworthy.
type shutdownSignalKeyType struct{}

var shutdownSignalKey shutdownSignalKeyType

// withShutdownSignal returns a base context carrying ch, from which every
// request context of the server derives.
func withShutdownSignal(parent context.Context, ch <-chan struct{}) context.Context {
	if ch == nil {
		return parent
	}

	return context.WithValue(parent, shutdownSignalKey, ch)
}

// ShutdownSignal returns the channel that is closed when the server carrying ctx
// begins a graceful shutdown, or nil when ctx did not come from a server this
// package started.
//
// It answers a question net/http does not. A request context is cancelled when
// the client disconnects, when an HTTP/2 stream is reset, or when ServeHTTP
// returns, and shutdown is on none of those lists - so a handler blocked in a
// stream loop is never told that the process is going away. http.Server.Shutdown
// then waits for a handler that has no way of knowing it should stop, until its
// own deadline expires, and returns with the connection still open.
//
// A nil channel blocks forever in a select, which is the behaviour a handler
// wants when there is no server to shut down.
//
// Reach it from a handler through Context.ShuttingDown rather than calling this
// directly.
func ShutdownSignal(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}

	ch, _ := ctx.Value(shutdownSignalKey).(<-chan struct{})
	return ch
}
