package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The tests of this file are about the end of a server's life: when ListenAndServe
// returns once a shutdown is asked for, and how long the shutdown of the interrupt
// handler waits. They run in process: the interrupt is never a real signal, it is
// the function the server registers for it (shutdownOnInterrupt).

// inFlightWindow is how long a test watches for something that must not happen
// while a request is still being served.
const inFlightWindow = 250 * time.Millisecond

// heldHandler keeps every request in flight until finish is called.
type heldHandler struct {
	entered  chan struct{} // receives when a request reached the handler.
	release  chan struct{} // closed by finish.
	once     sync.Once
	answered atomic.Bool // set once a request was answered.
}

func newHeldHandler(t *testing.T) *heldHandler {
	t.Helper()

	h := &heldHandler{entered: make(chan struct{}, 1), release: make(chan struct{})}
	t.Cleanup(h.finish) // a test that fails early leaves no handler behind.
	return h
}

func (h *heldHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.entered <- struct{}{}
	<-h.release
	io.WriteString(w, "done")
	h.answered.Store(true)
}

// finish lets the held requests return.
func (h *heldHandler) finish() {
	h.once.Do(func() { close(h.release) })
}

// newShutdownTestServer returns a server for a loopback port the system picks.
// It registers no interrupt handler: the tests call what it would run.
func newShutdownTestServer(options ServerOptions) *Server {
	options.Addr = "127.0.0.1:0"
	options.DisableStartupLog = true
	options.DisableInterruptHandler = true
	if options.LogWriter == nil {
		options.LogWriter = io.Discard
	}

	return NewServer(options)
}

// serveInBackground starts the server and returns the "host:port" it listens on
// and the channel that receives what ListenAndServe returns.
func serveInBackground(t *testing.T, server *Server, handler http.Handler) (string, <-chan error) {
	t.Helper()

	addrCh := make(chan string, 1)
	server.OnListenerCreated(func(ln net.Listener, _ *url.URL) error {
		addrCh <- ln.Addr().String()
		return nil
	})

	served := make(chan error, 1)
	go func() { served <- server.ListenAndServe(handler) }()

	select {
	case addr := <-addrCh:
		t.Cleanup(func() { server.Close() })
		return addr, served
	case err := <-served:
		t.Fatalf("ListenAndServe() returned before it listened: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not start listening in time")
	}

	return "", nil
}

// newShutdownTestClient returns a client of its own, so no test reuses the
// connections of another one. It speaks HTTP/2 to a TLS server.
func newShutdownTestClient(t *testing.T) *http.Client {
	t.Helper()

	client := newHTTP2Client()
	t.Cleanup(client.CloseIdleConnections)
	return client
}

// getInBackground sends a GET request and returns the channel that receives
// "<status code> <body>", or "error: <reason>" when the request failed.
func getInBackground(client *http.Client, url string) <-chan string {
	result := make(chan string, 1)

	go func() {
		resp, err := client.Get(url)
		if err != nil {
			result <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result <- "error: " + err.Error()
			return
		}

		result <- fmt.Sprintf("%d %s", resp.StatusCode, body)
	}()

	return result
}

// receive returns what ch delivers. It fails the test when nothing arrives,
// which is how a deadlock shows up here.
func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// inBackground runs fn in a goroutine and returns the channel that receives its error.
func inBackground(fn func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	return done
}

// expectServeToOutlastShutdown starts shutdown while h holds a request in flight
// and checks what follows: ListenAndServe keeps blocking for as long as the request
// is served, the request is answered, and both the shutdown and ListenAndServe
// return nil.
func expectServeToOutlastShutdown(t *testing.T, h *heldHandler, served <-chan error, response <-chan string, shutdown func() error) {
	t.Helper()

	receive(t, h.entered, "the request to reach the handler")

	start := time.Now()
	shutdownErr := inBackground(shutdown)

	select {
	case err := <-served:
		t.Fatalf("ListenAndServe() returned %v, %v after the shutdown began and with a request still in flight", err, time.Since(start))
	case <-time.After(inFlightWindow):
	}

	h.finish()

	if got, want := receive(t, response, "the response"), "200 done"; got != want {
		t.Errorf("expected the request in flight to be answered with %q but got %q", want, got)
	}
	if err := receive(t, shutdownErr, "the shutdown to return"); err != nil {
		t.Errorf("the shutdown failed: %v", err)
	}
	if err := receive(t, served, "ListenAndServe() to return"); err != nil {
		t.Errorf("ListenAndServe() failed: %v", err)
	}
	if !h.answered.Load() {
		t.Error("ListenAndServe() returned before the request in flight was answered")
	}
}

// TestListenAndServeReturnsAfterShutdown is a regression test: http.Server.Serve
// returns the moment a shutdown begins and ListenAndServe returned with it. A
// program that ends when its server stops, a main whose last call is Listen,
// exited while the shutdown was still draining, and its requests in flight were
// reset. ListenAndServe has to return when the shutdown ends, whatever listener
// the server opened.
func TestListenAndServeReturnsAfterShutdown(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)
	certificate, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	// The tunnel options are how a server gets a listener it did not open itself.
	RegisterNgrokListener(func(context.Context, *NgrokOptions, []string, string) (NgrokTunnelListener, error) {
		local, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}

		return &fakeTunnelListener{Listener: local, publicURL: &url.URL{Scheme: "https", Host: "demo.ngrok.app"}}, nil
	})
	t.Cleanup(func() { RegisterNgrokListener(nil) })

	tests := []struct {
		name    string
		scheme  string
		options ServerOptions
	}{
		{"plain", "http", ServerOptions{}},
		{"certificate and key", "https", ServerOptions{CertFile: certPEM, KeyFile: keyPEM}},
		{"custom tls config", "https", ServerOptions{TLSConfig: &tls.Config{Certificates: []tls.Certificate{certificate}}}},
		// A handshake to an IP address carries no server name: the autocert manager
		// refuses it without contacting the CA and the local certificate is served.
		{"letsencrypt with a local fallback", "https", ServerOptions{Letsencrypt: true, CertFile: certPEM, KeyFile: keyPEM}},
		{"custom listener", "http", ServerOptions{Ngrok: &NgrokOptions{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHeldHandler(t)
			s := newShutdownTestServer(tt.options)
			addr, served := serveInBackground(t, s, h)
			response := getInBackground(newShutdownTestClient(t), tt.scheme+"://"+addr)

			expectServeToOutlastShutdown(t, h, served, response, func() error {
				return s.Shutdown(context.Background())
			})
		})
	}
}

// TestShutdownOnInterruptTimeout is a regression test for the shutdown the
// interrupt signal runs. A zero or negative ShutdownOnInterruptTimeout is
// documented as "no timeout" but it was passed to context.WithTimeout as it was,
// which gives a context that is expired already: the graceful shutdown failed at
// once and every connection was closed, so Ctrl+C reset the requests in flight.
// Zero is what a TimeoutOptions literal, or a configuration file whose Timeout
// section leaves the field out, carries. Without a Timeout at all the server
// takes the 15 seconds of the Timeout function.
func TestShutdownOnInterruptTimeout(t *testing.T) {
	noDeadline := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	}

	for _, tt := range noDeadline {
		t.Run(tt.name, func(t *testing.T) {
			h := newHeldHandler(t)
			s := newShutdownTestServer(ServerOptions{Timeout: &TimeoutOptions{ShutdownOnInterruptTimeout: tt.timeout}})
			addr, served := serveInBackground(t, s, h)
			response := getInBackground(newShutdownTestClient(t), "http://"+addr)
			receive(t, h.entered, "the request to reach the handler")

			start := time.Now()
			interrupted := inBackground(s.shutdownOnInterrupt)

			select {
			case err := <-interrupted:
				t.Fatalf("the shutdown on interrupt returned %v after %v with a request still in flight, which got %q",
					err, time.Since(start), receive(t, response, "the response"))
			case <-time.After(inFlightWindow):
			}

			h.finish()

			if got, want := receive(t, response, "the response"), "200 done"; got != want {
				t.Errorf("expected the request in flight to be answered with %q but got %q", want, got)
			}
			if err := receive(t, interrupted, "the shutdown on interrupt to return"); err != nil {
				t.Errorf("the shutdown on interrupt failed: %v", err)
			}
			if err := receive(t, served, "ListenAndServe() to return"); err != nil {
				t.Errorf("ListenAndServe() failed: %v", err)
			}
		})
	}

	// A positive value is still the most the drain may take.
	t.Run("positive", func(t *testing.T) {
		h := newHeldHandler(t) // never finishes on its own.
		s := newShutdownTestServer(ServerOptions{Timeout: &TimeoutOptions{ShutdownOnInterruptTimeout: 100 * time.Millisecond}})
		addr, served := serveInBackground(t, s, h)
		response := getInBackground(newShutdownTestClient(t), "http://"+addr)
		receive(t, h.entered, "the request to reach the handler")

		start := time.Now()
		if err := receive(t, inBackground(s.shutdownOnInterrupt), "the shutdown on interrupt to return"); err != nil {
			t.Errorf("the shutdown on interrupt failed: %v", err)
		}
		if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
			t.Errorf("the shutdown on interrupt returned after %v, before its 100ms timeout", elapsed)
		}
		if err := receive(t, served, "ListenAndServe() to return"); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
		if got := receive(t, response, "the response"); !strings.HasPrefix(got, "error: ") {
			t.Errorf("expected the request that outlived the timeout to be cut but got %q", got)
		}
	})
}

// TestListenAndServeReturnsAfterShutdownOnInterrupt tests that ListenAndServe
// waits for the shutdown the interrupt signal runs as it does for a Shutdown call.
func TestListenAndServeReturnsAfterShutdownOnInterrupt(t *testing.T) {
	h := newHeldHandler(t)
	s := newShutdownTestServer(ServerOptions{}) // the default timeout, 15 seconds.
	addr, served := serveInBackground(t, s, h)
	response := getInBackground(newShutdownTestClient(t), "http://"+addr)

	expectServeToOutlastShutdown(t, h, served, response, s.shutdownOnInterrupt)
}

// TestShutdownFromHandler tests the two ways a handler can shut down the server
// that runs it without waiting for itself forever. A handler that calls Shutdown
// in place with a context that has no deadline is not one of them:
// http.Server.Shutdown waits for the connection of that very handler. With
// context.Background() it returns only when something else closes the server,
// and with the context of its own request when the client gives up.
// ListenAndServe waits as long as it does.
func TestShutdownFromHandler(t *testing.T) {
	t.Run("in a goroutine", func(t *testing.T) {
		s := newShutdownTestServer(ServerOptions{})
		shutdownErr := make(chan error, 1)
		addr, served := serveInBackground(t, s, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			go func() { shutdownErr <- s.Shutdown(context.Background()) }()
			io.WriteString(w, "bye")
		}))
		response := getInBackground(newShutdownTestClient(t), "http://"+addr)

		if got, want := receive(t, response, "the response"), "200 bye"; got != want {
			t.Errorf("expected the response %q but got %q", want, got)
		}
		if err := receive(t, shutdownErr, "Shutdown() to return"); err != nil {
			t.Errorf("Shutdown() failed: %v", err)
		}
		if err := receive(t, served, "ListenAndServe() to return"); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	})

	t.Run("in place with a deadline", func(t *testing.T) {
		s := newShutdownTestServer(ServerOptions{})
		shutdownErr := make(chan error, 1)
		addr, served := serveInBackground(t, s, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			shutdownErr <- s.Shutdown(ctx)
		}))
		getInBackground(newShutdownTestClient(t), "http://"+addr) // its connection is closed at the deadline.

		if err := receive(t, shutdownErr, "Shutdown() to return"); err != nil {
			t.Errorf("Shutdown() failed: %v", err)
		}
		if err := receive(t, served, "ListenAndServe() to return"); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	})
}

// TestShutdownTwice tests that ListenAndServe returns when two Shutdown calls run
// at the same time, and that a Shutdown or a Close after the server stopped
// returns at once.
func TestShutdownTwice(t *testing.T) {
	h := newHeldHandler(t)
	s := newShutdownTestServer(ServerOptions{})
	addr, served := serveInBackground(t, s, h)
	response := getInBackground(newShutdownTestClient(t), "http://"+addr)
	receive(t, h.entered, "the request to reach the handler")

	shutdown := func() error { return s.Shutdown(context.Background()) }
	first, second := inBackground(shutdown), inBackground(shutdown)
	time.Sleep(50 * time.Millisecond) // let both begin.
	h.finish()

	if got, want := receive(t, response, "the response"), "200 done"; got != want {
		t.Errorf("expected the request in flight to be answered with %q but got %q", want, got)
	}
	if err := receive(t, first, "the first Shutdown() to return"); err != nil {
		t.Errorf("the first Shutdown() failed: %v", err)
	}
	if err := receive(t, second, "the second Shutdown() to return"); err != nil {
		t.Errorf("the second Shutdown() failed: %v", err)
	}
	if err := receive(t, served, "ListenAndServe() to return"); err != nil {
		t.Errorf("ListenAndServe() failed: %v", err)
	}

	if err := receive(t, inBackground(shutdown), "Shutdown() of a stopped server to return"); err != nil {
		t.Errorf("Shutdown() of a stopped server failed: %v", err)
	}
	if err := receive(t, inBackground(s.Close), "Close() of a stopped server to return"); err != nil {
		t.Errorf("Close() of a stopped server failed: %v", err)
	}
}

// TestCloseEndsShutdownInProgress tests that Close still cuts a graceful shutdown
// short while ListenAndServe is waiting for it: the request never ends on its own,
// so nothing but Close can end the three of them.
func TestCloseEndsShutdownInProgress(t *testing.T) {
	h := newHeldHandler(t) // never finishes on its own.
	s := newShutdownTestServer(ServerOptions{})
	addr, served := serveInBackground(t, s, h)
	response := getInBackground(newShutdownTestClient(t), "http://"+addr)
	receive(t, h.entered, "the request to reach the handler")

	shutdownErr := inBackground(func() error { return s.Shutdown(context.Background()) })
	time.Sleep(50 * time.Millisecond) // let the shutdown begin and ListenAndServe wait for it.

	if err := receive(t, inBackground(s.Close), "Close() to return"); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
	if err := receive(t, shutdownErr, "Shutdown() to return"); err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
	if err := receive(t, served, "ListenAndServe() to return"); err != nil {
		t.Errorf("ListenAndServe() failed: %v", err)
	}
	if got := receive(t, response, "the response"); !strings.HasPrefix(got, "error: ") {
		t.Errorf("expected the request to be cut by Close() but got %q", got)
	}
}

// TestShutdownBeforeServerStarted tests that Shutdown and Close return at once on
// a server that never started, and that a server which starts after them still
// waits for the shutdown that stops it.
func TestShutdownBeforeServerStarted(t *testing.T) {
	s := newShutdownTestServer(ServerOptions{})

	if err := receive(t, inBackground(func() error { return s.Shutdown(context.Background()) }), "Shutdown() to return"); err != nil {
		t.Errorf("Shutdown() before the server started failed: %v", err)
	}
	if err := receive(t, inBackground(s.Close), "Close() to return"); err != nil {
		t.Errorf("Close() before the server started failed: %v", err)
	}

	h := newHeldHandler(t)
	addr, served := serveInBackground(t, s, h)
	response := getInBackground(newShutdownTestClient(t), "http://"+addr)

	expectServeToOutlastShutdown(t, h, served, response, func() error {
		return s.Shutdown(context.Background())
	})
}

// TestListenAndServeDoesNotWaitWithoutShutdown tests the ways a server stops that
// are not a Shutdown or a Close call of the Server: there is no shutdown to wait
// for, so ListenAndServe returns at once, and with the error when there is one.
func TestListenAndServeDoesNotWaitWithoutShutdown(t *testing.T) {
	t.Run("the address is in use", func(t *testing.T) {
		busy, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		defer busy.Close()

		s := NewServer(ServerOptions{Addr: busy.Addr().String(), DisableStartupLog: true, DisableInterruptHandler: true})
		served := inBackground(func() error { return s.ListenAndServe(http.NotFoundHandler()) })

		if err := receive(t, served, "ListenAndServe() to return"); err == nil {
			t.Error("expected ListenAndServe() to fail on an address that is in use")
		}
	})

	t.Run("OnServerCreated fails", func(t *testing.T) {
		errCreated := errors.New("not this server")
		s := newShutdownTestServer(ServerOptions{OnServerCreated: func(*http.Server) error { return errCreated }})
		served := inBackground(func() error { return s.ListenAndServe(http.NotFoundHandler()) })

		if err := receive(t, served, "ListenAndServe() to return"); !errors.Is(err, errCreated) {
			t.Errorf("expected ListenAndServe() to return the error of OnServerCreated but got %v", err)
		}
	})

	t.Run("the listener breaks", func(t *testing.T) {
		s := newShutdownTestServer(ServerOptions{})
		listener := make(chan net.Listener, 1)
		s.OnListenerCreated(func(ln net.Listener, _ *url.URL) error {
			listener <- ln
			return nil
		})
		served := inBackground(func() error { return s.ListenAndServe(http.NotFoundHandler()) })

		receive(t, listener, "the listener").Close()

		err := receive(t, served, "ListenAndServe() to return")
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			t.Errorf("expected ListenAndServe() to return the error of the listener but got %v", err)
		}
	})

	// OnServerCreated hands out the http.Server, so it can be shut down without
	// the Server knowing: Serve stops with http.ErrServerClosed all the same.
	t.Run("the http.Server is shut down directly", func(t *testing.T) {
		created := make(chan *http.Server, 1)
		s := newShutdownTestServer(ServerOptions{OnServerCreated: func(httpServer *http.Server) error {
			created <- httpServer
			return nil
		}})
		served := inBackground(func() error { return s.ListenAndServe(http.NotFoundHandler()) })

		httpServer := receive(t, created, "the http.Server")
		if err := receive(t, inBackground(func() error { return httpServer.Shutdown(context.Background()) }), "http.Server.Shutdown() to return"); err != nil {
			t.Errorf("http.Server.Shutdown() failed: %v", err)
		}
		if err := receive(t, served, "ListenAndServe() to return"); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	})
}
