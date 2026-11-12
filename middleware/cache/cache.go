// Package cache sets client- and proxy-cache headers (Cache-Control, ETag,
// Last-Modified, Expires, Pragma, Vary, Proxy-Authenticate) on the response.
//
// Its entry point is New(Options), which returns a Handler and cannot fail.
// `Strict` is New with one extra rule for credentialed requests; read the
// SECURITY WARNING on DefaultOptions before registering either one globally.
// For a server-side response store see the sibling cache/httpcache package.
package cache

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

func init() {
	context.SetHandlerName("iris/middleware/cache.*", "iris.cache")
}

const (
	// NoPragmaCache is an HTTP/1.0 header used for backwards compatibility, instructing caches not to store the response.
	NoPragmaCache string = "no-cache"
)

// Options holds configuration options for the caching middleware.
type Options struct {
	// Whether to use ETag headers for cache validation.
	ETag bool
	// The Cache-Control directives to include in the response headers.
	// Use the CacheControlBuilder to construct this value.
	CacheControl string
	// Whether to use Last-Modified headers for cache validation.
	LastModified bool
	// Duration for the Expires header.
	Expires time.Duration
	// The Pragma directive to include in the response headers,
	// e.g.  NoPragmaCache.
	Pragma string
	// The Vary header specifies how caches should treat the response based on request headers.
	// Use the VaryBuilder to construct this value.
	Vary string
	// The Proxy-Authenticate header is used to specify the authentication method
	// that should be used to access a resource behind a proxy server.
	// Use the ProxyAuthenticateBuilder to construct this value, e.g.
	// NewProxyAuthenticateBuilder().Basic("Restricted Area").Build()
	ProxyAuthenticate string
}

// DefaultOptions are the default options for the caching middleware.
// These options are used when the middleware is created without any options.
// The default options are:
// - ETag: true
// - CacheControl: public, max-age=3600, s-maxage=7200, must-revalidate, proxy-revalidate
// - LastModified: true
// - Expires: 24 hours
// - Pragma: no-cache
// - Vary: Accept-Encoding
// To create a new middleware with the default options, use:
//
//	app.Use(cache.New(cache.DefaultOptions))
//
// SECURITY WARNING: these defaults declare every response "public" and list only
// Accept-Encoding in Vary.
//
// "public" is not decoration. Per RFC 9111 it is the directive that authorizes a
// shared cache - a CDN, a corporate proxy, nginx proxy_cache - to store a
// response even though the request carried credentials. Applied broadly, which
// is what app.Use does, that means an authenticated /account response can be
// stored under a URL-only key for an hour and served to the next visitor.
//
// Register this per route, on the routes you know are the same for everybody.
// For a route that serves both anonymous and authenticated callers, use
// `Strict`, which keeps this policy for the former and downgrades it for the
// latter. For a route that is always user-specific, use `PrivateOptions`.
var DefaultOptions = Options{
	ETag:         true,
	CacheControl: NewCacheControlBuilder().Public().MaxAge(time.Hour).SMaxAge(2 * time.Hour).MustRevalidate().ProxyRevalidate().Build(),
	LastModified: true,
	Expires:      24 * time.Hour,
	Pragma:       NoPragmaCache,
	Vary:         NewVaryBuilder().AcceptEncoding().Build(),
}

// PrivateOptions are DefaultOptions with a private, shorter-lived policy: a
// response may be stored by the client's own browser but never by a shared
// cache.
//
// Prefer these whenever responses can be user-specific.
var PrivateOptions = Options{
	ETag:         true,
	CacheControl: NewCacheControlBuilder().Private().MaxAge(0).MustRevalidate().Build(),
	LastModified: true,
	Pragma:       NoPragmaCache,
	Vary:         NewVaryBuilder().AcceptEncoding().Build(),
}

// New returns a new Iris middleware that sets caching headers based on the provided options.
//
// The headers are set for every request that reaches it, so register it on the
// routes whose responses are the same for every client. See the SECURITY
// WARNING on DefaultOptions.
//
// Example:
//
//	app.Use(cache.New(cache.Options{
//		ETag:         true,
//		CacheControl: cache.NewCacheControlBuilder().Public().MaxAge(time.Hour).SMaxAge(2 * time.Hour).MustRevalidate().ProxyRevalidate().Build(),
//		LastModified: true,
//		Expires:      24 * time.Hour,
//		Pragma:       "no-cache",
//		Vary:         cache.NewVaryBuilder().AcceptEncoding().Build(),
//	}))
func New(options Options) context.Handler {
	return newHandler(options, false)
}

// Strict is New with one extra rule: a request that carries credentials
// (`Authorization`, `Proxy-Authorization` or `Cookie`) never receives a
// shared-cache directive.
//
// `New` applies your options to every request, which is correct when the
// response is the same for everybody. It is the wrong answer when it is not:
// `public` and `s-maxage` are precisely what authorize a CDN or a corporate
// proxy to store a response that was fetched with credentials and hand it to
// the next visitor.
//
// Strict lets you keep an aggressive shared-cache policy for anonymous traffic
// and fall back to a private one the moment a request identifies its client:
//
//	app.Use(cache.Strict(cache.DefaultOptions))
//
// Validators (`ETag`, `Last-Modified`) and `Vary` are applied either way; only
// the Cache-Control and Expires policy is downgraded. Use it when one route
// serves both anonymous and authenticated callers. When a route is public for
// everyone, `New` is the better fit and keeps the shared cache working.
func Strict(options Options) context.Handler {
	return newHandler(options, true)
}

// privateCacheControl is the policy Strict substitutes for a credentialed
// request: the client may hold its own copy, no shared cache may.
var privateCacheControl = NewCacheControlBuilder().Private().MaxAge(0).MustRevalidate().Build()

func newHandler(options Options, strict bool) context.Handler {
	return func(ctx *context.Context) {
		// Note(@kataras): the per-request policy is a LOCAL copy.
		//
		// "options" is captured by this closure and shared by every request, so
		// assigning to its fields here would make the first credentialed request
		// downgrade the middleware permanently, for everybody. It is also a data
		// race. See TestCacheStrictDoesNotLeakBetweenRequests.
		cacheControl, expires := options.CacheControl, options.Expires
		if strict && isCredentialedRequest(ctx) {
			// Downgrade rather than skip: the validators below still save
			// bandwidth, and the client's own cache is not the problem.
			cacheControl, expires = privateCacheControl, 0
		}

		// Handle ETag and If-None-Match headers for cache validation.
		if options.ETag {
			etag := ctx.Header("If-None-Match")
			currentETag := fmt.Sprintf("\"%s\"", strconv.Itoa(int(time.Now().Unix())))
			ctx.SetHeader(context.ETagHeaderKey, currentETag)
			if etag == currentETag && isConditionalMethod(ctx.Method()) {
				writeNotModified(ctx)
				return
			}
		}

		// Set Cache-Control header.
		if v := cacheControl; v != "" {
			ctx.SetHeader(context.CacheControlHeaderKey, v)
		}

		// Handle Last-Modified and If-Modified-Since headers for cache validation.
		if options.LastModified {
			lastMod := time.Now().Add(-24 * time.Hour) // TODO: add an option to get the actual last modified or option to set.

			lastModValue := lastMod.Format(http.TimeFormat)
			ctx.SetHeader(context.LastModifiedHeaderKey, lastModValue)
			if ctx.Header(context.IfModifiedSinceHeaderKey) != "" && isConditionalMethod(ctx.Method()) {
				if ctx.Header(context.IfModifiedSinceHeaderKey) == lastModValue {
					writeNotModified(ctx)
					return
				}
			}
		}

		// Set Expires header.
		if v := expires; v > 0 {
			ctx.SetHeader("Expires", time.Now().Add(v).Format(http.TimeFormat))
		}

		// Set Pragma headers.
		if v := options.Pragma; v != "" {
			ctx.SetHeader("Pragma", v)
		}

		// Set Vary header.
		if v := options.Vary; v != "" {
			ctx.AddHeader(context.VaryHeaderKey, v)
		}

		if v := options.ProxyAuthenticate; v != "" {
			ctx.SetHeader("Proxy-Authenticate", v)
		}

		// Continue to the next handler.
		ctx.Next()
	}
}

// isCredentialedRequest reports whether the request carries anything that makes
// its response client-specific.
func isCredentialedRequest(ctx *context.Context) bool {
	h := ctx.Request().Header
	return h.Get("Authorization") != "" || h.Get("Proxy-Authorization") != "" || h.Get("Cookie") != ""
}

func writeNotModified(ctx *context.Context) {
	h := ctx.ResponseWriter().Header()
	// Note(@kataras): Header().Del, not delete() on the raw map.
	//
	// ctx.SetHeader goes through Header().Add, which canonicalises "ETag" to
	// "Etag", so the raw-map delete with the "ETag" spelling never matched and
	// the validator was left on a 304 response. Content-Type and Content-Length
	// are already canonical, which is why only this one was affected.
	h.Del(context.ContentTypeHeaderKey)
	h.Del(context.ContentLengthHeaderKey)
	h.Del(context.ETagHeaderKey)
	ctx.StatusCode(http.StatusNotModified)
}

// isConditionalMethod reports whether a request of "method" may be answered
// with 304 Not Modified. RFC 10008 makes QUERY responses cacheable too.
func isConditionalMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == router.MethodQuery
}
