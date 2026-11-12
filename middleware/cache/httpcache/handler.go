package httpcache

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/recorder"
)

const handlerName = "iris.httpcache"

func init() {
	context.SetHandlerName("iris/middleware/cache/httpcache.*.handler*", handlerName)
}

// resourceGetMethods are the cacheable methods. QUERY is the RFC 10008 way to list with a filter body;
// POST stays for clients that cannot send QUERY yet. The cache key hashes the body of both.
var resourceGetMethods = []string{"GET", router.MethodQuery, "POST"}

// Pair returns two Iris middleware handlers:
// 1. A cache handler that caches HTTP responses for GET, QUERY and POST requests.
// 2. An invalidate handler that invalidates the cache for the specific routes
// that the cache handler was registered on.
// Read more about the cache handler in the `Handler` method.
// Read more about the invalidate handler in the `InvalidateRoutes` method.
func (c *Cache) Pair(r router.Router) (context.Handler, context.Handler) {
	if c.Options.Disable {
		return nil, nil
	}

	cacheHandler := c.Handler()
	invalidateHandler := c.InvalidateRoutes(r)

	return cacheHandler, invalidateHandler
}

// Handler is an Iris middleware handler that caches HTTP responses.
// It uses the provided Cache instance to store and retrieve cached responses.
// The handler checks the request method and only caches responses for GET and POST requests.
// If a cached response is found, it writes it directly to the context response.
// If no cached response is found, it records the response using a response recorder.
// After the handler chain is executed, if the response status code is 200,
// it stores the response body in the cache using a key derived from the request.
// The cache key is generated based on the request's URL path, method, query parameters,
// headers, cookies and body.
//
// The cache is designed to store only successful responses (HTTP 200) for GET and POST requests.
func (c *Cache) Handler() context.Handler {
	if c.Options.Disable {
		// Return a nil handler if caching is disabled. The framework will skip this middleware.
		return nil
	}

	// This is a method of the Cache struct, so it can be used as a NAMED handler.
	return c.handler
}

func (c *Cache) handler(ctx *context.Context) {
	// if cache.options.Disable {
	// 	// Return if caching is disabled. The framework will skip this middleware.
	// 	return
	// }

	if !slices.Contains(resourceGetMethods, ctx.Method()) {
		// If the method is not GET, skip caching.
		ctx.Next()
		return
	}

	// Note(@kataras): never serve or store a response for a credentialed request
	// unless the operator told us how to key it.
	//
	// The cache key is built from method, path, query and body. Unless
	// Options.Headers or Options.Cookies name the credential, user A's
	// "GET /api/me" and user B's identical request collapse onto the SAME key -
	// so B (or an anonymous caller) is served A's private response. That is a
	// cross-user data leak, and the default configuration has it.
	if c.requestIsCredentialed(ctx) && !c.keyIncludesCredentials() {
		c.warnCredentialedOnce(ctx)
		ctx.Next()
		return
	}

	// Generate the cache key based on the route or request.
	cacheKey := c.ContextKey(ctx)
	// Try to get the cached response.
	cachedResponse, ok, err := c.Get(ctx, cacheKey)
	if err != nil {
		// Fail OPEN: a transient store outage must not take the site down.
		//
		// This used to SetErr and return without calling Next or stopping, which
		// answered an empty 200 to every request for as long as the store was
		// unhealthy.
		//
		// The error is logged and not stored on the Context. A stored error is
		// what an error handler renders for the request, so any failure the
		// route answered next (a 403, a 404) went out with the store's failure
		// text as its message.
		ctx.Application().Logger().Errorf("httpcache: the store failed, serving %s %s uncached: %v", ctx.Method(), ctx.Path(), err)
		ctx.Next()
		return
	}
	if ok {
		cachedResponse.WriteTo(ctx.ResponseWriter())
		return
	}

	// Acquire a response recorder. If it's new set it and release it by this handler,
	// otherwise another handler initialized a response recorder and we should not release it here.
	previous := ctx.ResponseWriter()
	rr, ok := recorder.AcquireResponseRecorder(previous)
	if ok {
		// Release the captured response when the whole handler chain is done,
		// and put the previous writer back on the Context first: a handler
		// registered outside this one reads the Context's writer after the
		// chain returns, and a released recorder may already belong to another
		// request.
		defer func() {
			ctx.SetResponseWriter(previous)
			recorder.ReleaseResponseRecorder(rr)
		}()
		// Set the response writer to the response recorder.
		ctx.SetResponseWriter(rr)
	}

	// Call the next handler.
	ctx.Next()

	if rr.Code == 200 {
		headers := ctx.ResponseWriter().Header()

		// A response that issues a cookie is per-client by definition: caching it
		// would hand that session to every later cache hit.
		if len(headers[setCookieHeaderKey]) > 0 || len(headers[setCookie2HeaderKey]) > 0 {
			return
		}

		resp := Response{
			Body: rr.GetBody(),
			// Only cache headers that are safe to replay to a different client.
			Headers: cacheableHeaders(headers),
		}
		err = c.Set(ctx, cacheKey, resp)
		if err != nil {
			ctx.SetErrPrivate(err) // a store error is never a message for the client.
			return
		}
	}
}

const (
	authorizationHeaderKey      = "Authorization"
	proxyAuthorizationHeaderKey = "Proxy-Authorization"
	cookieHeaderKey             = "Cookie"
	setCookieHeaderKey          = "Set-Cookie"
	setCookie2HeaderKey         = "Set-Cookie2"
)

// uncacheableHeaders are per-connection or per-client headers that must never be
// stored and replayed to another client.
//
// Set-Cookie is the dangerous one: a cached 200 that happened to issue a fresh
// session cookie used to hand that session to every subsequent cache hit.
var uncacheableHeaders = []string{
	setCookieHeaderKey,
	setCookie2HeaderKey,
	authorizationHeaderKey,
	proxyAuthorizationHeaderKey,
	// Hop-by-hop headers, per RFC 7230 section 6.1.
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// cacheableHeaders returns a copy of "headers" without the per-client ones.
func cacheableHeaders(headers http.Header) http.Header {
	out := make(http.Header, len(headers))
	maps.Copy(out, headers)

	for _, key := range uncacheableHeaders {
		out.Del(key)
	}

	return out
}

// requestIsCredentialed reports whether the request carries anything that makes
// its response client-specific.
func (c *Cache) requestIsCredentialed(ctx *context.Context) bool {
	r := ctx.Request()
	return r.Header.Get(authorizationHeaderKey) != "" ||
		r.Header.Get(proxyAuthorizationHeaderKey) != "" ||
		r.Header.Get(cookieHeaderKey) != ""
}

// keyIncludesCredentials reports whether the configured cache key distinguishes
// one authenticated client from another.
func (c *Cache) keyIncludesCredentials() bool {
	return len(c.Options.Headers) > 0 || len(c.Options.Cookies) > 0
}

// warnCredentialedOnce logs, once, that credentialed requests are being skipped.
func (c *Cache) warnCredentialedOnce(ctx *context.Context) {
	c.warnOnce.Do(func() {
		ctx.Application().Logger().Warn(
			"httpcache: requests carrying Authorization or Cookie are NOT cached, because the cache key " +
				"does not distinguish clients. Set Options.Headers (e.g. []string{\"Authorization\"}) or " +
				"Options.Cookies to cache them safely.")
	})
}

var (
	resourceMutationMethods = []string{
		"DELETE",
		"POST",
		"PUT",
		"PATCH",
	}

	resourceMutationStatusCodes = []int{
		200, // OK
		201, // Created
		204, // No Content
	}
)

// InvalidateHandler returns an Iris middleware handler that invalidates the cache
// after a resource mutation operation (DELETE, POST, PUT, PATCH).
// It checks the request method and status code to determine if cache invalidation is necessary.
// If the status code is 200, 201, or 204 and the method is one of DELETE, POST, PUT, or PATCH,
// it invalidates the cache entries for the specified prefixes.
// If no prefixes are provided, it invalidates all cache entries.
// If an error occurs during invalidation, it sets the error in the context.
func (c *Cache) InvalidateHandler(prefixes ...string) context.Handler {
	if c.Options.Disable {
		// Return a nil handler if caching is disabled. The framework will skip this middleware.
		return nil
	}

	return func(ctx *context.Context) {
		invalidateContext(ctx, c, prefixes)
	}
}

// InvalidateRoutes returns an Iris middleware handler that invalidates the cache
// for all routes of the provided router after a resource mutation operation (DELETE, POST, PUT, PATCH).
// It collects all route prefixes that have the cache handler registered and invalidates them.
func (c *Cache) InvalidateRoutes(r router.Router) context.Handler {
	if c.Options.Disable {
		// Return a nil handler if caching is disabled. The framework will skip this middleware.
		return nil
	}

	var prefixes []string

	r.Application().RegisterOnBuild(nil, func() error {
		routes := r.Application().ListRoutesReadOnly()
		for _, route := range routes {
			if r != route.Router() {
				// fmt.Printf("skipping route %s, it does not belong to the provided router\n", route.StaticPath())
				continue
			}

			// Check if the route has the cache handler registered.
			// So we skip routes that do not have the cache handler registered.
			// This is not found when a cache middleware is not registered or the cache instance is disabled,
			// so the cache handler call would return nil.
			if !route.GetHandlerList().Exists(handlerName) {
				// fmt.Printf("skipping route %s, it does not have the cache %q handler registered\n", route.StaticPath(), handlerName)
				continue
			}

			prefix := fmt.Sprintf("httpcache:%s:%s:", getRouteIdentifier(route), route.GetMethod())
			// fmt.Printf("Found route %s with cache handler, prefix: %s\n", route.StaticPath(), prefix)
			prefixes = append(prefixes, prefix)
		}

		prefixes = router.RemoveDuplicates(prefixes)
		return nil
	})

	return func(ctx *context.Context) {
		invalidateContext(ctx, c, prefixes)
	}
}

func getRouteIdentifier(route context.RouteReadOnly) string {
	// Contains any parameter e.g. /users/{id:identifier} and parameter functions(!) e.g. /users/{id:identiifer min(4) max(255)}, so
	// we should trim it. But only the parameter, dynamic sections.
	// Because an end-developer may use _ for route paths as well.

	// Subdomain + method?
	return route.Tmpl().SanitizedText
}

func getRouteContextKey(ctx *context.Context) string { // NOTE: this doesn't support subdomains, if we want to we could.
	currentRoute := ctx.CurrentRoute()
	routeIdentifier := getRouteIdentifier(currentRoute)
	requestPath := ctx.Path()
	// Trim so static paths are not duplicated in the key.
	// httpcache:/u/groups/count:GET:...
	key := fmt.Sprintf("httpcache:%s:%s", routeIdentifier, ctx.Method())
	pathPart := strings.TrimPrefix(requestPath, currentRoute.StaticPath())
	pathPart = strings.TrimPrefix(pathPart, "/") // Remove leading slash if exists.
	if pathPart != "" {
		key += ":" + pathPart
	}
	return key
}

func invalidateContext(ctx *context.Context, cache *Cache, prefixes []string) {
	// Call the next handler in the chain.
	// This allows the handler to process the request and set the response and
	// the status code.
	ctx.Next()

	if !slices.Contains(resourceMutationStatusCodes, ctx.GetStatusCode()) {
		// If the status code is not 200, 201 or 204, skip invalidation.
		// This is to avoid invalidating the cache for unsuccessful operations.
		return
	}

	if !slices.Contains(resourceMutationMethods, ctx.Method()) {
		// If the method is not DELETE, POST, PUT or PATCH, skip invalidation.
		return
	}

	if ctx.GetErr() != nil {
		// If there was an error in the context, skip invalidation.
		return
	}

	if len(prefixes) == 0 {
		// If no prefixes are provided, invalidate all cache entries.
		if _, err := cache.InvalidateAll(ctx); err != nil {
			ctx.SetErrPrivate(err) // a store error is never a message for the client.
			return
		}
	} else {
		// Invalidate cache entries for the provided prefixes.
		for _, prefix := range prefixes {
			if _, err := cache.InvalidatePrefix(ctx, prefix); err != nil {
				ctx.SetErrPrivate(err) // a store error is never a message for the client.
				return
			}
		}
	}
}

// ContextKey generates a unique cache key for the current context.
// The key format is:
// httpcache:<route_tmpl>:<method>:<url_path>[:<query_hash>][:<body_hash>][:<headers_hash>][:<cookies_hash>]
// If the route is static, it uses the path and method.
// If the route is dynamic, it uses the route's template source, path, and method.
// The key is then appended with a suffix that includes the request's query, body, headers, and cookies hashes.
// This ensures that the cache key is unique for each request, even if the path and method are the same.
// This key is used to store and retrieve cached responses for the current request.
// If the cache is disabled, it returns an empty string.
// Examples:
// - httpcache:/users/list:POST
// - httpcache:/u/{identifier:string}:GET
func (c *Cache) ContextKey(ctx *context.Context) string {
	if c.db == nil {
		return "" // No-op if cache is disabled.
	}

	currentRoute := ctx.CurrentRoute()
	if currentRoute == nil {
		return c.RequestKey(ctx.Request()) // No route found, use request key.
	}

	key := getRouteContextKey(ctx)
	// Use the route's template source and method.
	// So on routes the prefix is always httpcache:<route_src>:<method>.

	key = c.appendKeySuffix(key, ctx.Request()) // Append query, body, headers, and cookies hashes.
	return key
}

/*
// Pair returns two Iris middleware handlers:
// 1. A cache handler that caches HTTP responses for GET and POST requests.
// 2. An invalidate handler that invalidates the cache for the specific routes
// that the cache handler was registered on.
//
// Read more about the cache handler in the `Handler` method.
// Read more about the invalidate handler in the `InvalidateHandler` method.
func (c *Cache) Pair(r router.Router) (context.Handler, context.Handler) {
	// if c.options.Disable {
	// 	return nil, nil
	// }

	cacheHandler := func(ctx *context.Context) {
		cacheContext(ctx, c, true)
	}
// generate unique handler name for this instance of the cache handler...
// set it to the routes' handler name by replacing the iris.httpcache with that
// or use the route's properties...
// No, that way will not keep the cache handler name on server restarts.
// Forget that idea.
	thisCacheHandlerName := ...
	invalidateHandler := invalidateRoutesHandler(r, c, thisCacheHandlerName)

	return cacheHandler, invalidateHandler
}
*/
