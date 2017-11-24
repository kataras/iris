package mvc2_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/mvc2"
)

var (
	lowLevelHandler = func(ctx iris.Context) {
		ctx.Writef("low-level handler")
	}
)

func TestHandler(t *testing.T) {
	app := iris.New()
	m := mvc2.New()

	// should just return a context.Handler
	// without performance penalties.
	app.Get("/", m.Handler(lowLevelHandler))

	e := httptest.New(t, app, httptest.LogLevel("debug"))
	// 1
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("low-level handler")

}
