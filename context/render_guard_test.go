package context_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

// A body that already went out is never followed by an error document: the
// wire would carry "partial" plus a JSON error under the 200 that was
// already committed.
func TestStopWithJSONAfterPartialBody(t *testing.T) {
	app := iris.New()
	app.Get("/partial", func(ctx iris.Context) {
		ctx.WriteString("partial")
		_ = ctx.StopWithJSON(http.StatusInternalServerError, iris.Map{"message": "late"})
	})

	srv := serve(t, app)
	resp, err := http.Get(srv.URL + "/partial")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want the already-committed 200", resp.StatusCode)
	}
	if string(body) != "partial" {
		t.Fatalf("body = %q, want exactly the partial body", body)
	}
}

// The built-in fallback renderer answers an oversize body with 413 like the
// canonical one.
func TestMinimalRendererTooLarge(t *testing.T) {
	app := iris.New()
	app.Get("/large", func(ctx iris.Context) {
		context.MinimalRestErrorRenderer.HandleRequestError(ctx, context.ErrRequestEntityTooLarge)
	})

	e := httptest.New(t, app)
	e.GET("/large").Expect().Status(httptest.StatusRequestEntityTooLarge)
}

// Once the chain is stopped the fallback renderer writes nothing, on every
// method, matching the canonical renderer.
func TestMinimalRendererRespectsStopped(t *testing.T) {
	cases := map[string]func(ctx iris.Context){
		"validation": func(ctx iris.Context) {
			context.MinimalRestErrorRenderer.HandleValidationError(ctx, io.ErrUnexpectedEOF)
		},
		"not-found": func(ctx iris.Context) {
			context.MinimalRestErrorRenderer.HandleNotFound(ctx, "missing %s", "thing")
		},
		"invalid-argument": func(ctx iris.Context) {
			context.MinimalRestErrorRenderer.HandleInvalidArgument(ctx, "bad %s", "thing")
		},
	}

	app := iris.New()
	for name, render := range cases {
		app.Get("/"+name, func(ctx iris.Context) {
			ctx.StopExecution()
			render(ctx)
		})
	}

	e := httptest.New(t, app)
	for name := range cases {
		e.GET("/" + name).Expect().Status(httptest.StatusOK).Body().IsEmpty()
	}
}

// JSONP marshals before it writes: a marshal failure must not leave a dangling
// "callback(" on the wire, and the payload is escaped for a script body
// (U+2028/U+2029 break JavaScript string literals).
func TestWriteJSONPMarshalsBeforeWriting(t *testing.T) {
	app := iris.New()
	app.Get("/bad", func(ctx iris.Context) {
		if err := ctx.JSONP(make(chan int), iris.JSONP{Callback: "cb"}); err == nil {
			t.Error("expected a marshal error")
		}
	})
	app.Get("/escape", func(ctx iris.Context) {
		_ = ctx.JSONP(iris.Map{"s": "a\u2028b\u2029c"}, iris.JSONP{Callback: "cb"})
	})

	e := httptest.New(t, app)
	if body := e.GET("/bad").Expect().Body().Raw(); strings.HasPrefix(body, "cb(") {
		t.Fatalf("a failed marshal wrote the callback prefix: %q", body)
	}
	e.GET("/escape").Expect().Status(httptest.StatusOK).Body().IsEqual("cb({\"s\":\"a\\u2028b\\u2029c\"}\n);")
}

// MsgPack reports the marshal error instead of writing an empty body.
func TestMsgPackReturnsMarshalError(t *testing.T) {
	app := iris.New()
	app.Get("/bad", func(ctx iris.Context) {
		if err := ctx.MsgPack(make(chan int)); err == nil {
			t.Error("expected a marshal error")
		}
		if n := ctx.ResponseWriter().Written(); n > 0 {
			t.Errorf("wrote %d bytes on a failed marshal", n)
		}
	})

	e := httptest.New(t, app)
	e.GET("/bad").Expect()
}
