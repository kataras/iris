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
// It can be overriden.
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
// simillary to the `Party#StaticWeb` (which sends status OK(200) and browser disk caching instead of 304).
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
