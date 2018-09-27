package cache

import (
	"strconv"
	"time"

	"github.com/kataras/iris/cache/client"
	"github.com/kataras/iris/context"
)

// CacheControlHeaderValue is the header value of the
// "Cache-Control": "private, no-cache, max-age=0, must-revalidate, no-store, proxy-revalidate, s-maxage=0".
//
// It can be overridden.
var CacheControlHeaderValue = "private, no-cache, max-age=0, must-revalidate, no-store, proxy-revalidate, s-maxage=0"

const (
	// PragmaHeaderKey is the header key of "Pragma".
	PragmaHeaderKey = "Pragma"
	// PragmaNoCacheHeaderValue is the header value of "Pragma": "no-cache".
	PragmaNoCacheHeaderValue = "no-cache"
	// ExpiresHeaderKey is the header key of "Expires".
	ExpiresHeaderKey = "Expires"
	// ExpiresNeverHeaderValue is the header value of "ExpiresHeaderKey": "0".
	ExpiresNeverHeaderValue = "0"
)

// NoCache is a middleware which overrides the Cache-Control, Pragma and Expires headers
// in order to disable the cache during the browser's back and forward feature.
//
// A good use of this middleware is on HTML routes; to refresh the page even on "back" and "forward" browser's arrow buttons.
//
// See `cache#StaticCache` for the opposite behavior.
var NoCache = func(ctx context.Context) {
	ctx.Header(context.CacheControlHeaderKey, CacheControlHeaderValue)
	ctx.Header(PragmaHeaderKey, PragmaNoCacheHeaderValue)
	ctx.Header(ExpiresHeaderKey, ExpiresNeverHeaderValue)
	// Add the X-No-Cache header as well, for any customized case, i.e `cache#Handler` or `cache#Cache`.
	client.NoCache(ctx)

	ctx.Next()
}

// StaticCache middleware for caching static files by sending the "Cache-Control" and "Expires" headers to the client.
// It accepts a single input parameter, the "cacheDur", a time.Duration that it's used to calculate the expiration.
//
// If "cacheDur" <=0 then it returns the `NoCache` middleware instaed to disable the caching between browser's "back" and "forward" actions.
//
// Usage: `app.Use(cache.StaticCache(24 * time.Hour))` or `app.Use(cache.Staticcache(-1))`.
// A middleware, which is a simple Handler can be called inside another handler as well, example:
// cacheMiddleware := cache.StaticCache(...)
// func(ctx iris.Context){
//  cacheMiddleware(ctx)
//  [...]
// }
var StaticCache = func(cacheDur time.Duration) context.Handler {
	if int64(cacheDur) <= 0 {
		return NoCache
	}

	cacheControlHeaderValue := "public, max-age=" + strconv.Itoa(int(cacheDur.Seconds()))
	return func(ctx context.Context) {
		cacheUntil := time.Now().Add(cacheDur).Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
		ctx.Header(ExpiresHeaderKey, cacheUntil)
		ctx.Header(context.CacheControlHeaderKey, cacheControlHeaderValue)

		ctx.Next()
	}
}

const ifNoneMatchHeaderKey = "If-None-Match"

// ETag is another browser & server cache request-response feature.
// It can be used side by side with the `StaticCache`, usually `StaticCache` middleware should go first.
// This should be used on routes that serves static files only.
// The key of the `ETag` is the `ctx.Request().URL.Path`, invalidation of the not modified cache method
// can be made by other request handler as well.
//
// In typical usage, when a URL is retrieved, the web server will return the resource's current
// representation along with its corresponding ETag value,
// which is placed in an HTTP response header "ETag" field:
//
// ETag: "/mypath"
//
// The client may then decide to cache the representation, along with its ETag.
// Later, if the client wants to retrieve the same URL resource again,
// it will first determine whether the local cached version of the URL has expired
// (through the Cache-Control (`StaticCache` method) and the Expire headers).
// If the URL has not expired, it will retrieve the local cached resource.
// If it determined that the URL has expired (is stale), then the client will contact the server
// and send its previously saved copy of the ETag along with the request in a "If-None-Match" field.
//
// Usage with combination of `StaticCache`:
// assets := app.Party("/assets", cache.StaticCache(24 * time.Hour), ETag)
// assets.StaticWeb("/", "./assets") or StaticEmbedded("/", "./assets") or StaticEmbeddedGzip("/", "./assets").
//
// Similar to `Cache304` but it doesn't depends on any "modified date", it uses just the ETag and If-None-Match headers.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching and
// https://en.wikipedia.org/wiki/HTTP_ETag
var ETag = func(ctx context.Context) {
	key := ctx.Request().URL.Path
	ctx.Header(context.ETagHeaderKey, key)
	if match := ctx.GetHeader(ifNoneMatchHeaderKey); match == key {
		ctx.WriteNotModified()
		return
	}
	ctx.Next()
}

// Cache304 sends a `StatusNotModified` (304) whenever
// the "If-Modified-Since" request header (time) is before the
// time.Now() + expiresEvery (always compared to their UTC values).
// Use this `cache#Cache304` instead of the "github.com/kataras/iris/cache" or iris.Cache
// for better performance.
// Clients that are compatible with the http RCF (all browsers are and tools like postman)
// will handle the caching.
// The only disadvantage of using that instead of server-side caching
// is that this method will send a 304 status code instead of 200,
// So, if you use it side by side with other micro services
// you have to check for that status code as well for a valid response.
//
// Developers are free to extend this method's behavior
// by watching system directories changes manually and use of the `ctx.WriteWithExpiration`
// with a "modtime" based on the file modified date,
// can be used on Party's that contains a static handler,
// i.e `StaticWeb`, `StaticEmbedded` or even `StaticEmbeddedGzip`.
var Cache304 = func(expiresEvery time.Duration) context.Handler {
	return func(ctx context.Context) {
		now := time.Now()
		if modified, err := ctx.CheckIfModifiedSince(now.Add(-expiresEvery)); !modified && err == nil {
			ctx.WriteNotModified()
			return
		}

		ctx.SetLastModified(now)
		ctx.Next()
	}
}
