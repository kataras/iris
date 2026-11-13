package iris_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
)

// Shutdown runs the closers in order and the server's own shutdown is only the
// first of them: Run and Listen return once the whole list has finished, so a
// main that ends with them does not cut a closer that is still running.

// freeAddr returns an address nothing listens on. Run takes an address, not a
// listener, so the port is picked by the system and given back.
func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// runInBackground starts the application on its own goroutine and returns once
// the server has answered a request. The channel carries what Run returned.
//
// OnServerCreated fires before the server keeps its http.Server and before it
// serves. A Shutdown called at that point reads the field Run is still writing,
// which go test -race reports, and it can find no server to stop. One answered
// request puts the caller after both.
func runInBackground(t *testing.T, app *iris.Application, addr string) <-chan error {
	t.Helper()

	listening, ran := make(chan struct{}), make(chan error, 1)
	go func() {
		ran <- app.Run(iris.ServerOptions{
			Addr:                    addr,
			DisableStartupLog:       true,
			DisableInterruptHandler: true,
			LogWriter:               io.Discard,
			OnServerCreated: func(*http.Server) error {
				close(listening)
				return nil
			},
		})
	}()

	select {
	case <-listening:
	case err := <-ran:
		t.Fatalf("Run returned before the server listened: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not start listening")
	}

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("the server did not answer its first request: %v", err)
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck // only the round trip matters.
	resp.Body.Close()

	return ran
}

// expectRun fails the test unless Run has returned nil by now.
func expectRun(t *testing.T, ran <-chan error, what string) {
	t.Helper()

	select {
	case err := <-ran:
		if err != nil {
			t.Fatalf("expected Run to return nil %s, got %v", what, err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("Run did not return %s", what)
	}
}

// newQuietApp returns an application that logs nothing and hands a failed Run
// back to the caller instead of exiting the process.
func newQuietApp() *iris.Application {
	app := iris.New(iris.Options{LogLevel: "disable", DisableFatalRun: true})
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	return app
}

// The closer that follows the server in the list is still running when the
// server's shutdown has returned. Run waits for it.
func TestRunWaitsForTheClosersOfShutdown(t *testing.T) {
	app := newQuietApp()

	var closerDone atomic.Bool
	app.AddCloser(func() error {
		time.Sleep(300 * time.Millisecond)
		closerDone.Store(true)
		return nil
	})

	ran := runInBackground(t, app, freeAddr(t))

	shutdown := make(chan error, 1)
	go func() { shutdown <- app.Shutdown(context.Background()) }()

	expectRun(t, ran, "after the shutdown")
	if !closerDone.Load() {
		t.Error("Run returned while a closer was still running")
	}

	select {
	case err := <-shutdown:
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown did not return")
	}
}

// Two Shutdown calls at once: Run waits for the last one to finish.
func TestRunWaitsForEveryShutdownCall(t *testing.T) {
	app := newQuietApp()

	var running, peak atomic.Int32
	var closerDone atomic.Int32
	app.AddCloser(func() error {
		if now := running.Add(1); now > peak.Load() {
			peak.Store(now)
		}
		time.Sleep(200 * time.Millisecond)
		running.Add(-1)
		closerDone.Add(1)
		return nil
	})

	ran := runInBackground(t, app, freeAddr(t))

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if err := app.Shutdown(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}

	expectRun(t, ran, "after the shutdown")
	if got := closerDone.Load(); got != 2 {
		t.Errorf("Run returned after %d of the 2 closer calls", got)
	}

	wg.Wait()
	if peak.Load() < 1 {
		t.Error("the closer never ran")
	}
}

// A closer that calls Shutdown itself does not deadlock. The second call runs
// the list again, so the closer guards itself with a sync.Once.
func TestRunWaitsForAShutdownStartedByACloser(t *testing.T) {
	app := newQuietApp()

	var (
		started atomic.Bool
		calls   atomic.Int32
		inner   = make(chan error, 1)
	)
	app.AddCloser(func() error {
		calls.Add(1)
		// Only the first run starts another shutdown: that one runs the whole
		// list again, this closer included.
		if started.CompareAndSwap(false, true) {
			inner <- app.Shutdown(context.Background())
		}
		return nil
	})

	ran := runInBackground(t, app, freeAddr(t))

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	expectRun(t, ran, "after the nested shutdown")
	select {
	case err := <-inner:
		if err != nil {
			t.Errorf("the shutdown the closer started failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the shutdown the closer started did not return")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected the closer to run twice (once per Shutdown), got %d", got)
	}
}

// A handler that stops its own application on another goroutine: Run returns
// once the request is answered and the closers are done.
func TestRunWaitsForTheClosersOfAShutdownFromAHandler(t *testing.T) {
	app := newQuietApp()

	var closerDone atomic.Bool
	app.AddCloser(func() error {
		time.Sleep(200 * time.Millisecond)
		closerDone.Store(true)
		return nil
	})

	answered := make(chan string, 1)
	app.Get("/stop", func(ctx iris.Context) {
		go app.Shutdown(context.Background()) //nolint:errcheck // the test reads the result through Run.
		ctx.WriteString("stopping")
	})

	addr := freeAddr(t)
	ran := runInBackground(t, app, addr)

	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		defer client.CloseIdleConnections()

		resp, err := client.Get("http://" + addr + "/stop")
		if err != nil {
			answered <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		answered <- resp.Status + " " + string(body)
	}()

	expectRun(t, ran, "after the shutdown the handler started")
	if !closerDone.Load() {
		t.Error("Run returned while a closer was still running")
	}

	select {
	case got := <-answered:
		if want := "200 OK stopping"; got != want {
			t.Errorf("expected the request to be answered with %q, got %q", want, got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the request was not answered")
	}
}

// Shutdown before Run runs the closers registered by then, and the Run after
// it still serves and still returns.
func TestShutdownBeforeRun(t *testing.T) {
	app := newQuietApp()

	var calls atomic.Int32
	app.AddCloser(func() error {
		calls.Add(1)
		return nil
	})

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected the closer to run once before Run, got %d", got)
	}

	ran := runInBackground(t, app, freeAddr(t))
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	expectRun(t, ran, "after the second shutdown")

	if got := calls.Load(); got != 2 {
		t.Errorf("expected the closer to run twice, got %d", got)
	}
}
