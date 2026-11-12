package accesslog_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
)

// File wraps the *os.File in a bufio.ReadWriter, which is a Flusher but not an
// io.Closer, so the closer registration used to see nothing at all: Close
// flushed the buffer and leaked the descriptor.
func TestFileClosesTheUnderlyingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")

	ac := accesslog.File(path)
	ac.Async = false

	app := iris.New()
	app.UseRouter(ac.Handler)
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK)

	if len(ac.Closers) == 0 {
		t.Fatal("no closer was registered for the file opened by accesslog.File")
	}

	if err := ac.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// The descriptor is gone: closing it a second time reports os.ErrClosed.
	// A leaked fd would close cleanly here instead.
	for _, closer := range ac.Closers {
		if err := closer.Close(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("second Close on the registered closer = %v, want os.ErrClosed (the file was never closed)", err)
		}
	}

	// The log line still reached the file, so this is a leak fix, not data loss.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("the log file is empty; Close must flush before it closes")
	}
}

// FileUnbuffered passes the *os.File straight through, so SetOutput registers it
// once. It must not be registered a second time by the File path's addCloser.
func TestFileUnbufferedClosesTheFileExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")

	ac := accesslog.FileUnbuffered(path)
	if got := len(ac.Closers); got != 1 {
		t.Fatalf("len(Closers) = %d, want exactly 1", got)
	}

	if err := ac.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// os.Stdout is an *os.File, so it used to be registered as a Closer and a
// graceful shutdown closed the process's standard output for every other writer
// in the program. The application builder's default access log does exactly
// AddOutput(os.Stdout), so this hit any app with LogRequests(true).
func TestStdStreamsAreNeverClosed(t *testing.T) {
	for name, w := range map[string]*os.File{"Stdout": os.Stdout, "Stderr": os.Stderr} {
		t.Run(name, func(t *testing.T) {
			ac := accesslog.New(io.Discard)
			ac.AddOutput(w)

			for _, closer := range ac.Closers {
				if closer == io.Closer(w) {
					t.Fatalf("os.%s was registered as a Closer", name)
				}
			}

			if err := ac.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			// Still usable: a write to the real descriptor succeeds.
			if _, err := w.WriteString(""); err != nil {
				t.Fatalf("os.%s is unusable after Close: %v", name, err)
			}
		})
	}
}

// SetOutput and AddOutput are variadic and inspect each writer they are given,
// which is what makes `SetOutput(os.Stdout, f)` register f as a closer while
// skipping os.Stdout. Pre-combining them into one io.MultiWriter hides f, and
// the README says so; this pins both halves of that claim.
func TestSetOutputRegistersEachWriterSeparately(t *testing.T) {
	newFile := func(t *testing.T) *os.File {
		t.Helper()
		f, err := os.OpenFile(filepath.Join(t.TempDir(), "access.log"),
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}

	t.Run("separate writers", func(t *testing.T) {
		f := newFile(t)

		ac := accesslog.New(io.Discard)
		ac.SetOutput(os.Stdout, f)

		if got := len(ac.Closers); got != 1 {
			t.Fatalf("len(Closers) = %d, want exactly 1 (the file, not os.Stdout)", got)
		}

		if err := ac.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		if err := f.Close(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("second Close on the file = %v, want os.ErrClosed (Close did not close it)", err)
		}
	})

	t.Run("pre-combined multiwriter", func(t *testing.T) {
		f := newFile(t)
		defer f.Close()

		ac := accesslog.New(io.Discard)
		ac.SetOutput(io.MultiWriter(os.Stdout, f))

		// One opaque writer: nothing inside it can be found.
		if got := len(ac.Closers); got != 0 {
			t.Fatalf("len(Closers) = %d, want 0; a MultiWriter hides its writers", got)
		}

		if err := ac.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
}

// SetOutput with a replacement destination flushes and closes the writers it is
// replacing, so switching outputs does not leak the old file.
func TestSetOutputClosesTheReplacedWriters(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first.log")

	ac := accesslog.File(first)

	ac.SetOutput(io.Discard)

	if got := len(ac.Closers); got != 0 {
		t.Fatalf("len(Closers) = %d after SetOutput, want 0 (io.Discard is not a Closer)", got)
	}

	// The replaced file is closed, not leaked: reopening the path and reading it
	// works, and a write through the old descriptor would fail. The descriptor
	// itself is unreachable from here, so assert through the OS instead: on
	// Windows an open handle keeps the file locked and this remove fails.
	if err := os.Remove(first); err != nil {
		t.Fatalf("the replaced log file could not be removed, its descriptor is still open: %v", err)
	}

	if err := ac.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
