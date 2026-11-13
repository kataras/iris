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
// A response that sets a cookie, or whose Cache-Control carries private, no-store
// or no-cache, is not stored: its handler runs for every request.
// The cache key is generated based on the request's URL path, method, query parameters,
// headers, cookies and body.
//
// A request that carries Authorization, Proxy-Authorization or cookies is looked
// up and stored only when the cache key includes every one of them: the header
// is named in Options.Headers and each cookie it sends is named in
// Options.Cookies. Otherwise its handler runs for every request and nothing is
// stored, which is what happens to such a request under the default Options.
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
	//
	// Naming something is not enough: with Cookies: ["theme"] a session cookie
	// is still outside the key. Every credential of the request has to be in.
	if !c.keyIncludesCredentials(ctx.Request()) {
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

		// A handler registered before this one set its headers on a writer
		// under the recorder. The recorded ones replace those by name when the
		// recorder is released, so the ones below count when none was recorded:
		// those of the writer the recorder wraps, then those of the writer at
		// the bottom. The two are not the same when a second recorder (the
		// access log's) lies under a compression writer.
		sent := func(key string) []string {
			values, recorded := headers[key]
			if !recorded {
				values, recorded = rr.ResponseWriter.Header()[key]
			}
			if !recorded {
				if bottom := rr.ResponseWriter.Unwrap(); bottom != nil {
					values = bottom.Header()[key]
				}
			}

			return values
		}

		// A response that issues a cookie is per-client by definition: caching it
		// would hand that session to every later cache hit.
		if len(sent(setCookieHeaderKey)) > 0 || len(sent(setCookie2HeaderKey)) > 0 {
			return
		}

		// Nor is a response stored that says private, no-store or no-cache. The
		// handler runs for every request then, so each response goes out with
		// the Cache-Control it was given and not with the "public" of a replay.
		if forbidsSharedStorage(sent(context.CacheControlHeaderKey)) {
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

// forbidsSharedStorage reports whether one of the Cache-Control field lines
// carries private, no-store or no-cache (RFC 9111, section 5.2.2).
//
// The first marks a response for one client and the second a response for no
// cache at all. The third allows storing but no reuse before the origin has
// validated the response, and a replay does not run the handler. The forms
// with field names, private="Set-Cookie" and no-cache="Set-Cookie", count as
// the bare directive, which is how the RFC says caches commonly treat them.
//
// A directive is a name with an optional token or quoted argument. Names are
// compared without case and a comma inside a quoted argument does not end the
// directive. A line with a quote that is never closed cannot be read, so it
// counts as well.
func forbidsSharedStorage(cacheControl []string) bool {
	for _, line := range cacheControl {
		start, quoted := 0, false

		for i := 0; i <= len(line); i++ {
			if i < len(line) && (quoted || line[i] != ',') {
				switch {
				case line[i] == '"':
					quoted = !quoted
				case line[i] == '\\' && quoted:
					i++ // a quoted pair: the next byte does not close the argument.
				}

				continue
			}

			// line[start:i] is one directive.
			name, _, _ := strings.Cut(line[start:i], "=")
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "private", "no-store", "no-cache":
				return true
			}

			start = i + 1
		}

		if quoted {
			return true
		}
	}

	return false
}

// keyIncludesCredentials reports whether the cache key holds every credential
// the request carries: its Authorization header, its Proxy-Authorization
// header and each of its cookies. It is true for a request that carries none.
//
// A header is in the key when Options.Headers names it. Header names compare
// without regard to case, as http.Header.Get reads them for the key.
//
// A cookie counts by its name, compared as it is written: it is in the key
// when Options.Cookies names it. A name that is sent twice counts once, and
// the key holds the first value, the one http.Request.Cookie returns. A Cookie
// line that http.ParseCookie rejects, one that is not a list of name=value
// pairs, is in no key: http.Request.Cookie skips some of the parts of such a
// line. Nor are the cookies of a request that sends more of them than
// net/http reads: http.Request.Cookie returns none of them. The exception is
// the first Cookie line when Options.Headers names Cookie: the key holds that
// line as it was sent.
func (c *Cache) keyIncludesCredentials(r *http.Request) bool {
	inKey := func(header string) bool {
		return slices.ContainsFunc(c.Options.Headers, func(name string) bool {
			return http.CanonicalHeaderKey(name) == header
		})
	}

	for _, header := range []string{authorizationHeaderKey, proxyAuthorizationHeaderKey} {
		if r.Header.Get(header) != "" && !inKey(header) {
			return false
		}
	}

	lines := r.Header[cookieHeaderKey]
	if len(lines) > 0 && inKey(cookieHeaderKey) {
		lines = lines[1:]
	}

	named := false // the request sends a cookie that Options.Cookies names.

	for _, line := range lines {
		cookies, err := http.ParseCookie(line)
		if err != nil && strings.TrimSpace(line) != "" { // an empty line carries no cookie.
			return false
		}

		for _, cookie := range cookies {
			if !slices.Contains(c.Options.Cookies, cookie.Name) {
				return false
			}

			named = true
		}
	}

	// net/http reads no cookie at all from a request that sends more of them
	// than its limit (GODEBUG httpcookiemaxnum), counted over all the Cookie
	// lines: the key would hold none of the named ones.
	return !named || len(r.Cookies()) > 0
}

// warnCredentialedOnce logs, once, that credentialed requests are being skipped.
func (c *Cache) warnCredentialedOnce(ctx *context.Context) {
	c.warnOnce.Do(func() {
		ctx.Application().Logger().Warn(
			"httpcache: a request carrying Authorization, Proxy-Authorization or a cookie that the cache key " +
				"does not include is NOT cached, because the key would not distinguish clients. Name the header " +
				"in Options.Headers (e.g. []string{\"Authorization\"}) and every cookie the request sends in " +
				"Options.Cookies to cache it safely.")
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
