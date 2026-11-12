package cors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cors"
)

func TestDefaultCORS(t *testing.T) {
	app := iris.New()
	app.UseRouter(cors.New().Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.Text("Hello, World!")
	})

	e := httptest.New(t, app)
	tt := e.GET("/").WithHeader("Origin", "http://example.com").
		Expect().Status(httptest.StatusOK)
	tt.Header("Access-Control-Allow-Origin").IsEqual("http://example.com")
	tt.Body().IsEqual("Hello, World!")
	tt.HasContentType("text/plain", "utf-8")
	// A specific origin advertises credentials by default.
	tt.Header("Access-Control-Allow-Credentials").IsEqual("true")
}

// TestCORSCredentialsWithWildcardOrigin verifies that "Access-Control-Allow-Credentials: true"
// is suppressed alongside a wildcard ("*") origin by default (the browser-rejected
// combination), but is advertised when the developer explicitly opts in with
// AllowCredentials.
func TestCORSCredentialsWithWildcardOrigin(t *testing.T) {
	newApp := func(c *cors.CORS) *iris.Application {
		app := iris.New()
		app.UseRouter(c.Handler())
		app.Get("/", func(ctx iris.Context) { ctx.Text("ok") })
		return app
	}

	// No Origin header -> wildcard "*" origin; credentials suppressed by default.
	resp := httptest.New(t, newApp(cors.New())).GET("/").Expect().Status(httptest.StatusOK)
	resp.Header("Access-Control-Allow-Origin").IsEqual("*")
	if got := resp.Raw().Header.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("default wildcard origin: Allow-Credentials should be absent, got %q", got)
	}

	// AllowCredentials() opts in: credentials advertised even with a wildcard origin.
	resp2 := httptest.New(t, newApp(cors.New().AllowCredentials())).GET("/").Expect().Status(httptest.StatusOK)
	resp2.Header("Access-Control-Allow-Origin").IsEqual("*")
	resp2.Header("Access-Control-Allow-Credentials").IsEqual("true")
}

func TestCORSPreflightAllowsQuery(t *testing.T) {
	app := iris.New()
	app.UseRouter(cors.New().Handler())
	app.Query("/books", func(ctx iris.Context) {})

	e := httptest.New(t, app)
	e.OPTIONS("/books").
		WithHeader("Origin", "https://example.com").
		WithHeader("Access-Control-Request-Method", "QUERY").
		Expect().Status(httptest.StatusNoContent).
		Header("Access-Control-Allow-Methods").Contains("QUERY")
}
