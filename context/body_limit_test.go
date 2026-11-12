package context_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

// newBodyLimitApp builds an app whose /echo handler reads the raw body and maps
// ErrRequestEntityTooLarge to 413. When limit > 0 it sets ContextOptions.BodyMaxMemory
// per request (the production path is the app's Configuration.Context).
func newBodyLimitApp(limit int64) *iris.Application {
	app := iris.New()

	if limit > 0 {
		app.Use(func(ctx iris.Context) {
			ctx.Options().BodyMaxMemory = limit
			ctx.Next()
		})
	}

	app.Post("/echo", func(ctx iris.Context) {
		b, err := ctx.Body()
		if err != nil {
			if errors.Is(err, context.ErrRequestEntityTooLarge) {
				ctx.StopWithStatus(httptest.StatusRequestEntityTooLarge)
				return
			}
			ctx.StopWithStatus(httptest.StatusInternalServerError)
			return
		}
		ctx.Write(b)
	})

	return app
}

func TestContextBodyMaxMemory(t *testing.T) {
	app := newBodyLimitApp(16)
	e := httptest.New(t, app)

	// Under the limit: echoed back.
	e.POST("/echo").WithBytes([]byte("small")).Expect().
		Status(httptest.StatusOK).Body().IsEqual("small")

	// Over the limit: 413 Request Entity Too Large.
	e.POST("/echo").WithBytes([]byte(strings.Repeat("x", 64))).Expect().
		Status(httptest.StatusRequestEntityTooLarge)
}

func TestContextBodyUnlimitedByDefault(t *testing.T) {
	app := newBodyLimitApp(0) // no BodyMaxMemory configured -> unlimited (default).
	e := httptest.New(t, app)

	big := strings.Repeat("x", 4096)
	e.POST("/echo").WithBytes([]byte(big)).Expect().
		Status(httptest.StatusOK).Body().IsEqual(big)
}

// TestContextBodyMaxMemoryReadJSON verifies BodyMaxMemory is enforced on the
// plain ReadJSON path (and therefore ReadBody's JSON dispatch and
// ctx.BindJSON), not only on Body and the streaming reader.
func TestContextBodyMaxMemoryReadJSON(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().BodyMaxMemory = 16
		ctx.Next()
	})
	app.Post("/json", func(ctx iris.Context) {
		var v map[string]any
		if err := ctx.ReadJSON(&v); err != nil {
			if errors.Is(err, context.ErrRequestEntityTooLarge) {
				ctx.StopWithStatus(httptest.StatusRequestEntityTooLarge)
				return
			}
			ctx.StopWithStatus(httptest.StatusBadRequest)
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	// Under the limit.
	e.POST("/json").WithBytes([]byte(`{"a":1}`)).
		WithHeader("Content-Type", "application/json").Expect().
		Status(httptest.StatusOK)
	// Over the limit -> 413, not a fully-buffered decode.
	e.POST("/json").WithBytes([]byte(`{"a":"`+strings.Repeat("x", 10000)+`"}`)).
		WithHeader("Content-Type", "application/json").Expect().
		Status(httptest.StatusRequestEntityTooLarge)
}

// TestContextBodyMaxMemoryReadJSONStream verifies BodyMaxMemory is also enforced
// on the streaming JSON path (ReadJSONStream), not only Body.
func TestContextBodyMaxMemoryReadJSONStream(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().BodyMaxMemory = 16
		ctx.Next()
	})
	app.Post("/stream", func(ctx iris.Context) {
		err := ctx.ReadJSONStream(func(decode context.DecodeFunc) error {
			var v map[string]any
			return decode(&v)
		})
		if err != nil {
			if errors.Is(err, context.ErrRequestEntityTooLarge) {
				ctx.StopWithStatus(httptest.StatusRequestEntityTooLarge)
				return
			}
			ctx.StopWithStatus(httptest.StatusBadRequest)
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	// Under the limit.
	e.POST("/stream").WithBytes([]byte(`{"a":1}`)).Expect().Status(httptest.StatusOK)
	// Well over the limit (spans multiple decoder reads, so the cap triggers
	// mid-stream before a complete value is buffered) -> 413.
	e.POST("/stream").WithBytes([]byte(`{"a":"` + strings.Repeat("x", 10000) + `"}`)).Expect().
		Status(httptest.StatusRequestEntityTooLarge)
}
