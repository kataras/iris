// black-box testing
package handlerconv_test

import (
	stdContext "context"
	"net/http"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/handlerconv"
	"github.com/kataras/iris/v12/httptest"
)

func TestFromStd(t *testing.T) {
	expected := "ok"
	std := func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(expected))
		if err != nil {
			t.Fatal(err)
		}
	}

	h := handlerconv.FromStd(http.HandlerFunc(std))

	hFunc := handlerconv.FromStd(std)

	app := iris.New()
	app.Get("/handler", h)
	app.Get("/func", hFunc)

	e := httptest.New(t, app)

	e.GET("/handler").
		Expect().Status(iris.StatusOK).Body().IsEqual(expected)

	e.GET("/func").
		Expect().Status(iris.StatusOK).Body().IsEqual(expected)
}

func TestFromStdWithNext(t *testing.T) {
	basicauth := "secret"
	passed := "ok"

	type contextKey string
	stdWNext := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if username, password, ok := r.BasicAuth(); ok &&
			username == basicauth && password == basicauth {
			ctx := stdContext.WithValue(r.Context(), contextKey("key"), "ok")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		w.WriteHeader(iris.StatusForbidden)
	}

	h := handlerconv.FromStdWithNext(stdWNext)
	next := func(ctx *context.Context) {
		ctx.WriteString(ctx.Request().Context().Value(contextKey("key")).(string))
	}

	app := iris.New()
	app.Get("/handlerwithnext", h, next)

	e := httptest.New(t, app)

	e.GET("/handlerwithnext").
		Expect().Status(iris.StatusForbidden)

	e.GET("/handlerwithnext").WithBasicAuth(basicauth, basicauth).
		Expect().Status(iris.StatusOK).Body().IsEqual(passed)
}
