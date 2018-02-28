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
		ctx.WriteString("ROUTED@APP")
	})

	// new party
	party := app.Party("/api/{p:path}", func(ctx context.Context) {
		if ctx.Params().Get("p") == "A" {
			ctx.WriteString("H@PARTY-")
		}

		ctx.Next()
	})

	// existing route in the party
	party.Get("/value", func(ctx context.Context) {
		ctx.WriteString("ROUTED@PARTY")
	})

	// party specific fallback
	party.Fallback(func(ctx context.Context) {
		if ctx.Params().Get("p") == "B" {
			ctx.Next() // fires 404 not found.
			return
		}

		ctx.WriteString("FALLBACK@PARTY")
	})

	// global middleware
	app.UseGlobal(func(ctx context.Context) {
		ctx.WriteString("MW-")
	})

	// setup fallback handler
	app.Fallback(func(ctx context.Context) {
		if ctx.Method() != iris.MethodGet {
			ctx.Next() // fires 404 not found.
			return
		}

		ctx.WriteString("FALLBACK@APP")
	})

	// run the tests
	e := httptest.New(t, app, httptest.Debug(false))

	app.RefreshRouter()
	t.Log("\n" + app.RequestHandlerRepresention())

	e.Request(iris.MethodGet, "/route").Expect().Status(iris.StatusOK).Body().Equal("MW-ROUTED@APP")
	e.Request(iris.MethodPost, "/route").Expect().Status(iris.StatusNotFound)
	e.Request(iris.MethodPost, "/noroute").Expect().Status(iris.StatusNotFound)
	e.Request(iris.MethodGet, "/noroute").Expect().Status(iris.StatusOK).Body().Equal("MW-FALLBACK@APP")
	e.Request(iris.MethodGet, "/api/X/value").Expect().Status(iris.StatusOK).Body().Equal("MW-ROUTED@PARTY")
	e.Request(iris.MethodGet, "/api/A/value").Expect().Status(iris.StatusOK).Body().Equal("MW-H@PARTY-ROUTED@PARTY")
	e.Request(iris.MethodGet, "/api/X/no").Expect().Status(iris.StatusOK).Body().Equal("MW-FALLBACK@PARTY")
	e.Request(iris.MethodGet, "/api/A/no").Expect().Status(iris.StatusOK).Body().Equal("MW-H@PARTY-FALLBACK@PARTY")
	e.Request(iris.MethodGet, "/api/B/no").Expect().Status(iris.StatusNotFound)
}
