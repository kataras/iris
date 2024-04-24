package cache_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/kataras/iris/v12/cache"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
)

func TestNoCache(t *testing.T) {
	app := iris.New()
	app.Get("/", cache.NoCache, func(ctx iris.Context) {
		ctx.WriteString("no_cache")
	})

	// tests
	e := httptest.New(t, app)

	r := e.GET("/").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("no_cache")
	r.Header(context.CacheControlHeaderKey).Equal(cache.CacheControlHeaderValue)
	r.Header(cache.PragmaHeaderKey).Equal(cache.PragmaNoCacheHeaderValue)
	r.Header(cache.ExpiresHeaderKey).Equal(cache.ExpiresNeverHeaderValue)
}

func TestStaticCache(t *testing.T) {
	// test change the time format, which is not recommended but can be done.
	app := iris.New().Configure(iris.WithTimeFormat("02 Jan 2006 15:04:05 GMT"))

	cacheDur := 30 * (24 * time.Hour)
	var expectedTime time.Time
	app.Get("/", cache.StaticCache(cacheDur), func(ctx iris.Context) {
		expectedTime = time.Now()
		ctx.WriteString("static_cache")
	})

	// tests
	e := httptest.New(t, app)
	r := e.GET("/").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("static_cache")

	r.Header(cache.ExpiresHeaderKey).Equal(expectedTime.Add(cacheDur).Format(app.ConfigurationReadOnly().GetTimeFormat()))
	cacheControlHeaderValue := "public, max-age=" + strconv.Itoa(int(cacheDur.Seconds()))
	r.Header(context.CacheControlHeaderKey).Equal(cacheControlHeaderValue)
}

func TestCache304(t *testing.T) {
	// t.Parallel()
	app := iris.New()

	expiresEvery := 4 * time.Second
	app.Get("/", cache.Cache304(expiresEvery), func(ctx iris.Context) {
		ctx.WriteString("send")
	})
	// handlers
	e := httptest.New(t, app)

	// when 304, content type, content length and if ETagg is there are removed from the headers.
	insideCacheTimef := time.Now().Add(-expiresEvery).UTC().Format(app.ConfigurationReadOnly().GetTimeFormat())
	r := e.GET("/").WithHeader(context.IfModifiedSinceHeaderKey, insideCacheTimef).Expect().Status(httptest.StatusNotModified)
	r.Headers().NotContainsKey(context.ContentTypeHeaderKey).NotContainsKey(context.ContentLengthHeaderKey).NotContainsKey("ETag")
	r.Body().IsEqual("")

	// continue to the handler itself.
	cacheInvalidatedTimef := time.Now().Add(expiresEvery).UTC().Format(app.ConfigurationReadOnly().GetTimeFormat()) // after ~5seconds.
	r = e.GET("/").WithHeader(context.LastModifiedHeaderKey, cacheInvalidatedTimef).Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("send")
	// now without header, it should continue to the handler itself as well.
	r = e.GET("/").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("send")
}

func TestETag(t *testing.T) {
	// t.Parallel()

	app := iris.New()
	n := "_"
	app.Get("/", cache.ETag, func(ctx iris.Context) {
		ctx.WriteString(n)
		n += "_"
	})

	// the first and last test writes the content with status OK without cache,
	// the rest tests the cache headers and status 304 and return, so body should be "".
	e := httptest.New(t, app)

	r := e.GET("/").Expect().Status(httptest.StatusOK)
	r.Header("ETag").Equal("/") // test if header set.
	r.Body().IsEqual("_")

	e.GET("/").WithHeader("ETag", "/").WithHeader("If-None-Match", "/").Expect().
		Status(httptest.StatusNotModified).Body().IsEqual("") // browser is responsible, no the test engine.

	r = e.GET("/").Expect().Status(httptest.StatusOK)
	r.Header("ETag").Equal("/") // test if header set.
	r.Body().IsEqual("__")
}
