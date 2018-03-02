package router_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/httptest"
)

type tRouteList []*router.Route

func (rl tRouteList) String() string {
	var out bytes.Buffer

	for _, r := range rl {
		fmt.Fprintln(&out, r)
	}

	return out.String()
}

func TestFallbackHandlers(t *testing.T) {
	// build the api
	app := iris.New()

	// setup an existing routes
	app.Handle(iris.MethodGet, "/route", func(ctx context.Context) {
		ctx.WriteString(ctx.Values().GetString("my-value") + "ROUTED@APP")
	})

	// new party
	party := app.Party("/api/{p}", func(ctx context.Context) {
		if ctx.Params().Get("p") == "A" {
			vals := ctx.Values()

			vals.Set("my-value", vals.GetString("my-value")+"H@PARTY-")
		}

		ctx.Next()
	})

	// existing route in the party
	party.Get("/value", func(ctx context.Context) {
		ctx.WriteString(ctx.Values().GetString("my-value") + "ROUTED@PARTY")
	})

	// party specific fallback
	party.Fallback(func(ctx context.Context) {
		if ctx.Params().Get("p") == "B" {
			ctx.Next() // fires 404 not found.
			return
		}

		ctx.WriteString(ctx.Values().GetString("my-value") + "FALLBACK@PARTY")
	})

	// global middleware
	app.UseGlobal(func(ctx context.Context) {
		ctx.Values().Set("my-value", "MW-")
		ctx.Next()
	})

	// setup fallback handler
	app.Fallback(func(ctx context.Context) {
		if ctx.Method() != iris.MethodGet {
			ctx.Next() // fires 404 not found.
			return
		}

		ctx.WriteString(ctx.Values().GetString("my-value") + "FALLBACK@APP")
	})

	// run the tests
	e := httptest.New(t, app, httptest.Debug(false))

	app.RefreshRouter()
	t.Logf("Reporter:\n%s", app.APIBuilder.GetReporter())
	t.Logf("Routes:\n%s", tRouteList(app.GetRoutes()))
	t.Logf("Router:\n%s", app.RequestHandlerRepresention())

	fmt.Println(">>> CASE 001")
	e.Request(iris.MethodGet, "/route").Expect().Status(iris.StatusOK).Body().Equal("MW-ROUTED@APP")
	fmt.Println(">>> CASE 002")
	e.Request(iris.MethodPost, "/route").Expect().Status(iris.StatusNotFound)
	fmt.Println(">>> CASE 003")
	e.Request(iris.MethodPost, "/noroute").Expect().Status(iris.StatusNotFound)
	fmt.Println(">>> CASE 004")
	e.Request(iris.MethodGet, "/noroute").Expect().Status(iris.StatusOK).Body().Equal("MW-FALLBACK@APP")
	fmt.Println(">>> CASE 005")
	e.Request(iris.MethodGet, "/api/X/value").Expect().Status(iris.StatusOK).Body().Equal("MW-ROUTED@PARTY")
	fmt.Println(">>> CASE 006")
	e.Request(iris.MethodGet, "/api/A/value").Expect().Status(iris.StatusOK).Body().Equal("MW-H@PARTY-ROUTED@PARTY")
	fmt.Println(">>> CASE 007")
	e.Request(iris.MethodGet, "/api/X/no").Expect().Status(iris.StatusOK).Body().Equal("MW-FALLBACK@PARTY")
	fmt.Println(">>> CASE 008")
	e.Request(iris.MethodGet, "/api/A/no").Expect().Status(iris.StatusOK).Body().Equal("MW-H@PARTY-FALLBACK@PARTY")
	fmt.Println(">>> CASE 009")
	e.Request(iris.MethodGet, "/api/B/no").Expect().Status(iris.StatusNotFound)
}
