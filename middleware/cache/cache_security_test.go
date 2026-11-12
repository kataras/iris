package cache_test

import (
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cache"
)

func newCacheApp(options cache.Options) *iris.Application {
	app := iris.New()
	app.Use(cache.New(options))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	return app
}

func newStrictCacheApp(options cache.Options) *iris.Application {
	app := iris.New()
	app.Use(cache.Strict(options))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	return app
}

// New applies the options to every request, credentials or not. That is the
// documented contract: "public" is how RFC 9111 authorizes a shared cache to
// store an authenticated response, so a developer who wrote it meant it.
// Pinned so the behaviour is not changed by accident.
func TestCacheNewAppliesOptionsToEveryRequest(t *testing.T) {
	e := httptest.New(t, newCacheApp(cache.DefaultOptions))

	for _, req := range []func() string{
		func() string { return e.GET("/").Expect().Raw().Header.Get("Cache-Control") },
		func() string {
			return e.GET("/").WithHeader("Authorization", "Bearer token").Expect().Raw().Header.Get("Cache-Control")
		},
		func() string {
			return e.GET("/").WithCookie("session", "abc").Expect().Raw().Header.Get("Cache-Control")
		},
	} {
		if cc := req(); !strings.Contains(cc, "public") {
			t.Fatalf("Cache-Control = %q, expected the configured public policy", cc)
		}
	}
}

// Strict keeps the public policy for anonymous traffic and downgrades it the
// moment a request identifies its client.
func TestCacheStrictDowngradesCredentialedRequests(t *testing.T) {
	e := httptest.New(t, newStrictCacheApp(cache.DefaultOptions))

	// Anonymous: the public policy is applied as configured.
	anon := e.GET("/").Expect().Status(httptest.StatusOK)
	if cc := anon.Raw().Header.Get("Cache-Control"); !strings.Contains(cc, "public") {
		t.Fatalf("anonymous Cache-Control = %q, expected the public policy", cc)
	}

	// With an Authorization header: no shared-cache directive at all.
	authed := e.GET("/").WithHeader("Authorization", "Bearer token").
		Expect().Status(httptest.StatusOK)
	cc := authed.Raw().Header.Get("Cache-Control")
	if strings.Contains(cc, "public") || strings.Contains(cc, "s-maxage") {
		t.Fatalf("authenticated Cache-Control = %q, a shared cache may store this response", cc)
	}
	if !strings.Contains(cc, "private") {
		t.Fatalf("authenticated Cache-Control = %q, expected a private policy", cc)
	}
	if exp := authed.Raw().Header.Get("Expires"); exp != "" {
		t.Fatalf("authenticated Expires = %q, expected it dropped", exp)
	}

	// Same for a cookie-bearing request.
	withCookie := e.GET("/").WithCookie("session", "abc").
		Expect().Status(httptest.StatusOK)
	if cc := withCookie.Raw().Header.Get("Cache-Control"); strings.Contains(cc, "public") {
		t.Fatalf("cookie-bearing Cache-Control = %q, a shared cache may store this response", cc)
	}

	// Validators still apply either way.
	if etag := authed.Raw().Header.Get("Etag"); etag == "" {
		t.Error("expected the ETag validator to survive the downgrade")
	}
}

// Strict must not leak one request's downgrade into the next.
func TestCacheStrictDoesNotLeakBetweenRequests(t *testing.T) {
	e := httptest.New(t, newStrictCacheApp(cache.DefaultOptions))

	e.GET("/").WithHeader("Authorization", "Bearer token").Expect().Status(httptest.StatusOK)

	anon := e.GET("/").Expect().Status(httptest.StatusOK)
	if cc := anon.Raw().Header.Get("Cache-Control"); !strings.Contains(cc, "public") {
		t.Fatalf("anonymous Cache-Control = %q after a credentialed request, expected the public policy", cc)
	}
}

// PrivateOptions never declare a shared-cache directive, so they apply to
// credentialed requests too.
func TestCachePrivateOptionsApplyToCredentialedRequests(t *testing.T) {
	e := httptest.New(t, newCacheApp(cache.PrivateOptions))

	resp := e.GET("/").WithHeader("Authorization", "Bearer token").
		Expect().Status(httptest.StatusOK)

	cc := resp.Raw().Header.Get("Cache-Control")
	if !strings.Contains(cc, "private") {
		t.Fatalf("Cache-Control = %q, expected a private policy", cc)
	}
	if strings.Contains(cc, "public") || strings.Contains(cc, "s-maxage") {
		t.Fatalf("Cache-Control = %q, must not permit a shared cache", cc)
	}
}
