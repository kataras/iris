package router_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"
)

func TestStackCall(t *testing.T) {
	// build the api
	app := iris.New()

	// setup an existing routes
	app.Handle(iris.MethodGet, "/route", func(ctx context.Context) {
		ctx.WriteString("ROUTED")
	})

	// setup fallback handler
	app.Fallback(func(ctx context.Context) {
		if ctx.Method() != iris.MethodGet {
			ctx.NextOrNotFound() //	it checks if we have next, otherwise fire 404 not found.
			return
		}

		ctx.WriteString("FALLBACK")
	})

	// run the tests
	e := httptest.New(t, app, httptest.Debug(false))

	e.Request(iris.MethodGet, "/route").Expect().Status(iris.StatusOK).Body().Equal("ROUTED")
	e.Request(iris.MethodPost, "/route").Expect().Status(iris.StatusNotFound)
	e.Request(iris.MethodPost, "/noroute").Expect().Status(iris.StatusNotFound)
	e.Request(iris.MethodGet, "/noroute").Expect().Status(iris.StatusOK).Body().Equal("FALLBACK")
}
