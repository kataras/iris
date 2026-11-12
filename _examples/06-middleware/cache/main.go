// Caching on two layers: client caching headers and server-side response
// caching.
//
// middleware/cache sets the HTTP caching header family (Cache-Control,
// Expires, ETag, Last-Modified, Vary, Pragma) and answers conditional GET/HEAD
// requests with 304 Not Modified. It stores nothing; it teaches browsers and
// proxies to. The clean way to apply it is one handler on one group, so the
// policy for a cluster of rarely-changing "constants" endpoints lives in a
// single place.
//
// middleware/cache/httpcache is the other layer: it stores whole rendered
// responses in DragonflyDB/Redis and replays them without running your
// handlers, and Pair(r) couples the caching handler with one that flushes the
// group's entries after a successful mutation.
//
// Learn: set client caching headers for a whole group with middleware/cache,
// and store rendered responses with httpcache, dropped after a mutation.
// Before: ../accesslog
//
// Without a store the response cache is a working no-op. To use one, set REDIS_ADDR to the
// address of a DragonflyDB or Redis server.
//
// Run:
//
//	go run main.go
//	REDIS_ADDR=localhost:6379 go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/meta/countries   # 200, Cache-Control, ETag, Expires, Vary
//	curl -i http://localhost:8080/reports/summary  # 200; with a store the next call is replayed
//	curl -i -X POST http://localhost:8080/reports  # 201, invalidates the entries above
//
// Next: ../i18n
package main

import (
	"log"
	"os"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cache"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

func main() {
	app := iris.New()

	registerMeta(app)

	store, err := registerReports(app)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	app.Listen(":8080")
}

// registerMeta is the client-caching layer: lookup data that changes only on
// deployment, cached by one middleware on one group.
func registerMeta(app *iris.Application) {
	constants := cache.New(cache.Options{
		// Adds an ETag and answers a request whose If-None-Match repeats it
		// with 304 Not Modified. The tag marks when the response was produced
		// (one-second resolution); it is not a digest of the payload.
		ETag: true,
		// The builders produce plain header strings, so they can also be built
		// once at startup and reused.
		CacheControl: cache.NewCacheControlBuilder().
			Public().                // any cache may store it.
			MaxAge(24 * time.Hour).  // browsers: refetch at most once a day.
			SMaxAge(48 * time.Hour). // shared caches: two days.
			MustRevalidate().
			Build(),
		LastModified: true,
		Expires:      24 * time.Hour,
		Pragma:       cache.NoPragmaCache, // "no-cache", for HTTP/1.0 clients.
		// Vary tells caches which request headers change the answer. Without
		// Accept-Encoding a gzipped body can be replayed to a client that
		// cannot decompress it.
		Vary: cache.NewVaryBuilder().AcceptEncoding().Build(),
	})

	// A handler argument is registered with Use on the group, so it applies to
	// every route the closure declares: "everything in this block is cached
	// with this policy".
	app.Router("/meta", constants, func(r iris.Router) {
		r.Get("/countries", func(ctx iris.Context) {
			ctx.JSON([]string{"DE", "GR", "JP", "US"})
		})

		r.Get("/currencies", func(ctx iris.Context) {
			ctx.JSON([]string{"EUR", "JPY", "USD"})
		})
	})

	// cache.New(cache.DefaultOptions) is a ready public preset with the same
	// shape. For per-user content use NewCacheControlBuilder().Private(), and
	// NoCache().NoStore() for responses that must never be reused.
}

// registerReports is the server-caching layer: expensive responses stored in
// DragonflyDB/Redis and dropped as soon as a mutation succeeds.
// The caller owns the returned store and closes it on shutdown. With the
// builder stack it would be registered through Services instead: the container
// injects it into controller constructors and, because *httpcache.Cache has a
// Close() error method, closes it at shutdown for you.
func registerReports(app *iris.Application) (*httpcache.Cache, error) {
	addr := os.Getenv("REDIS_ADDR")

	store, err := httpcache.New(httpcache.Options{
		// Disable returns a fully working no-op instance: Handler and Pair
		// return nil handlers (the framework skips nil middleware) and every
		// store method returns immediately. The wiring below stays identical
		// with or without a store.
		Disable: addr == "",
		Addr:    addr,
		TTL:     5 * time.Minute,
		// The identity of the caller must be part of the key whenever the
		// response is user-specific, otherwise the first user's response is
		// replayed to everyone.
		Headers: []string{"Authorization"},
	})
	if err != nil {
		return nil, err
	}

	reports := app.Router("/reports")

	// Pair binds the two halves to the same group: caching marks the routes
	// whose responses are stored, invalidating deletes every entry under those
	// routes' key prefixes after a POST/PUT/PATCH/DELETE finishes 200/201/204.
	caching, invalidating := store.Pair(reports)

	reports.Get("/summary", caching, func(ctx iris.Context) {
		// Stands in for the aggregation query, template render or downstream
		// call this layer exists to avoid: it runs at most once per TTL window.
		time.Sleep(300 * time.Millisecond)
		ctx.JSON(iris.Map{"total": 42, "generated": time.Now()})
	})

	reports.Post("/", invalidating, func(ctx iris.Context) {
		ctx.Created(iris.Map{"queued": true}, nil)
	})

	// InvalidateHandler(prefixes...), InvalidateRoutes(r) and the store methods
	// Invalidate/InvalidatePrefix/InvalidateAll/CountPrefix/CountAll cover the
	// cases Pair does not, such as an admin endpoint or a background job.

	return store, nil
}
