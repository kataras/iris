package router_test

import (
	"reflect"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/httptest"
)

func TestRegisterRule(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			EnableDynamicHandler: true,
		},
	})

	// collect the error on RouteError rule.
	// buf := new(bytes.Buffer)
	// app.Logger().SetTimeFormat("").DisableNewLine().SetOutput(buf)

	v1 := app.Router("/v1")
	v1.Routes().SetRegisterRule(iris.RouteSkip)

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
	v1.Routes().SetRegisterRule(iris.RouteOverride)
	v1.Any("/", anyHandler)
	app.BuildRouter()
	testRegisterRule(e, "[any] GET")

	// test RouteError.
	v1.Routes().SetRegisterRule(iris.RouteError)
	// It should panic, test the panic's error message through recover,
	// the message should be: new route GET /v1 conflicts with the old route GET /v1

	defer func() {
		if r := recover(); r != nil {
			if expected, got := "new route GET /v1 conflicts with the old route GET /v1", r.(error).Error(); expected != got {
				t.Fatalf("expected api builder's error to be:\n'%s'\nbut got:\n'%s'", expected, got)
			}
		}
	}()

	if route := v1.Get("/", getHandler); route != nil {
		t.Fatalf("expected duplicated route, with RouteError rule, to be nil but got: %s", route)
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
