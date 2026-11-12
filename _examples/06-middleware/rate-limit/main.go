// Rate limiting: token buckets per authenticated principal, with Retry-After on
// rejection.
//
// rate.Limit(limit, burst, rate.Options{}) is a token-bucket limiter with one bucket per
// client: every request spends a token and the bucket refills at "limit"
// tokens per second, up to "burst". A client staying under the refill rate
// never notices; one that surges drains its bucket and is rejected until
// tokens accumulate again. rate.Every converts an interval into that
// per-second float, so "two per minute" is rate.Every(time.Minute) with a
// burst of 2.
//
// A request that finds an empty bucket does NOT reach the handler: the exceed
// handler runs instead of the rest of the chain, and the default one replies
// 429 Too Many Requests with an empty body, while
// errors.DefaultRateLimitExceededHandler renders the rest/errors envelope as
// 429 RESOURCE_EXHAUSTED. rate.Get(ctx) returns the
// *rate.Client of the current request; its Delay field holds how long until
// the next token - the value to put in Retry-After. In a normal handler the
// same Client exposes ID, the optional Data filled by rate.Options.ClientData,
// and the underlying limiter.
//
// Clients are keyed by ctx.RemoteAddr() unless a middleware registered BEFORE
// the limiter calls rate.SetIdentifier(ctx, key). The identifier below is the
// principal middleware/apikey authenticated, never the raw header value: a
// bucket keyed by whatever the client typed is a bucket the client can reset
// by typing something else. Each rate.Limit call owns its own map of clients,
// so the two limiters below count separately.
// rate.RegisterExceedHandler sets the exceed handler for a whole router;
// applications built with iris.NewBuilder already bind it to the JSON error
// renderer of rest/errors.
//
// The limiter state is in-process memory - limits shared across replicas need
// a proxy or a store-backed limiter in front.
//
// Learn: limit requests per client with rate.Limit, key the buckets by an
// authenticated identity, and render the 429 with Retry-After.
// Before: ../cors
//
// Run:
//
//	go run main.go
//
// Try:
//
//	for i in 1 2 3 4 5 6; do curl -s -o /dev/null -w "%{http_code}\n" \
//	    -H "X-API-Key: alice-key" http://localhost:8080/api/items; done   # 200 five times, then 429
//	curl -s -H "X-API-Key: bob-key" http://localhost:8080/api/items       # 200, a different principal is a separate full bucket
//	curl -i http://localhost:8080/api/items                               # 401, before the limiter is reached
//	curl -i -X POST http://localhost:8080/auth/login                      # 200 twice per minute per IP, then 429 with Retry-After
//
// Next: ../request-id
package main

import (
	"math"
	"strconv"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/apikey"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// General traffic: one request per second, bursts of up to five, counted
	// per authenticated principal. The three handlers run in this order, so
	// the identifier is in place by the time the limiter looks it up.
	api := app.Router("/api",
		apikey.New(apikey.Options{
			Header: "X-API-Key",
			Scheme: apikey.NoScheme,
			Keys: map[string]string{
				"alice": "alice-key",
				"bob":   "bob-key",
			},
		}),
		identifyByPrincipal,
		rate.Limit(1, 5, rate.Options{ExceedHandler: tooManyRequests}))

	api.Get("/items", func(ctx iris.Context) {
		client := rate.Get(ctx) // never nil below the limiter.

		ctx.JSON(iris.Map{
			"client":           client.ID,
			"tokens_remaining": math.Floor(client.Limiter.Tokens()),
		})
	})

	// Logins have nobody to authenticate yet, so they keep the default
	// identity, ctx.RemoteAddr(), and get a much stricter bucket of their
	// own: two per minute, plus a sweep that drops clients idle for more than
	// five minutes so the map cannot grow forever.
	auth := app.Router("/auth", rate.Limit(rate.Every(time.Minute), 2, rate.Options{
		PurgeEvery:       time.Minute,
		PurgeMaxLifetime: 5 * time.Minute,
		ExceedHandler:    tooManyRequests,
	}))

	auth.Post("/login", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"token": "..."})
	})

	app.Listen(":8080")
}

// identifyByPrincipal keys the bucket by the caller apikey authenticated.
// It runs below the guard, so a request without a principal never gets here.
func identifyByPrincipal(ctx iris.Context) {
	if principal, ok := apikey.Get(ctx); ok {
		rate.SetIdentifier(ctx, principal)
	}

	ctx.Next()
}

// tooManyRequests replaces the rest of the chain when the client's bucket is
// empty. Delay is the wait until a token is available for it again.
//
// It is errors.DefaultRateLimitExceededHandler plus the Retry-After header:
// the error code carries the 429 and stops the chain, so all this handler
// adds is the header and the wait in the detail.
func tooManyRequests(ctx iris.Context) {
	delay := rate.Get(ctx).Delay

	ctx.SetHeader("Retry-After",
		strconv.Itoa(int(math.Ceil(delay.Seconds()))))
	errors.ResourceExhausted.Detailf(ctx, "rate limit exceeded",
		"you may try again in %s", delay)
}
