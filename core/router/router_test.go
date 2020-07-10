package router_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
)

func TestRouteExists(t *testing.T) {
	// build the api
	app := iris.New()
	emptyHandler := func(*context.Context) {}

	// setup the tested routes
	app.Handle("GET", "/route-exists", emptyHandler)
	app.Handle("POST", "/route-with-param/{param}", emptyHandler)

	// check RouteExists
	app.Handle("GET", "/route-test", func(ctx *context.Context) {
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

func TestLowercaseRouting(t *testing.T) {
	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// test bottom to begin wrapper, the last ones should execute first.
		// The ones that are registered at `Build` state, after this `WrapRouter` call.
		// So path should be already lowecased.
		if expected, got := strings.ToLower(r.URL.Path), r.URL.Path; expected != got {
			t.Fatalf("expected path: %s but got: %s", expected, got)
		}
		next(w, r)
	})

	h := func(ctx iris.Context) { ctx.WriteString(ctx.Path()) }

	// Register routes.
	tests := []string{"/", "/lowercase", "/UPPERCASE", "/Title", "/m1xEd2"}
	for _, tt := range tests {
		app.Get(tt, h)
	}

	app.Configure(iris.WithLowercaseRouting)
	// Test routes.
	e := httptest.New(t, app)
	for _, tt := range tests {
		s := strings.ToLower(tt)
		e.GET(tt).Expect().Status(httptest.StatusOK).Body().Equal(s)
		e.GET(s).Expect().Status(httptest.StatusOK).Body().Equal(s)
		e.GET(strings.ToUpper(tt)).Expect().Status(httptest.StatusOK).Body().Equal(s)
	}
}
