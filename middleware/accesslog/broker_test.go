package accesslog

import (
	"io"
	"testing"
	"time"
)

// Regression test: the broker's run goroutine must exit when the AccessLog
// is closed. It used to keep looping after the close signal (its close case
// had no return), a leak the goroutineleak profile reports.
func TestBrokerRunExitsOnClose(t *testing.T) {
	ac := New(io.Discard)
	broker := ac.Broker()
	ln := broker.NewListener()

	if err := ac.Close(); err != nil {
		t.Fatal(err)
	}

	// Close sends on broker.close synchronously; run then closes the
	// listeners and returns, closing broker.done on its way out.
	select {
	case <-broker.done:
	case <-time.After(5 * time.Second):
		t.Fatal("broker run goroutine still alive after Close")
	}

	if _, ok := <-ln; ok {
		t.Fatal("expected the listener channel to be closed")
	}
}
