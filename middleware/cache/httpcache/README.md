# HTTP Cache Middleware for Iris

A high-performance HTTP response caching middleware for [Iris](https://www.iris-go.com) using DragonflyDB/Redis as a backend. It supports flexible cache key generation based on request path, method, query parameters, body, headers, and cookies, and provides efficient cache invalidation for dynamic APIs.

## Features

- **Plug-and-play Iris middleware** for GET and POST endpoints.
- **DragonflyDB/Redis backend** for fast, scalable caching.
- **Customizable cache key generation**: include/exclude query, body, headers, cookies.
- **Efficient cache invalidation**: by request, by prefix, or by path.
- **Batched SCAN+DEL** for safe production cache cleanup.
- **Only caches successful (HTTP 200) responses.** A response that sets a cookie, or whose `Cache-Control` carries `private`, `no-store` or `no-cache`, is not stored: its handler runs on every request and the response keeps the header it was given. One cookie is missed: a handler registered before the cache handler can write its cookie after the rest of the chain has returned, which is what `sessions.Handler` does (see [Best Practices](#best-practices)).
- **Caches a request with credentials only when the key tells its sender apart.** A request that carries `Authorization`, `Proxy-Authorization` or cookies is looked up and stored only when the cache key includes every one of them. Any other such request runs its handler every time.

## Installation

To use the `httpcache` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/cache/httpcache"
```

## Usage

### 1. Configure and Initialize the Cache

```go
import "github.com/kataras/iris/v14/middleware/cache/httpcache"

cache, err := httpcache.New(httpcache.Options{
    Addr:     "localhost:6379",
    Password: "",
    TTL:      10 * time.Minute,
    Headers:  []string{"Authorization"}, // Optional: include headers in cache key
    Cookies:  []string{"session_id"},    // Optional: include cookies in cache key
    StartupCleanup: true, // Optional: clean up cache on startup.
})
if err != nil {
    log.Fatalf("failed to initialize cache: %v", err)
}
// Or, to panic instead of reporting the error:
//	cache := httpcache.Must(httpcache.Options{...})
defer cache.Close()
```

`Headers` and `Cookies` also decide which requests with credentials are cached at all. A request that carries `Authorization`, `Proxy-Authorization` or cookies is looked up and stored only when the key includes every one of them: the header is named in `Headers`, in any letter case, and each cookie the request sends is named in `Cookies`, in the same letter case. Any other request with credentials runs its handler every time and nothing is stored, which is what happens to all of them under the default `Options`. The first request that is skipped this way logs one warning.

With the options above, a request that sends `Authorization` and `session_id` is cached, one entry per pair of values. A request that sends one more cookie next to `session_id` (analytics, a theme) is not, until that cookie is named too. To key on the whole `Cookie` header instead, whatever cookies it holds, name `Cookie` in `Headers`.

### 2. Register the Middleware in Iris

```go
app := iris.New()
app.Use(cache.Handler())
```

### 3. Invalidate Cache When Data Changes

#### Invalidate all cached user lists after a user is deleted:

```go
// In your controller after deleting a user:
prefix := "httpcache:/users/list:POST:"
deleted, err := cache.InvalidatePrefix(ctx, prefix)
```

## Cache Key Structure

The cache key is generated as:

```
httpcache:<route_tmpl>:<method>:<url_path>[:<query_hash>][:<body_hash>][:<headers_hash>][:<cookies_hash>]
```

- **route_tmpl**: The route template (e.g. `/users/{id:uuid}`)
- **method**: HTTP method (GET, POST, etc.)
- **url_path**: The request path (e.g. `/users/2707df68-5f7b-480a-a8b8-365830b39a20`)
- **query_hash**: SHA256 of the query string (optional)
- **body_hash**: SHA256 of the request body (optional)
- **headers_hash**: SHA256 of selected headers (optional)
- **cookies_hash**: SHA256 of selected cookies (optional)

You can control which parts are included via the `Options` struct.

## Example

```go
package main

import (
    "log"
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/cache/httpcache"
)

func main() {
    cache, err := httpcache.New(httpcache.Options{
        Addr:     "localhost:6379",
        TTL:      5 * time.Minute,
        Headers:  []string{"Authorization"},
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cache.Close()

    app := iris.New()
    app.Use(cache.Handler())

    app.Get("/hello", func(ctx iris.Context) {
        ctx.WriteString("Hello, world!")
    })

    app.Listen(":8080")
}
```

`Headers: []string{"Authorization"}` gives each `Authorization` value its own entry, and the requests without the header share one. A request that sends a cookie, with the header or without it, is not cached here, because no cookie is part of the key: name it in `Cookies`.

## Controller Example with Invalidation

```go
type UserController struct {
	cache *httpcache.Cache
}

func NewUserController(cache *httpcache.Cache) *UserController {
	return &UserController{
		cache:        cache,
	}
}

func (c *UserController) Routing(r router.Router) {
	// caching := c.cache.Handler()
	// invalidating := c.cache.InvalidateHandler()
    // OR:
    // invalidating := c.cache.InvalidateRoutes(r)
    // OR:
    caching, invalidating := c.cache.Pair(r)

	r.Post("/signup", invalidating, c.signup)
	r.Post("/signin", caching, c.signin)
	r.Delete("/", invalidating, c.deleteUser)
	r.Post("/restore", invalidating, c.restoreDeletedUser)
	r.Post("/request-password-reset", c.requestPasswordReset)
	r.Post("/reset-password", invalidating, c.resetPassword)
	r.Put("/", invalidating, c.updateUsers)
	r.Post("/list", caching, c.listUsers)
}

// [...implement the controller methods like signup, signin, deleteUser, etc.]
```

```go
app := iris.New()
// [...]

cache, err := httpcache.New(httpcache.Options{
        Addr:     "localhost:6379",
        Password: "kataras!123", // Optional: set password if your Redis requires it.
        TTL:      5 * time.Minute,
        Headers:  []string{"Authorization"},
})
if err != nil {
    panic(fmt.Errorf("failed to connect to the cache: %w", err))
}

app.Router("/u", cache, NewUserController)
```

As in the example above, the key holds `Authorization` and no cookie, so a client of `/u` that sends a cookie is not cached until `Cookies` names it.

## Best Practices

- **Invalidate cache** after any data-changing operation (create, update, delete).
- **Use prefix invalidation** for endpoints with many possible query/body variations.
- **Include headers/cookies** in the cache key if responses are user/session-specific. For `Authorization`, `Proxy-Authorization` and cookies the middleware insists on it: a request that carries one the key does not include is not cached. Under `Cookies: []string{"sid"}` that is a request that sends `sid` and one more cookie, and under `Headers: []string{"Authorization"}` one that sends any cookie. A header of your own that identifies the caller, an `X-Token` or an `X-Api-Key`, is not checked: leave it out of `Headers` and every caller shares one entry.
- **Send `Cache-Control: private`** from a handler whose response must not be shared: a response marked `private`, `no-store` or `no-cache` is never stored. That also holds when a middleware registered before the cache handler set the header, so one that sends `no-store` on every response turns the cache off for the routes under it, without a log line, until a route sets its own `Cache-Control` in its handler.
- **Register the cache handler before `sessions.Handler`.** Registered after it, the cache handler stores the response before the session cookie is written, so the page made for one new session is served to the next visitor who has no cookie. Send `Cache-Control: private` on such a page when the order cannot change. A cookie that a middleware sets before the cache handler runs does keep the response out of the store, so one that sets a cookie on every response turns the cache off for the routes behind it.
- **Monitor cache hit/miss rates** for tuning.
