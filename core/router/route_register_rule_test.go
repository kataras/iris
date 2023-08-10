package router_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/httptest"
)

func TestRegisterRule(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithDynamicHandler)

	// collect the error on RouteError rule.
	buf := new(bytes.Buffer)
	app.Logger().SetTimeFormat("").DisableNewLine().SetOutput(buf)

	v1 := app.Party("/v1")
	v1.SetRegisterRule(iris.RouteSkip)

	getHandler := func(ctx iris.Context) {
		ctx.Writef("[get] %s", ctx.Method())
	}

	anyHandler := func(ctx iris.Context) {
		ctx.Writef("[any] %s", ctx.Method())
	}

	getRoute := v1.Get("/", getHandler)
	v1.Any("/", anyHandler)
	if route := v1.Get("/", getHandler); !reflect.DeepEqual(route, getRoute) {
		t.Fatalf("expected route to be equal with the original get route")
	}

	// test RouteSkip.
	e := httptest.New(t, app, httptest.LogLevel("error"))
	testRegisterRule(e, "[get] GET")

	// test RouteOverride (default behavior).
	v1.SetRegisterRule(iris.RouteOverride)
	v1.Any("/", anyHandler)
	app.RefreshRouter()
	testRegisterRule(e, "[any] GET")

	// test RouteError.
	v1.SetRegisterRule(iris.RouteError)
	if route := v1.Get("/", getHandler); route != nil {
		t.Fatalf("expected duplicated route, with RouteError rule, to be nil but got: %#+v", route)
	}
	if expected, got := "[ERRO] new route: GET /v1 conflicts with an already registered one: GET /v1 route", buf.String(); expected != got {
		t.Fatalf("expected api builder's error to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}
}

func testRegisterRule(e *httptest.Expect, expectedGetBody string) {
	for _, method := range router.AllMethods {
		tt := e.Request(method, "/v1").Expect().Status(httptest.StatusOK).Body()
		if method == iris.MethodGet {
			tt.IsEqual(expectedGetBody)
		} else {
			tt.IsEqual("[any] " + method)
		}
	}
}

func TestRegisterRuleOverlap(t *testing.T) {
	app := iris.New()
	// TODO(@kataras) the overlapping does not work per-party yet,
	// it just checks compares from the total app's routes (which is the best possible action to do
	// because MVC applications can be separated into different parties too?).
	usersRouter := app.Party("/users")
	usersRouter.SetRegisterRule(iris.RouteOverlap)

	// second handler will be executed, status will be reset-ed as well,
	// stop without data written.
	usersRouter.Get("/", func(ctx iris.Context) {
		ctx.StopWithStatus(iris.StatusUnauthorized)
	})
	usersRouter.Get("/", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusOK)
		ctx.WriteString("data")
	})

	// first handler will be executed, no stop called.
	usersRouter.Get("/p1", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusUnauthorized)
	})
	usersRouter.Get("/p1", func(ctx iris.Context) {
		ctx.WriteString("not written")
	})

	// first handler will be executed, stop but with data sent on default writer
	// (body sent cannot be reset-ed here).
	usersRouter.Get("/p2", func(ctx iris.Context) {
		ctx.StopWithText(iris.StatusUnauthorized, "no access")
	})
	usersRouter.Get("/p2", func(ctx iris.Context) {
		ctx.WriteString("not written")
	})

	// second will be executed, response can be reset-ed on recording.
	usersRouter.Get("/p3", func(ctx iris.Context) {
		ctx.Record()
		ctx.StopWithText(iris.StatusUnauthorized, "no access")
	})
	usersRouter.Get("/p3", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusOK)
		ctx.WriteString("p3 data")
	})

	e := httptest.New(t, app)

	e.GET("/users").Expect().Status(httptest.StatusOK).Body().IsEqual("data")
	e.GET("/users/p1").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
	e.GET("/users/p2").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("no access")
	e.GET("/users/p3").Expect().Status(httptest.StatusOK).Body().IsEqual("p3 data")
}
