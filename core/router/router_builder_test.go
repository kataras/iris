package router_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestRouterBuilderEditContextOptions_SetCharset_Upper(t *testing.T) {
	app := iris.New()
	app.ContextOptions().Charset = "ISO-8859-1"

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hi</h1>")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).HasContentType("text/html", "iso-8859-1").Body().IsEqual("<h1>Hi</h1>")
}

func TestRouterBuilderGetRootNested(t *testing.T) {
	// Regression test for the GetRoot infinite-loop bug. Three-or-more levels
	// of nesting previously caused GetRoot to spin forever; this test would
	// hang the goroutine in that case.
	app := iris.New()
	root := app.Router("/")
	child := root.Router("/child")
	grandchild := child.Router("/grand")
	greatgrand := grandchild.Router("/great")

	done := make(chan iris.Router, 1)
	go func() { done <- greatgrand.GetRoot() }()

	select {
	case got := <-done:
		if !got.IsRoot() {
			t.Fatalf("GetRoot did not return the root RouterBuilder (IsRoot=false)")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("GetRoot hung — likely infinite loop reintroduced")
	}
}

func TestRouterBuilderEnableStatusCodeResponse(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusBadRequest) // no body.
	})

	child := app.Router("/child")
	child.EnableStatusCodeResponse()
	child.Get("/", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNotFound)
	})

	childOfChild := child.Router("/child") // EnableStatusCodeResponse is inherited.
	childOfChild.Get("/", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNotFound)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusBadRequest).Body().IsEmpty()
	e.GET("/child").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
	e.GET("/child/child").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
}
