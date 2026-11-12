package geolocation_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/geolocation"
)

func TestNew(t *testing.T) {
	// Stand up a local stand-in for ip-api.com so the test is hermetic and does
	// not depend on the live third-party service (or its rate limits).
	provider := nethttptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","country":"Greece","countryCode":"GR","query":"31.152.143.115"}`))
	}))
	defer provider.Close()

	app := iris.New()
	app.ContextOptions().RemoteAddrHeaders = []string{"X-Real-IP"}
	app.Use(geolocation.New(geolocation.Options{
		LocationClient:  client.New(client.BaseURL(provider.URL)),
		DisableLanguage: true,
		DisableCache:    true, // avoid cross-test cache reuse; each request hits the mock.
	}))

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(geolocation.Get(ctx))
	})

	e := httptest.New(t, app, httptest.Debug(false))

	e.GET("/").WithHeader("X-Real-IP", "31.152.143.115").
		Expect().Status(httptest.StatusOK).JSON().Object().
		ContainsKey("status").HasValue("status", "success").
		ContainsKey("countryCode").HasValue("countryCode", "GR")
}

// The zero-value Options keep the default in-memory cache, so a second request
// from the same IP is answered without touching the provider again.
func TestNewDefaultCacheHitsTheProviderOnce(t *testing.T) {
	var hits atomic.Int32

	provider := nethttptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","country":"Greece","countryCode":"GR","query":"31.152.143.115"}`))
	}))
	defer provider.Close()

	app := iris.New()
	app.ContextOptions().RemoteAddrHeaders = []string{"X-Real-IP"}
	app.Use(geolocation.New(geolocation.Options{
		// Only the location client and the language lookup are overridden:
		// Cache is left at its default, which is the point of this test.
		LocationClient:  client.New(client.BaseURL(provider.URL)),
		DisableLanguage: true,
	}))

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(geolocation.Get(ctx))
	})

	e := httptest.New(t, app, httptest.Debug(false))

	for range 3 {
		e.GET("/").WithHeader("X-Real-IP", "31.152.143.115").
			Expect().Status(httptest.StatusOK).JSON().Object().
			HasValue("countryCode", "GR")
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("the provider was called %d times, want 1 (the default cache should serve the rest)", got)
	}
}
