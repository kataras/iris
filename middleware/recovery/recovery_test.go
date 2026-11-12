package recovery_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/recovery"
)

func TestRecoveryHandlesPanic(t *testing.T) {
	app := iris.New()
	app.Use(recovery.Handler)
	app.Get("/panic", func(ctx iris.Context) {
		panic("boom")
	})
	app.Get("/ok", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	e.GET("/panic").Expect().Status(httptest.StatusInternalServerError)
	// The server keeps serving after a panic.
	e.GET("/ok").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
}

// A 500 from a recovered panic must not disclose the panic value, the stack or
// any internal file path to the client.
func TestRecoveryDoesNotLeakInternalsToClient(t *testing.T) {
	app := iris.New()
	app.Use(recovery.Handler)
	app.Get("/panic", func(ctx iris.Context) {
		panic("SECRET_INTERNAL_DETAIL")
	})

	e := httptest.New(t, app)

	body := e.GET("/panic").Expect().Status(httptest.StatusInternalServerError).Body().Raw()

	for _, leak := range []string{"SECRET_INTERNAL_DETAIL", "goroutine", "recovery.go", "runtime."} {
		if strings.Contains(body, leak) {
			t.Errorf("the 500 response leaks %q:\n%s", leak, body)
		}
	}
}

// http.ErrAbortHandler is the standard library's deliberate "abandon this
// response" signal, not a bug to report. Recovering from it turned an intended
// abort into a logged stack trace and a 500.
func TestRecoveryRepanicsErrAbortHandler(t *testing.T) {
	app := iris.New()
	app.Use(recovery.Handler)
	app.Get("/abort", func(ctx iris.Context) {
		panic(http.ErrAbortHandler)
	})
	app.Get("/ok", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	// Calling ServeHTTP directly, because net/http would swallow the re-panic on
	// a real connection - which is exactly the behaviour we want to preserve.
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected http.ErrAbortHandler to propagate, it was recovered instead")
		}
		if rec != http.ErrAbortHandler {
			t.Fatalf("expected http.ErrAbortHandler, got %v", rec)
		}
	}()

	req := nethttptest.NewRequest(http.MethodGet, "/abort", nil)
	app.ServeHTTP(nethttptest.NewRecorder(), req)
}
