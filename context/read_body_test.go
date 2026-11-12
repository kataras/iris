package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// newReadBodyApp exposes ReadBody over four routes so the raw-target
// (*string, *[]byte) special cases can be asserted through real requests.
func newReadBodyApp() *iris.Application {
	app := iris.New()

	app.Post("/string", func(ctx iris.Context) {
		var s string
		if err := ctx.ReadBody(&s); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString(err.Error())
			return
		}
		ctx.WriteString(s)
	})

	app.Post("/bytes", func(ctx iris.Context) {
		var b []byte
		if err := ctx.ReadBody(&b); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString(err.Error())
			return
		}
		ctx.Write(b)
	})

	app.Post("/json", func(ctx iris.Context) {
		var payload struct {
			Name string `json:"name"`
		}
		if err := ctx.ReadBody(&payload); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString(err.Error())
			return
		}
		ctx.WriteString(payload.Name)
	})

	app.Get("/raw", func(ctx iris.Context) {
		var s string
		if err := ctx.ReadBody(&s); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString(err.Error())
			return
		}
		ctx.WriteString("ok:" + s)
	})

	return app
}

// ReadBody with a *string target must fill the string with the raw body and
// return immediately, without re-dispatching to a content-type decoder over
// the already-drained body.
func TestReadBodyStringTarget(t *testing.T) {
	e := httptest.New(t, newReadBodyApp())

	e.POST("/string").WithText("hello, iris").Expect().
		Status(httptest.StatusOK).Body().IsEqual("hello, iris")
}

// ReadBody with a *[]byte target must assign the body bytes (not copy into a
// zero-length destination) and return immediately.
func TestReadBodyBytesTarget(t *testing.T) {
	e := httptest.New(t, newReadBodyApp())

	e.POST("/bytes").WithBytes([]byte("raw payload bytes")).Expect().
		Status(httptest.StatusOK).Body().IsEqual("raw payload bytes")
}

// Struct targets keep the content-type dispatch (JSON here).
func TestReadBodyStructTargetStillDispatches(t *testing.T) {
	e := httptest.New(t, newReadBodyApp())

	e.POST("/json").WithJSON(map[string]string{"name": "iris"}).Expect().
		Status(httptest.StatusOK).Body().IsEqual("iris")
}

// A raw *string target on a GET request with a query string must not be
// routed through ReadQuery: raw targets read the body as-is (empty here).
func TestReadBodyStringTargetIgnoresQuery(t *testing.T) {
	e := httptest.New(t, newReadBodyApp())

	e.GET("/raw").WithQuery("x", "1").Expect().
		Status(httptest.StatusOK).Body().IsEqual("ok:")
}
