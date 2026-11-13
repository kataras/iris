package router_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// The mux is compiled once from a snapshot of the route registry, so a route
// registered afterwards is never served. That used to happen in silence.
func TestRouteRegisteredAfterBuildWarns(t *testing.T) {
	app := iris.New()
	app.Get("/before", func(ctx iris.Context) {})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	app.Logger().SetLevel("warn")
	app.Logger().SetOutput(&buf)

	app.Get("/after", func(ctx iris.Context) {})

	got := buf.String()
	if !strings.Contains(got, "/after") {
		t.Fatalf("expected a warning naming the late route, got: %q", got)
	}
	// Only ".go:", not this file's name: getCaller walks past every frame under
	// github.com/kataras/iris/v14 to find the end-developer's, and an in-repo
	// test is such a frame, so it resolves to testing.go here. Outside the
	// module it names the caller's own file.
	if !strings.Contains(got, ".go:") {
		t.Fatalf("expected the warning to name the caller's file and line, got: %q", got)
	}
	if strings.Contains(got, "/before") {
		t.Fatalf("the route registered before the build must not be reported, got: %q", got)
	}
	if !strings.Contains(got, "Options.Router.EnableDynamicHandler") {
		t.Fatalf("expected the warning to name the option that serves late routes, got: %q", got)
	}
}

// With EnableDynamicHandler a route registered after the build is served once
// BuildRouter runs again, so registering it is not reported.
func TestRouteRegisteredAfterBuildDynamicHandler(t *testing.T) {
	app := iris.New(iris.Options{Router: iris.RouterOptions{EnableDynamicHandler: true}})
	app.Get("/before", func(ctx iris.Context) {})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	app.Logger().SetLevel("warn")
	app.Logger().SetOutput(&buf)

	app.Get("/after", func(ctx iris.Context) { ctx.WriteString("after") })
	app.Router("/v1").Get("/after", func(ctx iris.Context) {})

	if got := buf.String(); got != "" {
		t.Fatalf("expected no warning for a late route with the option on, got: %q", got)
	}

	if err := app.BuildRouter(); err != nil {
		t.Fatal(err)
	}
	e := httptest.New(t, app)
	e.GET("/after").Expect().Status(httptest.StatusOK).Body().IsEqual("after")
	e.GET("/v1/after").Expect().Status(httptest.StatusOK)
}

// Without EnableDynamicHandler the router is built once: a second BuildRouter
// returns an error that names the option and leaves the router as it was.
func TestBuildRouterTwiceReturnsError(t *testing.T) {
	app := iris.New()

	var wrapped atomic.Int64
	app.AddRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		wrapped.Add(1)
		next(w, r)
	})
	app.Get("/before", func(ctx iris.Context) { ctx.WriteString("before") })

	e := httptest.New(t, app)
	e.GET("/before").Expect().Status(httptest.StatusOK).Body().IsEqual("before")

	app.Get("/after", func(ctx iris.Context) { ctx.WriteString("after") })
	for range 2 {
		err := app.BuildRouter()
		if err == nil {
			t.Fatal("expected a second BuildRouter to return an error")
		}
		if !strings.Contains(err.Error(), "Options.Router.EnableDynamicHandler") {
			t.Fatalf("expected the error to name the option, got: %v", err)
		}
	}

	wrapped.Store(0)
	e.GET("/before").Expect().Status(httptest.StatusOK).Body().IsEqual("before")
	if n := wrapped.Load(); n != 1 {
		t.Fatalf("the router wrapper ran %d times for one request", n)
	}
	e.GET("/after").Expect().Status(httptest.StatusNotFound)
}

// EnableDynamicHandler is read at the first build, as the rest of the router
// options: what a second BuildRouter does depends on how the router was built.
func TestBuildRouterTwiceOptionChangedAfterBuild(t *testing.T) {
	for _, builtWith := range []bool{false, true} {
		app := iris.New(iris.Options{Router: iris.RouterOptions{EnableDynamicHandler: builtWith}})
		app.Get("/before", func(ctx iris.Context) {})
		e := httptest.New(t, app)

		app.EditOptions(func(opts *iris.Options) {
			opts.Router.EnableDynamicHandler = !builtWith
		})
		app.Get("/after", func(ctx iris.Context) {})
		err := app.BuildRouter()

		if builtWith {
			if err != nil {
				t.Fatalf("built with the option on: %v", err)
			}
			e.GET("/after").Expect().Status(httptest.StatusOK)
		} else {
			if err == nil || !strings.Contains(err.Error(), "Options.Router.EnableDynamicHandler") {
				t.Fatalf("built with the option off: expected the error that names the option, got: %v", err)
			}
			e.GET("/after").Expect().Status(httptest.StatusNotFound)
		}
		e.GET("/before").Expect().Status(httptest.StatusOK)
	}
}

// Build runs once, so a repeated Build (httptest.New and then Build, NewServer,
// Listen or Run) never asks for a second BuildRouter.
func TestBuildTwice(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		app := iris.New(iris.Options{Router: iris.RouterOptions{EnableDynamicHandler: dynamic}})
		app.Get("/a", func(ctx iris.Context) { ctx.WriteString("a") })

		e := httptest.New(t, app) // the first Build.
		for range 2 {
			if err := app.Build(); err != nil {
				t.Fatalf("EnableDynamicHandler=%v: repeated Build: %v", dynamic, err)
			}
		}

		srv := httptest.NewServer(t, app) // Build again, then a listener on a free port.
		resp, err := http.Get(srv.URL + "/a")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		srv.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "a" {
			t.Fatalf("EnableDynamicHandler=%v: got %d %q", dynamic, resp.StatusCode, body)
		}

		e.GET("/a").Expect().Status(httptest.StatusOK).Body().IsEqual("a")
	}
}

// A rebuild compiles the routes and the 404 and 405 handlers again. The router
// wrappers, the UseRouter handlers and the router options stay as the first
// build left them.
func TestDynamicHandlerRebuildScope(t *testing.T) {
	app := iris.New(iris.Options{Router: iris.RouterOptions{
		EnableDynamicHandler: true,
		FireMethodNotAllowed: true,
	}})
	app.Get("/a", func(ctx iris.Context) { ctx.WriteString("a") })
	app.Get("/Upper", func(ctx iris.Context) {})

	e := httptest.New(t, app)
	e.GET("/nope").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
	e.POST("/a").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")

	var late atomic.Int64
	app.HandleNoRoute(func(ctx iris.Context) { ctx.WriteString("custom 404") })
	app.HandleNoMethod(func(ctx iris.Context) { ctx.WriteString("custom 405") })
	app.AddRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		late.Add(1)
		next(w, r)
	})
	app.UseRouter(func(ctx iris.Context) {
		late.Add(1)
		ctx.Next()
	})
	app.EditOptions(func(opts *iris.Options) {
		opts.Router.FireMethodNotAllowed = false
		opts.Router.ForceLowercaseRouting = true
	})
	if err := app.BuildRouter(); err != nil {
		t.Fatal(err)
	}

	// Rebuilt.
	e.GET("/nope").Expect().Status(httptest.StatusNotFound).Body().IsEqual("custom 404")
	e.POST("/a").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("custom 405")
	// Not rebuilt: POST /a above is still a 405, and the paths are not lowercased.
	e.GET("/UPPER").Expect().Status(httptest.StatusNotFound)
	e.GET("/a").Expect().Status(httptest.StatusOK).Body().IsEqual("a")
	if n := late.Load(); n != 0 {
		t.Fatalf("a router wrapper or a UseRouter handler registered after the first build ran %d times", n)
	}
}
