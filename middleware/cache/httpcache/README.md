# HTTP Cache Middleware for Iris

A high-performance HTTP response caching middleware for [Iris](https://www.iris-go.com) using DragonflyDB/Redis as a backend. It supports flexible cache key generation based on request path, method, query parameters, body, headers, and cookies, and provides efficient cache invalidation for dynamic APIs.

## Features

- **Plug-and-play Iris middleware** for GET and POST endpoints.
- **DragonflyDB/Redis backend** for fast, scalable caching.
- **Customizable cache key generation**: include/exclude query, body, headers, cookies.
- **Efficient cache invalidation**: by request, by prefix, or by path.
- **Batched SCAN+DEL** for safe production cache cleanup.
- **Only caches successful (HTTP 200) responses** by default.

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

## Best Practices

- **Invalidate cache** after any data-changing operation (create, update, delete).
- **Use prefix invalidation** for endpoints with many possible query/body variations.
- **Include headers/cookies** in the cache key if responses are user/session-specific.
- **Monitor cache hit/miss rates** for tuning.
