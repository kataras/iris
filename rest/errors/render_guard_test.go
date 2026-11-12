package errors_test

import (
	"encoding/json"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/golog"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// Error() formats a description; it must not rewrite the receiver, which is
// the very value that renders on the wire right after being logged.
func TestErrorStringDoesNotMutate(t *testing.T) {
	e := &errors.Error{Message: "x"}
	if s := e.Error(); !strings.Contains(s, "message: x") {
		t.Fatalf("unexpected text: %s", s)
	}
	if e.Detail != "" || e.ErrorCode.CanonicalName != "" || e.ErrorCode.Status != 0 {
		t.Fatalf("Error() mutated the receiver: %+v", e)
	}
}

func TestLogErrOfErrorValueKeepsEmptyDetail(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	app.Get("/err", func(ctx iris.Context) {
		errors.Internal.LogErr(ctx, &errors.Error{Message: "x"})
	})

	e := httptest.New(t, app)
	e.GET("/err").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().NotContainsKey("detail")
}

func serve(t *testing.T, app *iris.Application) *nethttptest.Server {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)
	return srv
}

// An error after a partial body is logged but not rendered: the 200 and the
// body bytes are already on the wire.
func TestHandleErrorAfterPartialBody(t *testing.T) {
	app := iris.New()
	var logged []string
	app.Logger().Handle(func(l *golog.Log) bool {
		if l.Level <= golog.ErrorLevel {
			logged = append(logged, l.Message)
		}
		return true
	})

	var handled bool
	app.Get("/partial", func(ctx iris.Context) {
		ctx.WriteString("partial")
		handled = errors.HandleError(ctx, io.ErrClosedPipe)
	})

	srv := serve(t, app)
	resp, err := http.Get(srv.URL + "/partial")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if string(body) != "partial" {
		t.Fatalf("body = %q, want exactly the partial body", body)
	}
	if !handled {
		t.Fatal("HandleError must report the error as handled so the caller stops")
	}
	if len(logged) != 1 || !strings.Contains(logged[0], io.ErrClosedPipe.Error()) {
		t.Fatalf("the error must still be logged, got: %q", logged)
	}
}

// A truncated document is malformed JSON, not an empty body: the syntactic
// error carries io.ErrUnexpectedEOF and must be classified first.
func TestHandleJSONErrorTruncatedBody(t *testing.T) {
	app := iris.New()
	app.Post("/read", func(ctx iris.Context) {
		var v map[string]any
		errors.HandleRequestJSONError(ctx, ctx.ReadJSON(&v))
	})

	e := httptest.New(t, app)
	e.POST("/read").WithBytes([]byte(`{"a":`)).Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("detail", "malformed JSON")
}

// The encoding/json (v1) fallback branches never forward the error text,
// which names the server's Go types.
func TestHandleJSONErrorV1FallbackTexts(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	app.Get("/unsupported-type", func(ctx iris.Context) {
		errors.HandleResponseJSONError(ctx, &json.UnsupportedTypeError{Type: reflect.TypeOf(make(chan int))})
	})
	app.Get("/unsupported-value", func(ctx iris.Context) {
		errors.HandleResponseJSONError(ctx, &json.UnsupportedValueError{Str: "chan int"})
	})
	app.Get("/invalid-unmarshal", func(ctx iris.Context) {
		errors.HandleRequestJSONError(ctx, &json.InvalidUnmarshalError{Type: reflect.TypeOf(make(chan int))})
	})

	e := httptest.New(t, app)
	for _, path := range []string{"/unsupported-type", "/unsupported-value", "/invalid-unmarshal"} {
		body := e.GET(path).Expect().Body().Raw()
		if strings.Contains(body, "chan") {
			t.Fatalf("%s: the Go type reached the client: %s", path, body)
		}
	}
}

type fieldErr struct {
	Reason string `json:"reason"`
}

func (e fieldErr) Error() string     { return e.Reason }
func (e fieldErr) GetReason() string { return e.Reason }

// A validation error given to Err renders exactly one document.
func TestErrWithValidationErrorRendersOnce(t *testing.T) {
	app := iris.New()
	app.Get("/one", func(ctx iris.Context) {
		errors.InvalidArgument.Err(ctx, fieldErr{"must be positive"})
	})
	app.Get("/many", func(ctx iris.Context) {
		errors.InvalidArgument.Err(ctx, errors.ValidationErrors{fieldErr{"must be positive"}})
	})

	e := httptest.New(t, app)
	for _, path := range []string{"/one", "/many"} {
		body := e.GET(path).Expect().Status(httptest.StatusBadRequest).Body().Raw()
		var doc map[string]any
		if err := json.Unmarshal([]byte(body), &doc); err != nil {
			t.Fatalf("%s: body is not a single JSON document: %v\n%s", path, err, body)
		}
		if doc["validation"] == nil {
			t.Fatalf("%s: validation payload missing: %s", path, body)
		}
	}
}

func TestHandleRequestErrorNil(t *testing.T) {
	app := iris.New()
	app.Get("/nil", func(ctx iris.Context) {
		if errors.HandleRequestError(ctx, nil) {
			t.Error("a nil error must not be handled")
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
}

// An error implementing context.ErrPrivate is a server error: it renders as
// 500 with a generic message.
func TestHandleErrorPrivateIsServerError(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	app.Get("/private", func(ctx iris.Context) {
		errors.HandleError(ctx, iris.PrivateError(io.ErrClosedPipe))
	})

	e := httptest.New(t, app)
	e.GET("/private").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().NotHasValue("message", io.ErrClosedPipe.Error())
}

// The variadic args must not be appended in place: with spare capacity the
// caller's backing array would receive the error.
func TestHandleErrorfDoesNotWriteCallerArgs(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")

	args := make([]any, 1, 2)
	args[0] = "42"
	app.Get("/f", func(ctx iris.Context) {
		errors.HandleErrorf(ctx, io.ErrClosedPipe, "plan %s", args...)
	})
	app.Get("/rf", func(ctx iris.Context) {
		errors.HandleRequestErrorf(ctx, io.ErrClosedPipe, "plan %s", args...)
	})

	e := httptest.New(t, app)
	e.GET("/f").Expect().Status(httptest.StatusInternalServerError)
	if spare := args[:2][1]; spare != nil {
		t.Fatalf("HandleErrorf wrote into the caller's backing array: %v", spare)
	}
	e.GET("/rf").Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("message", "plan 42: "+io.ErrClosedPipe.Error())
	if spare := args[:2][1]; spare != nil {
		t.Fatalf("HandleRequestErrorf wrote into the caller's backing array: %v", spare)
	}
}

// Every render path, not only HandleError, stays silent after a body has gone
// out: the status and the partial body are committed.
func TestRenderAfterPartialBodyIsSuppressed(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	app.Get("/message", func(ctx iris.Context) {
		ctx.WriteString("partial")
		errors.NotFound.Message(ctx, "nope")
	})
	app.Get("/json", func(ctx iris.Context) {
		ctx.WriteString("partial")
		errors.HandleRequestJSONError(ctx, io.ErrUnexpectedEOF)
	})

	srv := serve(t, app)
	for _, path := range []string{"/message", "/json"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK || string(body) != "partial" {
			t.Fatalf("%s: status %d body %q, want 200 and exactly the partial body", path, resp.StatusCode, body)
		}
	}
}

// HandleRequestJSONError is the request side of the old three-argument
// HandleJSONError: a decode failure is the client's, so 400. Its twin
// HandleResponseJSONError is the encode side, so 500. Both report whether
// they wrote, and neither writes for a nil error.
func TestHandleJSONErrorSplitStatuses(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	app.Get("/request", func(ctx iris.Context) {
		if !errors.HandleRequestJSONError(ctx, io.ErrUnexpectedEOF) {
			t.Error("request: expected a written response")
		}
	})
	app.Get("/response", func(ctx iris.Context) {
		if !errors.HandleResponseJSONError(ctx, io.ErrUnexpectedEOF) {
			t.Error("response: expected a written response")
		}
	})
	app.Get("/nil", func(ctx iris.Context) {
		if errors.HandleRequestJSONError(ctx, nil) || errors.HandleResponseJSONError(ctx, nil) {
			t.Error("nil: nothing should be written")
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/request").Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("message", "unable to parse request body")
	e.GET("/response").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().HasValue("message", "unable to parse response body")
	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
}
