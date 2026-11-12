package errors_test

import (
	stdcontext "context"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestCancelledErrorCode(t *testing.T) {
	app := iris.New()
	app.Get("/cancelled", func(ctx iris.Context) {
		errors.Cancelled.Message(ctx, "the client stopped waiting")
	})

	e := httptest.New(t, app)
	e.GET("/cancelled").Expect().Status(httptest.StatusRequestTimeout).
		JSON().Object().Value("http_error_code").Object().HasValue("canonical_name", "CANCELLED")
}

// runLogErr runs one request through "call" and records what the logger saw and
// what the client got. It mirrors runReport in stream_test.go: the client's own
// request context is cancelled in-handler when cancelClient is set, while the
// connection itself stays alive so the response can still be observed.
func runLogErr(t *testing.T, cancelClient bool, call func(iris.Context)) (logged error, status int) {
	t.Helper()

	original := errors.LogError
	t.Cleanup(func() { errors.LogError = original })
	errors.LogError = func(_ iris.Context, err error) { logged = err }

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if cancelClient {
			cancelled, cancel := stdcontext.WithCancel(ctx.Request().Context())
			cancel()
			ctx.SetRequest(ctx.Request().WithContext(cancelled))
		}

		call(ctx)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)

	resp, getErr := http.Get(srv.URL)
	if getErr != nil {
		t.Fatal(getErr)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return logged, resp.StatusCode
}

// LogErr follows Report's policy: only the client's own cancellation silences
// it. An upstream dial timeout is a net.Error whose Timeout() is true, so it
// used to be swallowed - the one failure an operator most needs to see.
func TestLogErrTimeoutWithALiveClientIsLogged(t *testing.T) {
	logged, status := runLogErr(t, false, func(ctx iris.Context) {
		errors.Internal.LogErr(ctx, timeoutErr)
	})

	if logged == nil {
		t.Error("the logger never saw an upstream timeout while the client was still connected")
	}
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestLogErrSkipsACancelledClient(t *testing.T) {
	logged, status := runLogErr(t, true, func(ctx iris.Context) {
		errors.Internal.LogErr(ctx, timeoutErr)
	})

	if logged != nil {
		t.Errorf("the logger fired for a client that is gone: %v", logged)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d: nothing should have been written", status, http.StatusOK)
	}
}

// Log takes the same route through its formatted arguments: a timeout passed as
// an arg is not the client hanging up.
func TestLogTimeoutArgumentWithALiveClientIsLogged(t *testing.T) {
	logged, status := runLogErr(t, false, func(ctx iris.Context) {
		errors.Internal.Log(ctx, "relaying to the model failed: %w", timeoutErr)
	})

	if logged == nil {
		t.Error("the logger never saw a timeout passed as a format argument")
	}
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestLogSkipsACancelledClient(t *testing.T) {
	logged, status := runLogErr(t, true, func(ctx iris.Context) {
		errors.Internal.Log(ctx, "relaying to the model failed: %w", timeoutErr)
	})

	if logged != nil {
		t.Errorf("the logger fired for a client that is gone: %v", logged)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d: nothing should have been written", status, http.StatusOK)
	}
}
