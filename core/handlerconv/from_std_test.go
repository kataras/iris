// black-box testing
package handlerconv_test

import (
	"net/http"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
	"github.com/kataras/iris/httptest"
)

func TestFromStd(t *testing.T) {
	expected := "ok"
	std := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expected))
	}

	h := handlerconv.FromStd(http.HandlerFunc(std))

	hFunc := handlerconv.FromStd(std)

	app := iris.New()
	app.Get("/handler", h)
	app.Get("/func", hFunc)

	e := httptest.New(t, app)

	e.GET("/handler").
		Expect().Status(iris.StatusOK).Body().Equal(expected)

	e.GET("/func").
		Expect().Status(iris.StatusOK).Body().Equal(expected)
}

func TestFromStdWithNext(t *testing.T) {

	basicauth := "secret"
	passed := "ok"

	stdWNext := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if username, password, ok := r.BasicAuth(); ok &&
			username == basicauth && password == basicauth {
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(iris.StatusForbidden)
	}

	h := handlerconv.FromStdWithNext(stdWNext)
	next := func(ctx context.Context) {
		ctx.WriteString(passed)
	}

	app := iris.New()
	app.Get("/handlerwithnext", h, next)

	e := httptest.New(t, app)

	e.GET("/handlerwithnext").
		Expect().Status(iris.StatusForbidden)

	e.GET("/handlerwithnext").WithBasicAuth(basicauth, basicauth).
		Expect().Status(iris.StatusOK).Body().Equal(passed)
}
