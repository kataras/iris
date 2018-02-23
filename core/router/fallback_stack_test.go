package router_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/httptest"
)

func TestFallbackStackAdd(t *testing.T) {
	l := make([]string, 0)

	stk := &router.FallbackStack{}
	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS1")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS2")
		},
	})

	if stk.Size() != 2 {
		t.Fatalf("Bad size (%d != 2)", stk.Size())
	}

	for _, h := range stk.List() {
		h(nil)
	}

	if (l[0] != "POS2") || (l[1] != "POS1") {
		t.Fatal("Bad positions: ", l)
	}
}

func TestFallbackStackFork(t *testing.T) {
	l := make([]string, 0)

	stk := &router.FallbackStack{}

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS1")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS2")
		},
	})

	stk = stk.Fork()

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS3")
		},
	})

	stk.Add(context.Handlers{
		func(context.Context) {
			l = append(l, "POS4")
		},
	})

	if stk.Size() != 4 {
		t.Fatalf("Bad size (%d != 4)", stk.Size())
	}

	for _, h := range stk.List() {
		h(nil)
	}

	if (l[0] != "POS4") || (l[1] != "POS3") || (l[2] != "POS2") || (l[3] != "POS1") {
		t.Fatal("Bad positions: ", l)
	}
}

func TestFallbackStackCall(t *testing.T) {
	// build the api
	app := iris.New()

	// setup an existing routes
	app.Handle("GET", "/route", func(ctx context.Context) {
		ctx.WriteString("ROUTED")
	})

	// setup fallback handler
	app.Fallback(func(ctx context.Context) {
		if ctx.Method() != "GET" {
			ctx.NextOrNotFound() //	it checks if we have next, otherwise fire 404 not found.
			return
		}

		ctx.WriteString("FALLBACK")
	})

	// run the tests
	e := httptest.New(t, app, httptest.Debug(false))

	e.Request("GET", "/route").Expect().Status(iris.StatusOK).Body().Equal("ROUTED")
	e.Request("POST", "/route").Expect().Status(iris.StatusNotFound)
	e.Request("POST", "/noroute").Expect().Status(iris.StatusNotFound)
	e.Request("GET", "/noroute").Expect().Status(iris.StatusOK).Body().Equal("FALLBACK")
}
