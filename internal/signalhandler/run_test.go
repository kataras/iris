package signalhandler_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/internal/signalhandler"
)

// What iris.Application.Run does on an interrupt signal: it does not return
// before the callbacks of the signal are done, those callbacks receive the
// Timeout.ShutdownOnInterruptTimeout of the server as their deadline, and a
// failed start is reported without waiting for them.
//
// The tests are here and not beside Run because only this package can run the
// path of a signal without sending one: OnSignal is what the signal's goroutine
// calls, and NewProcess replaces the exit of the process.

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

// newApp returns an application that logs nothing and hands a failed Run back
// to the caller instead of exiting the process.
func newApp() *iris.Application {
	app := iris.New(iris.Options{LogLevel: "disable", DisableFatalRun: true})
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	return app
}

// run starts the application on its own goroutine, on an address of its own,
// and returns once the server has answered a request: it has registered its
// handler with the signal by then, which it does right before it serves. The
// channel carries what Run returned.
func run(t *testing.T, app *iris.Application, options iris.ServerOptions) <-chan error {
	t.Helper()

	addr := freeAddr(t)
	listening, ran := make(chan struct{}), make(chan error, 1)

	options.Addr = addr
	options.DisableStartupLog = true
	options.LogWriter = io.Discard
	options.OnServerCreated = func(*http.Server) error {
		close(listening)
		return nil
	}

	go func() { ran <- app.Run(options) }()

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
func expectRun(t *testing.T, ran <-chan error) {
	t.Helper()

	select {
	case err := <-ran:
		if err != nil {
			t.Fatalf("expected Run to return nil, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
	}
}

// expectExit fails the test unless the process was told to exit with code.
func expectExit(t *testing.T, exited <-chan int, code int) {
	t.Helper()

	select {
	case got := <-exited:
		if got != code {
			t.Errorf("expected the process to exit with %d, got %d", code, got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the process was not told to exit")
	}
}

// The server's own handler is the first callback of the signal here, so the
// server has drained and ListenAndServe has returned while the callback after
// it is still running. Run waits for it: the process exits when the callbacks
// are done, and a main that got Run back before that would end it first.
func TestRunWaitsForTheCallbacksOfASignal(t *testing.T) {
	exited := signalhandler.NewProcess()

	ran := run(t, newApp(), iris.ServerOptions{})

	// Registered after the server's own handler, so it runs after the drain.
	var done atomic.Bool
	iris.RegisterOnInterrupt(func() {
		time.Sleep(300 * time.Millisecond)
		done.Store(true)
	})

	go signalhandler.OnSignal()

	expectRun(t, ran)
	if !done.Load() {
		t.Error("Run returned while a callback of the signal was still running")
	}
	expectExit(t, exited, 0)
}

// A signal that arrives while the closers of a Shutdown are running: Run waits
// for the closers and then for the callbacks of the signal. In the other order
// it looks for the signal before there is one, and returns when the closers are
// done with the callbacks still running.
func TestRunWaitsForASignalThatArrivesDuringTheClosers(t *testing.T) {
	exited := signalhandler.NewProcess()

	app := newApp()
	closerStarted, release := make(chan struct{}), make(chan struct{})
	app.AddCloser(func() {
		close(closerStarted)
		<-release
	})

	ran := run(t, app, iris.ServerOptions{})

	callbackStarted := make(chan struct{})
	var done atomic.Bool
	iris.RegisterOnInterrupt(func() {
		close(callbackStarted)
		time.Sleep(300 * time.Millisecond)
		done.Store(true)
	})

	shutdown := make(chan error, 1)
	go func() { shutdown <- app.Shutdown(context.Background()) }()

	select {
	case <-closerStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("the closer did not run")
	}
	// The server has shut down and the closer holds the Shutdown call: give Run
	// the time to get from the server to its wait for that call.
	time.Sleep(50 * time.Millisecond)

	go signalhandler.OnSignal()

	select {
	case <-callbackStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("the callback of the signal did not run")
	}
	close(release)

	expectRun(t, ran)
	if !done.Load() {
		t.Error("Run returned while a callback of the signal was still running")
	}
	expectExit(t, exited, 0)

	select {
	case err := <-shutdown:
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown did not return")
	}
}

// The server hands its Timeout.ShutdownOnInterruptTimeout to the signal
// handler, and a callback of iris.RegisterOnInterrupt receives it as the
// deadline of its context.
func TestInterruptCallbackGetsTheTimeoutOfTheServer(t *testing.T) {
	tests := []struct {
		name    string
		options iris.ServerOptions
		want    time.Duration // zero: no deadline.
	}{
		{"default", iris.ServerOptions{}, 15 * time.Second},
		{"set", iris.ServerOptions{Timeout: iris.Timeout().WithShutdownOnInterruptTimeout(40 * time.Second)}, 40 * time.Second},
		{"zero", iris.ServerOptions{Timeout: iris.Timeout().WithShutdownOnInterruptTimeout(0)}, 0},
		// A literal carries zero, where iris.Timeout() starts at 15 seconds.
		{"literal", iris.ServerOptions{Timeout: &iris.TimeoutOptions{}}, 0},
		// The server keeps no handler, so it hands its option to nobody.
		{"no interrupt handler", iris.ServerOptions{DisableInterruptHandler: true}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exited := signalhandler.NewProcess()

			app := newApp()
			ran := run(t, app, tt.options)

			left := make(chan time.Duration, 1) // zero: no deadline.
			iris.RegisterOnInterrupt(func(ctx context.Context) {
				deadline, ok := ctx.Deadline()
				if !ok {
					left <- 0
					return
				}
				left <- time.Until(deadline)
			})

			go signalhandler.OnSignal()
			expectExit(t, exited, 0)

			if tt.options.DisableInterruptHandler {
				// The signal did not stop this server.
				if err := app.Shutdown(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			expectRun(t, ran)

			got := <-left
			switch {
			case tt.want == 0 && got != 0:
				t.Errorf("expected a context with no deadline, got one %s away", got)
			case tt.want != 0 && (got > tt.want || got < tt.want-10*time.Second):
				// Counted from the signal, so a little less than the option.
				t.Errorf("expected a context with a deadline %s away, got %s (zero: none)", tt.want, got)
			}
		})
	}
}

// A Run that fails to start while the callbacks of a signal are running reports
// its error at once. Waiting for them first loses it: that wait ends with the
// process, which the signal exits with status 0.
func TestRunReportsAFailedStartDuringASignal(t *testing.T) {
	exited := signalhandler.NewProcess()

	callbackStarted, release := make(chan struct{}), make(chan struct{})
	iris.RegisterOnInterrupt(func() {
		close(callbackStarted)
		<-release
	})

	go signalhandler.OnSignal()

	select {
	case <-callbackStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("the callback of the signal did not run")
	}

	// The address is taken, so the listener of Run cannot be opened.
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	ran := make(chan error, 1)
	go func() {
		ran <- newApp().Run(iris.ServerOptions{
			Addr:              busy.Addr().String(),
			DisableStartupLog: true,
			LogWriter:         io.Discard,
		})
	}()

	reported := false
	select {
	case err := <-ran:
		reported = true
		if err == nil {
			t.Error("expected Run to return the error of its listener, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Error("Run did not report its failed start while the callbacks of the signal were running")
	}

	close(release)
	expectExit(t, exited, 0)

	if !reported {
		// Let that Run go before the next test replaces the state it reads.
		select {
		case <-ran:
		case <-time.After(10 * time.Second):
			t.Fatal("Run did not return after the callbacks of the signal")
		}
	}
}
