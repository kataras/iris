# Caching

Three distinct layers, three different packages. Pick by where the copy lives:

| Layer | Copy lives | Package |
|-------|-----------|---------|
| Client / proxy cache headers | Browser, CDN, reverse proxy | `middleware/cache` |
| Server-side response cache | Redis / DragonflyDB, shared across instances | `middleware/cache/httpcache` |
| Application data cache | Process memory | `core/pkg/cache`, or your own |

## The Cache Middleware

`middleware/cache` writes cache headers and answers conditional `GET`, `HEAD` and `QUERY` requests with 304. It does not store responses.

```go
import "github.com/kataras/iris/v14/middleware/cache"

app.Use(cache.New(cache.Options{
    ETag:              true,
    CacheControl:      cache.NewCacheControlBuilder().Public().MaxAge(time.Hour).MustRevalidate().Build(),
    LastModified:      true,
    Expires:           24 * time.Hour,
    Pragma:            cache.NoPragmaCache, // "no-cache"
    Vary:              cache.NewVaryBuilder().AcceptEncoding().Build(),
    ProxyAuthenticate: "",
}))
```

| Field | Effect |
|-------|--------|
| `ETag bool` | Send an `ETag` and answer `If-None-Match` |
| `CacheControl string` | Verbatim `Cache-Control` value |
| `LastModified bool` | Send `Last-Modified` and answer `If-Modified-Since` |
| `Expires time.Duration` | `Expires`, computed from now |
| `Pragma string` | Legacy HTTP/1.0 header |
| `Vary string` | Which request headers vary the response |
| `ProxyAuthenticate string` | `Proxy-Authenticate` challenge |

`cache.DefaultOptions` is a reasonable public-content preset: ETag on, `public, max-age=3600, s-maxage=7200, must-revalidate, proxy-revalidate`, `LastModified` on, `Expires` 24h, `Pragma: no-cache`, `Vary: Accept-Encoding`.

**Important caveat.** This middleware derives its validators from the clock, not from your payload: the `ETag` is the current Unix second and `Last-Modified` is hardcoded to 24 hours ago. It is a header-policy and revalidation helper, **not** content validation. When a 304 must depend on whether the data actually changed, write the conditional logic yourself (below).

### The builders

All three return plain strings, so hoist them to startup rather than rebuilding per request.

```go
cache.NewCacheControlBuilder(). // Public, Private, NoCache, NoStore, MaxAge,
    Public().MaxAge(time.Hour). // SMaxAge, MustRevalidate, ProxyRevalidate
    MustRevalidate().Build()

cache.NewVaryBuilder().                        // Add, Accept, AcceptEncoding,
    AcceptEncoding().AcceptLanguage().Build()  // AcceptLanguage, UserAgent

cache.NewProxyAuthenticateBuilder().Basic("restricted").Build() // also Digest
```

## Caching a Group of Endpoints

Declare the policy once for a whole block. Passing `"/"` to `Router` returns the group itself rather than creating a child, and `context.Handler` arguments become that group's `Use`:

```go
func (api *CatalogAPI) Routing(r iris.Router) {
    cached := cache.New(cache.Options{
        ETag:         true,
        CacheControl: cache.NewCacheControlBuilder().Public().MaxAge(5 * time.Minute).Build(),
    })

    r.Router("/", cached, func(r iris.Router) {
        r.Get("/categories", api.categories) // both inherit the policy
        r.Get("/currencies", api.currencies)
    })

    r.Get("/orders", api.orders) // outside the block: uncached
}
```

## Conditional Requests by Hand

When the 304 must reflect real data changes, derive the validator from the data.

```go
func (api *ArticlesAPI) get(ctx iris.Context) {
    article, err := api.articles.ByID(ctx, ctx.Params().GetString("id"))
    if err != nil {
        errors.HandleError(ctx, err)
        return
    }

    modified, err := ctx.CheckIfModifiedSince(article.UpdatedAt)
    if !modified && err == nil {
        ctx.WriteNotModified() // 304, no body
        return
    }

    ctx.SetLastModified(article.UpdatedAt)
    ctx.OK(article, nil)
}
```

The error semantics are the subtle part: `CheckIfModifiedSince` returns an error wrapping `context.ErrPreconditionFailed` when the method is not `GET`, `HEAD` or `QUERY` or the request carried no usable `If-Modified-Since`, which means *send the full response*. That is why the guard is `!modified && err == nil` and not just `!modified`.

Related helpers: `ctx.WriteWithExpiration(body, modtime)` writes a body with validators in one call, and `ctx.MaxAge()` reads the request's `Cache-Control: max-age`.

## Server-Side Response Caching

`middleware/cache/httpcache` stores rendered responses in Redis or DragonflyDB, so the cache is shared across every instance and survives restarts. The Redis client lives inside the middleware; you operate the server but import nothing.

```go
import "github.com/kataras/iris/v14/middleware/cache/httpcache"

store, err := httpcache.New(httpcache.Options{
    Addr:           "127.0.0.1:6379",
    Password:       os.Getenv("CACHE_PASSWORD"),
    TTL:            10 * time.Minute,
    CommandTimeout: 200 * time.Millisecond,
    StartupCleanup: false,
    Headers:        []string{"Authorization"}, // part of the cache key
    Cookies:        []string{"session_id"},    // part of the cache key: name every cookie a request sends
    NoQuery:        false,
    NoBody:         false,
    Disable:        os.Getenv("ENV") == "development",
})
```

`store.Handler()` (and the caching half of `Pair`) caches `GET`, `QUERY` and `POST` requests and stores only 200 responses, and of those not one that sets a cookie or whose `Cache-Control` carries `private`, `no-store` or `no-cache`. QUERY and POST are there for list endpoints that carry their filter in a JSON body, which is why the body hash is part of the key unless `NoBody` is set. A hit is sent with `Cache-Control: public, max-age=<seconds left>`.

`httpcache.Must` takes the same `Options` and panics instead of reporting the error, for the common case of wiring the cache at start-up.

**Cache-key safety is the one thing you cannot get wrong.** If a response varies by user, the identity must be in the key via `Headers` or `Cookies`. Otherwise the first user's response is served to everyone. When you cannot enumerate what makes a response unique, do not response-cache that route.

Three guards back that rule up, and none is a substitute for configuring the key:

- A request carrying `Authorization`, `Proxy-Authorization` or cookies is looked up and stored only when the key includes every one of them: the header is named in `Headers` (in any letter case) and each cookie the request sends is named in `Cookies` (in the same letter case), or `Headers` names `Cookie` and the key holds the whole header. Any other request with credentials is **not cached at all**: its handler runs every time, nothing is stored, and the middleware logs once to say so. Under the default options that is every request with credentials. Naming something is not enough: with `Headers: []string{"Accept-Language"}` or `Cookies: []string{"theme"}` a request that carries a session cookie is not cached. That has a cost. Under `Cookies: []string{"sid"}` a request that sends `sid` and one more cookie (analytics, a theme) is not cached, and under `Headers: []string{"Authorization"}` neither is one that sends any cookie, until those cookies are named in `Cookies` too. Under `Cookies` a request is not cached either when its `Cookie` line is not a list of `name=value` pairs that `net/http` reads in full: a trailing semicolon, a bare name, a double quote or a non-ASCII byte inside a value. The check sees those three credentials only. One in a custom header (`X-Token`) is invisible to it unless `Headers` names it.
- A response that sets a cookie is never stored, and `Set-Cookie`, `Authorization` and the hop-by-hop headers are stripped from what is stored. Replaying a `Set-Cookie` to a later cache hit would hand that session to another client. The cookie counts as well when a handler registered before the cache handler has already set it, so a middleware that sets a cookie on every response switches the store off for the routes behind it. A cookie written after the response was stored does not count: that is the `sessions.Handler` order below.
- A response whose `Cache-Control` carries `private`, `no-store` or `no-cache` is never stored. Its handler runs on every request and the response keeps the header it was given. The directive counts whether the route handler set it or a handler registered before the cache handler did (security headers ahead of the access log and compression included), and `Response.WriteTo` leaves such a directive on a stored response in place where it would write `public, max-age=N`.

The third guard has costs. A middleware ahead of the cache handler that sends one of the three directives on every response switches the store off for every route under it, with no log line; a route opts back in by setting its own `Cache-Control` (`public, max-age=60`), which replaces the inherited one. A response cannot be both stored and `private`, not even with `Headers: []string{"Authorization"}` in the key. A route cached per credential has to leave the directive off, and its hits then go out as `public, max-age=N`: keep shared caches away from such a route, or do not response-cache it. A stream opened with `sse.Open` sends `no-cache`, so it is not stored and streams on every request. Other directives do not keep a response out: `max-age=0`, `s-maxage=0` and `must-revalidate` are still stored and replayed as `public, max-age=N`.

With `sessions.Handler`, register the cache handler first. `sessions.Handler` commits only a session the request changed, so a response to a request that only reads the session carries no `Set-Cookie` and the second guard does not see it. The first guard covers that request: it carries the session cookie, and it is cached only under a key that names every cookie it sends. What is left is a new session, whose cookie is written when `sessions.Handler` commits, after the handlers behind it have returned. With the cache handler registered before `sessions.Handler` the response reaches it with its `Set-Cookie` and is not stored. Registered after it, the response is stored before the cookie is written, so the page made for one new session is served to the next anonymous visitor. When the order cannot change, send `Cache-Control: private` on such a page.

A key that names the session cookie stores a page per session. After a sign-out, a request that still sends the old cookie gets the stored page until the entry expires, because a hit runs no handler behind the cache handler: invalidate on sign-out, or send `Cache-Control: private` on the page. A reverse proxy that caches is a different case. It skips responses with `Set-Cookie` at most, so behind one every page that depends on the session sends `Cache-Control: private`.

A store error fails open: the request proceeds to the handler rather than answering an empty body.

The header-only `middleware/cache` is a different thing and has no such guard, because it stores nothing: it only sets `Cache-Control` and friends. `cache.New(opts)` applies your policy to every request, which is right for a route that is the same for everybody. `cache.Strict(opts)` keeps that policy for anonymous traffic and substitutes a private one for any request carrying credentials, which is right for a route that serves both.

`Disable: true` is a real no-op, not a flag checked per request: `Pair` returns `nil, nil` and `Handler` returns `nil`, so development builds carry no overhead.

### Registering and invalidating

`Pair` returns two handlers: one serves from the cache, the other flushes affected prefixes after a mutation completes.

```go
func (api *CatalogAPI) Routing(r iris.Router) {
    caching, invalidating := api.store.Pair(r)

    r.Router("/", caching, invalidating, func(r iris.Router) {
        r.Get("/products", api.list)
        r.Get("/products/{id:uuid}", api.get)
        r.Post("/products", api.create)   // flushes the prefix on success
        r.Put("/products/{id:uuid}", api.update)
    })
}
```

The invalidating handler runs after your handler and flushes when a `POST`, `PUT`, `PATCH` or `DELETE` finishes with 200, 201 or 204. For explicit control: `store.InvalidateHandler(prefixes...)` as a route handler, or `store.InvalidateRoutes(r)`.

### Store methods

For operating and debugging the cache directly:

| Method | Purpose |
|--------|---------|
| `Get`/`Set`, `GetResponse`/`SetResponse` | Read and write entries |
| `Invalidate(ctx, key)`, `InvalidateRequest(ctx, r)` | Drop one entry |
| `InvalidatePrefix(ctx, prefix) (int, error)` | Drop a subtree, returns the count |
| `InvalidateAll(ctx) (int, error)` | Flush everything |
| `CountAll(ctx)`, `CountPrefix(ctx, prefix)` | Size checks for a stats endpoint |
| `RequestKey(r)`, `ContextKey(ctx)` | The key derivation, for debugging |
| `Close() error` | Shutdown |

The `Options` value stays reachable as the exported `store.Options` field for runtime inspection.

Register the store through `Services(store)` and its `Close` becomes a shutdown closer automatically:

```go
builder.Services(store, catalog.NewService)
```

## Application-Level Caching

For computed values rather than HTTP responses. The framework's contribution is lifecycle, not the cache itself: register it as a service, warm it in `Init`, close it on shutdown.

```go
type Cache struct {
    repo  ProductRepo
    mu    sync.RWMutex
    items []Product
    at    time.Time
}

func NewCache(repo ProductRepo) *Cache { return &Cache{repo: repo} }

// Init runs inside Build(), after DI and before serving.
func (c *Cache) Init(ctx context.Context) error { return c.refresh(ctx) }

func (c *Cache) Products(ctx context.Context) ([]Product, error) {
    c.mu.RLock()
    fresh := time.Since(c.at) < 5*time.Minute
    items := c.items
    c.mu.RUnlock()
    if fresh {
        return items, nil
    }
    if err := c.refresh(ctx); err != nil {
        if items != nil {
            return items, nil // stale beats broken
        }
        return nil, err
    }
    // ...
}
```

`Services(repo, NewCache)` is the whole wiring: the container builds it, `Init` warms it before the first request, and a `Close` method (if present) drains it at shutdown. See [architecture-and-patterns.md](./architecture-and-patterns.md).

For a bounded in-memory map with iteration, the framework ships one:

```go
c := cache.NewMemoryCache[Session](1000) // core/pkg/cache

c.Set(ctx, key, &session)                 // note: takes a pointer
v, found, err := c.Get(ctx, key)
c.Delete(ctx, key)
c.Walk(ctx, func(key string, v *Session) error { return nil })
key, v, found, err := c.First(ctx, func(v *Session) bool { return v.UserID == id })
```

## Invalidation

Prefer short TTLs with explicit invalidation on the write path over long TTLs and hope. `Pair` already covers the common case (a mutation under a prefix flushes that prefix). What it cannot know is a write in one resource that invalidates another; do those explicitly with `InvalidatePrefix`.

`httpcache` is already distributed, so a multi-instance deployment needs no extra coordination. Only process-local caches (the section above) go stale per instance.

## Source Files

- `middleware/cache/cache.go` - `New`, `Options`, `DefaultOptions`, `NoPragmaCache`
- `middleware/cache/cache_control.go`, `vary.go`, `proxy_authenticate.go` - The builders
- `middleware/cache/httpcache/cache.go`, `handler.go` - Server-side store, `Pair`, invalidation
- `context/context_response.go` - `SetLastModified`, `CheckIfModifiedSince`, `WriteNotModified`, `WriteWithExpiration`
- `core/pkg/cache/memcache.go` - `NewMemoryCache`, `Cache[T]`, `CacheIterator[T]`
- [book/14-caching-strategies.md](https://github.com/kataras/iris-book/blob/main/book/14-caching-strategies.md) - The long-form chapter this condenses
