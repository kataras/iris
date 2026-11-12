// black-box testing
package router_test

import (
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"

	"github.com/kataras/iris/v14/httptest"
)

func TestRouterBuilderWrapRouter(t *testing.T) {
	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		if r.URL.Path == "/route" {
			r.URL.Path = "/modified_route" // change the request path so the routing algorithm will find the /modified_route handler instead of /route.
		} else {
			// No content.
			w.WriteHeader(iris.StatusNoContent)
			return
		}
		router(w, r)
	})

	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})
	app.Get("/modified_route", func(ctx *context.Context) {
		ctx.WriteString("modified_route")
	})

	e := httptest.New(t, app)
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("modified_route")
	e.GET("/other").Expect().Status(iris.StatusNoContent)
}

func TestRouterBuilderUseRouter(t *testing.T) {
	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		r := ctx.Request()
		if r.URL.Path == "/route" {
			r.URL.Path = "/modified_route" // change the request path so the routing algorithm will find the /modified_route handler instead of /route.
		}

		ctx.Next()
	})
	app.UseRouter(func(ctx iris.Context) {
		if ctx.Path() == "/modified_route" {
			ctx.WriteString("before_use_router")
		}

		ctx.Next()
	})
	app.Use(func(ctx *context.Context) {
		if !ctx.CurrentRoute().IsNoRoute() && !ctx.CurrentRoute().IsMethodNotAllowed() { // or ctx.GetStatusCode() < 400
			ctx.WriteString("before_use")
		}

		ctx.Next()
	})

	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})
	app.Get("/modified_route", func(ctx *context.Context) {
		ctx.Writef("modified_route: handler index should be 1 because of the app.Use, app.UseRouter resets the handler index: %d", ctx.HandlerIndex(-1))
	})

	e := httptest.New(t, app)
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("before_use_routerbefore_usemodified_route: handler index should be 1 because of the app.Use, app.UseRouter resets the handler index: 1")
	e.GET("/other").Expect().Status(iris.StatusNotFound).Body().IsEqual("Not Found")
}

func TestRouterBuilderHandleNoRoute_Default(t *testing.T) {
	app := iris.New()
	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusNotFound).Body().IsEqual("Not Found")
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("route")
	e.POST("/route").Expect().Status(iris.StatusNotFound).Body().IsEqual("Not Found") // no fire method not allowed option was set, so this should fire 404.
}

func TestRouterBuilderHandleNoMethod_Default(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})
	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusNotFound).Body().IsEqual("Not Found")
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("route")
	e.POST("/route").Expect().Status(iris.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")
}

func TestRouterBuilderHandleNoRoute(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx *context.Context) {
		ctx.WriteString("before")
		ctx.Next()
	})
	app.HandleNoRoute(func(ctx *context.Context) {
		ctx.WriteString("custom 404")
	})

	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusNotFound).Body().IsEqual("beforecustom 404")
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("beforeroute")
}

func TestRouterBuilderHandleNoMethod(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})

	app.Use(func(ctx *context.Context) {
		ctx.WriteString("before")
		ctx.Next()
	})
	app.HandleNoMethod(func(ctx *context.Context) {
		ctx.WriteString("custom 405")
	})

	app.Get("/route", func(ctx *context.Context) {
		ctx.WriteString("route")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusNotFound).Body().IsEqual("before") // remember, the default handler should not proceed writing into client if already written (see the middleware .Use).
	e.GET("/route").Expect().Status(iris.StatusOK).Body().IsEqual("beforeroute")
	e.POST("/route").Expect().Status(iris.StatusMethodNotAllowed).Body().IsEqual("beforecustom 405")
}

// TestRouterRunsReleaseHooksBeforeRecycling: OnRelease is the last point at
// which the response writer is still this request's. The router runs the hooks
// after the handler chain returns and before the writer and the Context go back
// to their pools, on the request's own goroutine, in reverse registration
// order, and exactly once.
func TestRouterRunsReleaseHooksBeforeRecycling(t *testing.T) {
	var (
		order          []string
		handlerDone    bool
		ranAfterReturn bool
		writerLive     bool
		runs           int
	)

	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		ctx.OnRelease(func() {
			runs++
			order = append(order, "first")
			ranAfterReturn = handlerDone
			writerLive = ctx.ResponseWriter() != nil && ctx.Request() != nil
		})
		ctx.OnRelease(func() { order = append(order, "second") })

		ctx.WriteString("ok")
		handlerDone = true
	})
	app.Get("/plain", func(ctx *context.Context) { ctx.WriteString("plain") })

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("ok")

	if got, want := len(order), 2; got != want {
		t.Fatalf("hooks ran %d times, want %d: %v", got, want, order)
	}
	if order[0] != "second" || order[1] != "first" {
		t.Errorf("order = %v, want reverse registration order, like defer", order)
	}
	if !ranAfterReturn {
		t.Error("a hook ran before the handler chain had returned")
	}
	if !writerLive {
		t.Error("the response writer or the request was already gone inside the hook")
	}

	// The next request through the same pool must not inherit the hooks.
	e.GET("/plain").Expect().Status(iris.StatusOK).Body().IsEqual("plain")
	if runs != 1 {
		t.Errorf("the first request's hook ran %d times after two requests, want 1", runs)
	}
}

// TestForceReleaseDropsReleaseHooks: a handler that force-releases its Context
// (the websocket upgrade does) hands it to the pool mid-chain; the router then
// runs the release hooks on a Context that may already be someone else's. The
// hooks must be gone by then, and running none must be safe.
func TestForceReleaseDropsReleaseHooks(t *testing.T) {
	var runs int

	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		ctx.OnRelease(func() { runs++ })
		ctx.WriteString("ok")
		ctx.Application().ContextPool().ForceRelease(ctx)
	})
	app.Get("/plain", func(ctx *context.Context) { ctx.WriteString("plain") })

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("ok")
	e.GET("/plain").Expect().Status(iris.StatusOK).Body().IsEqual("plain")

	if runs != 0 {
		t.Errorf("a hook ran %d times after ForceRelease, want 0: the Context was already back in the pool", runs)
	}
}

// TestReleaseHooksRegisteredWhileRunningStillRun: a hook that registers another
// one (closing a stream that closes a coalescer, say) gets it run in the same
// pass rather than silently dropped.
func TestReleaseHooksRegisteredWhileRunningStillRun(t *testing.T) {
	var order []string

	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		ctx.OnRelease(func() {
			order = append(order, "outer")
			ctx.OnRelease(func() { order = append(order, "inner") })
		})
		ctx.WriteString("ok")
	})

	httptest.New(t, app).GET("/").Expect().Status(iris.StatusOK)

	if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
		t.Errorf("order = %v, want [outer inner]", order)
	}
}

// TestReleaseHookThatReArmsItselfIsBounded: a hook that registers itself again
// must not spin the request goroutine forever. It runs a bounded number of
// times and the rest is dropped.
func TestReleaseHookThatReArmsItselfIsBounded(t *testing.T) {
	var runs int

	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		var rearm func()
		rearm = func() {
			runs++
			ctx.OnRelease(rearm)
		}
		ctx.OnRelease(rearm)
		ctx.WriteString("ok")
	})
	app.Get("/plain", func(ctx *context.Context) { ctx.WriteString("plain") })

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("ok")

	if runs == 0 || runs > 64 {
		t.Errorf("the re-arming hook ran %d times, want a small bounded number", runs)
	}

	// The dropped hooks must not leak into the next request.
	before := runs
	e.GET("/plain").Expect().Status(iris.StatusOK).Body().IsEqual("plain")
	if runs != before {
		t.Errorf("the re-arming hook ran again on the next request")
	}
}
