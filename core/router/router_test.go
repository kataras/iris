package router_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"
)

func TestRouteExists(t *testing.T) {
	// build the api
	app := iris.New()
	emptyHandler := func(context.Context) {}

	// setup the tested routes
	app.Handle("GET", "/route-exists", emptyHandler)
	app.Handle("POST", "/route-with-param/{param}", emptyHandler)

	// check RouteExists
	app.Handle("GET", "/route-test", func(ctx context.Context) {
		if ctx.RouteExists("GET", "/route-not-exists") {
			t.Error("Route with path should not exists")
		}

		if ctx.RouteExists("POST", "/route-exists") {
			t.Error("Route with method should not exists")
		}

		if !ctx.RouteExists("GET", "/route-exists") {
			t.Error("Route 1 should exists")
		}

		if !ctx.RouteExists("POST", "/route-with-param/a-param") {
			t.Error("Route 2 should exists")
		}
	})

	// run the tests
	httptest.New(t, app, httptest.Debug(false)).Request("GET", "/route-test").Expect().Status(iris.StatusOK)
}
