package recorder_test

import (
	"fmt"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
	"github.com/kataras/iris/v14/middleware/recorder"
	"github.com/kataras/iris/v14/middleware/servertiming"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/rest/errors"
)

// recorderUsers returns every built-in middleware that puts a pooled
// ResponseRecorder on the Context and releases it when it returns.
func recorderUsers(t *testing.T) map[string]iris.Handler {
	t.Helper()

	ac := accesslog.New(io.Discard)
	ac.ResponseBody = true // this is what makes the access log record the response.
	t.Cleanup(func() { ac.Close() })

	type userSession struct{ Username string }
	sessionDB := sessions.NewCookieDatabase[userSession](sessions.NewOptions(
		[]byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"),
		[]byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")))

	// A Cache without a store: every Get is a miss and every Set is a no-op, so
	// the handler runs its recording path without a live Redis.
	cache := httpcache.Must(httpcache.Options{Disable: true})
	cache.Options.Disable = false

	return map[string]iris.Handler{
		"recorder.Response":    recorder.Response,
		"accesslog.Handler":    ac.Handler,
		"servertiming.Handler": servertiming.Handler,
		"sessions.Handler":     sessions.Handler(sessionDB, "session"),
		"httpcache.Handler":    cache.Handler(),
	}
}

// A middleware that swaps the Context's response writer for a recorder gives
// the recorder back to its pool when it returns. The writer that was on the
// Context before it has to be back by then: a handler registered outside it
// (an access log, an error handler, a metrics middleware) reads the Context's
// writer after the chain returns, and a pooled recorder may already belong to
// another request.
func TestRecorderUsersRestoreTheContextWriter(t *testing.T) {
	for name, user := range recorderUsers(t) {
		t.Run(name, func(t *testing.T) {
			var leftBehind string

			app := iris.New()
			app.UseRouter(func(ctx iris.Context) {
				before := ctx.ResponseWriter()
				ctx.Next()
				if after := ctx.ResponseWriter(); after != before {
					leftBehind = fmt.Sprintf("%T", after)
				}
			})
			app.Get("/", user, func(ctx iris.Context) {
				ctx.WriteString("ok")
			})

			e := httptest.New(t, app)
			e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

			if leftBehind != "" {
				t.Fatalf("%s left a released %s on the Context", name, leftBehind)
			}
		})
	}
}

// What the leftover costs a client: an error handler registered outside the
// recorder user writes its body after the chain returns, into a recorder that
// was already flushed and released, so the body never leaves the server.
func TestOuterHandlerWritesAfterARecorderUser(t *testing.T) {
	for name, user := range recorderUsers(t) {
		t.Run(name, func(t *testing.T) {
			app := iris.New()
			app.UseRouter(errors.Handler)
			app.Get("/", user, func(ctx iris.Context) {
				ctx.StopWithStatus(iris.StatusForbidden)
			})

			e := httptest.New(t, app)
			e.GET("/").Expect().Status(httptest.StatusForbidden).
				JSON().Object().Value("http_error_code").Object().
				HasValue("canonical_name", "PERMISSION_DENIED").
				HasValue("status", httptest.StatusForbidden)
		})
	}
}

// A response that is a status and headers and nothing else still reaches the
// client through a recorder. Releasing a recorder with an empty body hands the
// status and the headers to the writer under it and writes nothing; the router
// sends them when the request ends. Checked over a real connection, because
// that last step is the server's.
func TestRecorderUsersForwardAStatusOnlyResponse(t *testing.T) {
	for name, user := range recorderUsers(t) {
		t.Run(name, func(t *testing.T) {
			app := iris.New()
			app.Get("/", user, func(ctx iris.Context) {
				ctx.SetHeader("X-Plan", "starter")
				ctx.StatusCode(iris.StatusNoContent)
			})

			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			srv := nethttptest.NewServer(app)
			defer srv.Close()

			res, err := srv.Client().Get(srv.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if res.StatusCode != http.StatusNoContent {
				t.Fatalf("expected the 204 the handler set, got %d", res.StatusCode)
			}

			if got := res.Header.Get("X-Plan"); got != "starter" {
				t.Fatalf("expected the header the handler set, got %q", got)
			}

			if len(body) != 0 {
				t.Fatalf("expected no body, got %q", body)
			}
		})
	}
}

// Written is how a handler asks whether the response is still untouched. A
// recorder that holds nothing has to answer for the writer under it: nothing
// written there either, or a status line that already went out. Answering 0
// for both made "the headers are on the wire" look like "nothing happened".
func TestRecorderWrittenReportsTheWriterUnderneathWhenEmpty(t *testing.T) {
	var untouched, recorded, flushedUnderneath int

	app := iris.New()
	app.Get("/", recorder.Response, func(ctx iris.Context) {
		untouched = ctx.ResponseWriter().Written()

		ctx.WriteString("hello")
		recorded = ctx.ResponseWriter().Written()
	})
	app.Get("/flushed", func(ctx iris.Context) {
		ctx.ResponseWriter().Flush() // the status line goes out before the recorder exists.
		ctx.Next()
	}, recorder.Response, func(ctx iris.Context) {
		flushedUnderneath = ctx.ResponseWriter().Written()
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("hello")
	e.GET("/flushed").Expect().Status(httptest.StatusOK)

	if untouched != context.NoWritten {
		t.Fatalf("expected NoWritten (%d) from an empty recorder over an untouched writer, got %d", context.NoWritten, untouched)
	}

	if recorded != len("hello") {
		t.Fatalf("expected the %d recorded bytes, got %d", len("hello"), recorded)
	}

	if flushedUnderneath != context.StatusCodeWritten {
		t.Fatalf("expected StatusCodeWritten (%d) from an empty recorder over a flushed writer, got %d",
			context.StatusCodeWritten, flushedUnderneath)
	}
}
