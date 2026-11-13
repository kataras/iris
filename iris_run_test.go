package iris_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
)

var errBuildBoom = errors.New("boom")

// With DisableFatalRun set, a failed build is logged at error level and the
// error comes back to the caller, so a process that runs more than one
// server, or a test, decides for itself what to do with it.
func TestRunReturnsBuildErrorWhenFatalDisabled(t *testing.T) {
	app := iris.New(iris.Options{DisableFatalRun: true})

	var logs bytes.Buffer
	app.Logger().SetOutput(&logs)
	app.OnBuild = func() error { return errBuildBoom }

	err := app.Listen(":0")
	if !errors.Is(err, errBuildBoom) {
		t.Fatalf("expected the build error back, got %v", err)
	}

	if !strings.Contains(logs.String(), errBuildBoom.Error()) {
		t.Fatalf("expected the build error to be logged, got %q", logs.String())
	}
}

// By default a failed build is logged at fatal level and the process exits
// with status 1, which is what makes a bare app.Listen(":8080") enough for a
// program whose only job is to serve. The behaviour is observed from a child
// process, since it ends the one that triggers it.
func TestRunExitsOnBuildError(t *testing.T) {
	if os.Getenv("IRIS_TEST_RUN_EXIT") == "1" {
		app := iris.New()
		app.OnBuild = func() error { return errBuildBoom }
		app.Listen(":0")
		os.Exit(0) // not reached: Listen exits with status 1.
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestRunExitsOnBuildError$")
	cmd.Env = append(os.Environ(), "IRIS_TEST_RUN_EXIT=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected the child to exit with status 1, got err=%v output=%q", err, out)
	}

	if code := exitErr.ExitCode(); code != 1 {
		t.Fatalf("expected exit status 1, got %d (output %q)", code, out)
	}

	if !strings.Contains(string(out), errBuildBoom.Error()) {
		t.Fatalf("expected the build error in the child's output, got %q", out)
	}
}

// Run returns when the shutdown has ended, not when it begins. It used to
// return the moment Shutdown was called, so a main whose last call is Run or
// Listen exited while the server was still draining and the requests in flight
// were reset. The server runs on a real loopback listener, and the request is
// held in its handler for as long as the test needs.
func TestRunReturnsAfterShutdown(t *testing.T) {
	// A port the system picks: Run takes an address, not a listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	app := iris.New(iris.Options{LogLevel: "disable", DisableFatalRun: true})

	var (
		entered  = make(chan struct{})
		release  = make(chan struct{})
		answered atomic.Bool
	)
	app.Get("/slow", func(ctx iris.Context) {
		close(entered)
		<-release
		ctx.WriteString("done")
		answered.Store(true)
	})

	var (
		listening = make(chan struct{})
		ran       = make(chan error, 1)
		wait      = time.After(10 * time.Second)
	)
	go func() {
		ran <- app.Run(iris.ServerOptions{
			Addr:                    addr,
			DisableStartupLog:       true,
			DisableInterruptHandler: true,
			LogWriter:               io.Discard,
			// Called once the listener is open, right before it serves.
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
	case <-wait:
		t.Fatal("the server did not start listening")
	}

	response := make(chan string, 1)
	go func() {
		client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{}}
		defer client.CloseIdleConnections()

		resp, err := client.Get("http://" + addr + "/slow")
		if err != nil {
			response <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		response <- resp.Status + " " + string(body)
	}()

	select {
	case <-entered:
	case got := <-response:
		t.Fatalf("the request did not reach its handler: %s", got)
	case <-wait:
		t.Fatal("the request did not reach its handler")
	}

	shutdown := make(chan error, 1)
	go func() { shutdown <- app.Shutdown(context.Background()) }()

	select {
	case err := <-ran:
		t.Fatalf("Run returned %v with a request still in flight", err)
	case <-time.After(250 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-ran:
		if err != nil {
			t.Errorf("expected Run to return nil after a graceful shutdown, got %v", err)
		}
	case <-wait:
		t.Fatal("Run did not return after the shutdown")
	}

	if !answered.Load() {
		t.Error("Run returned before the request in flight was answered")
	}

	select {
	case err := <-shutdown:
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	case <-wait:
		t.Fatal("Shutdown did not return")
	}

	select {
	case got := <-response:
		if want := "200 OK done"; got != want {
			t.Errorf("expected the request in flight to be answered with %q, got %q", want, got)
		}
	case <-wait:
		t.Fatal("the request in flight was not answered")
	}
}
