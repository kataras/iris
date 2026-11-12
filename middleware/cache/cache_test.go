package cache_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cache"
)

func TestCache(t *testing.T) {
	// Setup the middleware with options
	app := iris.New()
	options := cache.DefaultOptions
	app.Use(cache.New(options))

	// Create a test handler.
	app.Get("/foo", func(ctx iris.Context) {
		ctx.WriteString("Hello, world!")
	})

	// Call the handler with the test request and response recorder.
	resp := httptest.New(t, app).GET("/foo").Expect().Status(http.StatusOK)

	// Verify Cache-Control header.
	if got, want := resp.Header("Cache-Control").Raw(), options.CacheControl; got != want {
		t.Errorf("Cache-Control header = %q, want %q", got, want)
	}

	// Verify Pragma header.
	if got, want := resp.Header("Pragma").Raw(), options.Pragma; got != want {
		t.Errorf("Pragma header = %q, want %q", got, want)
	}

	// Verify Vary header.
	if got, want := resp.Header("Vary").Raw(), options.Vary; got != want {
		t.Errorf("Vary header = %q, want %q", got, want)
	}

	// TODO: when customize the last modified option:
	// Verify Last-Modified header.
	// lastMod := options.LastModified.Format(http.TimeFormat)
	// if got, want := resp.Header("Last-Modified"), lastMod; got != want {
	// 	t.Errorf("Last-Modified header = %q, want %q", got, want)
	// }

	// Verify ETag header.
	if got := resp.Header("ETag").Raw(); got == "" {
		t.Errorf("Expected ETag header to be set")
	}

	// Verify Expires header.
	expires := time.Now().Add(options.Expires).Format(http.TimeFormat)
	if got, want := resp.Header("Expires").Raw(), expires; got != want {
		t.Errorf("Expires header = %q, want %q", got, want)
	}

	// Verify Proxy-Authenticate header.
	if got, want := resp.Header("Proxy-Authenticate").Raw(), options.ProxyAuthenticate; got != want {
		t.Errorf("Proxy-Authenticate header = %q, want %q", got, want)
	}
}

// QUERY responses are cacheable (RFC 10008), so a conditional QUERY is answered
// with 304 like a GET, while a POST to the same resource is not.
func TestCacheNotModifiedOnQuery(t *testing.T) {
	app := iris.New()
	app.Use(cache.New(cache.Options{LastModified: true}))
	app.Handle("POST QUERY", "/books/list", func(ctx iris.Context) {
		ctx.WriteString("books")
	})

	e := httptest.New(t, app)
	lastModified := e.Request(iris.MethodQuery, "/books/list").Expect().Status(http.StatusOK).Header("Last-Modified").Raw()

	e.Request(iris.MethodQuery, "/books/list").WithHeader("If-Modified-Since", lastModified).
		Expect().Status(http.StatusNotModified)
	e.POST("/books/list").WithHeader("If-Modified-Since", lastModified).
		Expect().Status(http.StatusOK).Body().IsEqual("books")
}
