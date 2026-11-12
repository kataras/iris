package context_test

import (
	stdContext "context"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type detachKey string

// WithoutCancel returns a context that survives request cancellation while
// keeping the request context's values — for must-finish work spawned from
// a handler.
func TestContextWithoutCancel(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		// Attach a cancelable, value-carrying context to the request,
		// as an auth or tracing middleware would.
		parent := stdContext.WithValue(ctx.Request().Context(), detachKey("user"), "makis")
		cancelable, cancel := stdContext.WithCancel(parent)
		ctx.SetRequest(ctx.Request().WithContext(cancelable))

		detached := ctx.WithoutCancel()
		cancel() // the request context dies; the detached one must not.

		if err := detached.Err(); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString("detached context canceled: " + err.Error())
			return
		}

		user, _ := detached.Value(detachKey("user")).(string)
		if user != "makis" {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString("value lost")
			return
		}

		ctx.WriteString("detached-ok")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("detached-ok")
}
