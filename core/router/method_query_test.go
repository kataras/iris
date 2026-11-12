package router_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/iris-contrib/httpexpect/v2"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/httptest"
)

type queryFilter struct {
	Term string `json:"term"`
}

func echoQueryFilter(ctx *context.Context) {
	var filter queryFilter
	if err := ctx.ReadJSON(&filter); err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("%s %s", ctx.Method(), filter.Term)
}

func TestRouterQueryMethod(t *testing.T) {
	app := iris.New()
	route := app.Query("/books", echoQueryFilter)
	if route == nil || route.Method != iris.MethodQuery {
		t.Fatalf("expected a QUERY route, got %v", route)
	}

	e := httptest.New(t, app)
	e.Request(iris.MethodQuery, "/books").WithJSON(queryFilter{Term: "go"}).Expect().
		Status(iris.StatusOK).Body().IsEqual("QUERY go")
	e.POST("/books").WithJSON(queryFilter{Term: "go"}).Expect().Status(iris.StatusNotFound)
}

func TestRouterAnyCoversQuery(t *testing.T) {
	if !slices.Contains(router.AllMethods, router.MethodQuery) {
		t.Fatalf("expected AllMethods to contain QUERY, got %v", router.AllMethods)
	}

	app := iris.New()
	registry := app.Any("/books", echoQueryFilter)
	if registry.Query() == nil {
		t.Fatal("expected Any to register a QUERY route")
	}

	e := httptest.New(t, app)
	e.Request(iris.MethodQuery, "/books").WithJSON(queryFilter{Term: "any"}).Expect().
		Status(iris.StatusOK).Body().IsEqual("QUERY any")
}

func TestRouterMethodNotAllowedListsQuery(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})
	app.Handle("POST QUERY", "/books/list", echoQueryFilter)

	e := httptest.New(t, app)
	allow := e.GET("/books/list").Expect().Status(iris.StatusMethodNotAllowed).Header("Allow").Raw()
	for _, method := range []string{iris.MethodPost, iris.MethodQuery} {
		if !strings.Contains(allow, method) {
			t.Fatalf("expected the Allow header to list %s, got %q", method, allow)
		}
	}

	e.Request(iris.MethodQuery, "/books/list").WithJSON(queryFilter{Term: "q"}).Expect().
		Status(iris.StatusOK).Body().IsEqual("QUERY q")
	e.POST("/books/list").WithJSON(queryFilter{Term: "p"}).Expect().
		Status(iris.StatusOK).Body().IsEqual("POST p")
}

// A trailing-slash redirect of a method with a body must be a 307, which keeps
// the method and the body; a 301 lets the client turn it into a bodyless GET.
func TestRouterTrailingSlashRedirectKeepsBody(t *testing.T) {
	app := iris.New()
	app.Handle("GET POST PUT PATCH QUERY", "/books", echoQueryFilter)

	e := httptest.New(t, app, httptest.URL("http://example.com"))
	for _, method := range []string{iris.MethodPost, iris.MethodPut, iris.MethodPatch, iris.MethodQuery} {
		e.Request(method, "/books/").WithRedirectPolicy(httpexpect.DontFollowRedirects).
			Expect().Status(iris.StatusTemporaryRedirect)
	}

	e.GET("/books/").WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().Status(iris.StatusMovedPermanently)
}

type queryMethodController struct{}

func (c *queryMethodController) Query(ctx *context.Context) {
	echoQueryFilter(ctx)
}

func TestRouterControllerQueryMethod(t *testing.T) {
	app := iris.New()
	app.Router("/books", new(queryMethodController))

	e := httptest.New(t, app)
	e.Request(iris.MethodQuery, "/books").WithJSON(queryFilter{Term: "ctrl"}).Expect().
		Status(iris.StatusOK).Body().IsEqual("QUERY ctrl")
}
