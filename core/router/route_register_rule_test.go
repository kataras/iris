package router_test

import (
	"reflect"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/httptest"
)

func TestRegisterRule(t *testing.T) {
	app := iris.New()
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
	if expected, got := 1, len(v1.GetReporter().Errors); expected != got {
		t.Fatalf("expected api builder's errors length to be: %d but got: %d", expected, got)
	}
}

func testRegisterRule(e *httptest.Expect, expectedGetBody string) {
	for _, method := range router.AllMethods {
		tt := e.Request(method, "/v1").Expect().Status(httptest.StatusOK).Body()
		if method == iris.MethodGet {
			tt.Equal(expectedGetBody)
		} else {
			tt.Equal("[any] " + method)
		}
	}
}
