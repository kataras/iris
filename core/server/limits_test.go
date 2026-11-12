package server

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
)

// TestLimitsOptionsApplyToServer tests that ApplyToServer is nil-safe,
// writes only the set (non-zero/true) options and never overwrites
// a customization already made to the server.
func TestLimitsOptionsApplyToServer(t *testing.T) {
	srv := &http.Server{}

	var nilOpts *LimitsOptions
	nilOpts.ApplyToServer(srv) // must not panic.

	zero := &LimitsOptions{}
	zero.ApplyToServer(srv)
	if srv.MaxHeaderBytes != 0 || srv.MaxHeaderValueCount != 0 || srv.DisableClientPriority {
		t.Fatalf("zero options must leave the server untouched, got: %+v", srv)
	}

	opts := &LimitsOptions{
		MaxHeaderBytes:        1 << 19,
		MaxHeaderValueCount:   64,
		DisableClientPriority: true,
	}
	opts.ApplyToServer(srv)
	if srv.MaxHeaderBytes != 1<<19 {
		t.Errorf("expected MaxHeaderBytes %d but got %d", 1<<19, srv.MaxHeaderBytes)
	}
	if srv.MaxHeaderValueCount != 64 {
		t.Errorf("expected MaxHeaderValueCount 64 but got %d", srv.MaxHeaderValueCount)
	}
	if !srv.DisableClientPriority {
		t.Error("expected DisableClientPriority to be set")
	}

	customized := &http.Server{MaxHeaderBytes: 4096, MaxHeaderValueCount: 10}
	opts.ApplyToServer(customized)
	if customized.MaxHeaderBytes != 4096 {
		t.Errorf("expected the server's own MaxHeaderBytes (4096) to win but got %d", customized.MaxHeaderBytes)
	}
	if customized.MaxHeaderValueCount != 10 {
		t.Errorf("expected the server's own MaxHeaderValueCount (10) to win but got %d", customized.MaxHeaderValueCount)
	}
}

// TestServerOnServerCreated tests that the OnServerCreated hook runs after the
// Timeout and Limits options are applied, and that a hook error aborts the
// startup, closing the listener.
func TestServerOnServerCreated(t *testing.T) {
	errStop := errors.New("stop before serving")

	var captured *http.Server
	server := NewServer(ServerOptions{
		Addr: "127.0.0.1:0",
		Limits: &LimitsOptions{
			MaxHeaderBytes:        1 << 19,
			MaxHeaderValueCount:   64,
			DisableClientPriority: true,
		},
		OnServerCreated: func(srv *http.Server) error {
			captured = srv
			return errStop
		},
		DisableStartupLog:       true,
		DisableInterruptHandler: true,
	})

	var addr string
	server.OnListenerCreated(func(ln net.Listener, _ *url.URL) error {
		addr = ln.Addr().String()
		return nil
	})

	if err := server.ListenAndServe(http.NotFoundHandler()); !errors.Is(err, errStop) {
		t.Fatalf("expected the hook's error to abort ListenAndServe, got: %v", err)
	}

	if captured == nil {
		t.Fatal("expected the hook to receive the http.Server instance")
	}
	if captured.MaxHeaderBytes != 1<<19 || captured.MaxHeaderValueCount != 64 || !captured.DisableClientPriority {
		t.Errorf("expected the hook to see the Limits already applied, got: MaxHeaderBytes=%d MaxHeaderValueCount=%d DisableClientPriority=%v",
			captured.MaxHeaderBytes, captured.MaxHeaderValueCount, captured.DisableClientPriority)
	}
	// The listener must be closed on hook failure: binding the same address again should succeed.
	if addr == "" {
		t.Fatal("expected the listener to have been created")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("expected the aborted server's listener to be closed, cannot rebind %s: %v", addr, err)
	}
	ln.Close()
}
