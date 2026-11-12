# Request Rate Limiting Middleware for Iris

This middleware caps how fast a client may send requests. It uses a token bucket per client: requests spend tokens, the bucket refills at the rate you set, and a request that finds it empty is rejected. The bucket's size is what lets a client burst above the steady rate for a short while.

## Installation

To use the `rate` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/rate"
```

## Usage

### Basic Setup

To use the rate limiter, you need to create an Iris application and register the middleware. Below is an example of how to set up the rate limiter:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rate"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	limit := rate.Limit(1, 5, rate.Options{
		PurgeEvery:       time.Minute,
		PurgeMaxLifetime: 5 * time.Minute,
	})
	app.Use(limit)

	app.Get("/", index)
	app.Get("/other", other)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}
```

This example allows 1 request per second with a maximum burst size of 5. It also purges old entries every minute if they haven't been seen for 5 minutes.

### Using `rate.Every` Helper

This example demonstrates how to use the `rate.Every` helper to set up a rate limiter:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rate"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Use rate.Every helper to set up the rate limiter.
	limit := rate.Limit(rate.Every(time.Minute), 5, rate.Options{})
	app.Use(limit)

	app.Get("/", index)
	app.Get("/other", other)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}
```

### Using API Key for Rate Limiting

This example demonstrates how to set up a rate limiter that uses an API key instead of the client's remote IP address:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Use API key for rate limiting.
	app.Use(useAPIKey)

	limit := rate.Limit(rate.Every(time.Minute), 300, rate.Options{
		PurgeEvery:       5 * time.Minute,
		PurgeMaxLifetime: 15 * time.Minute,
	})
	app.Use(limit)

	app.Get("/list", list)

	app.Listen(":8080")
}

func useAPIKey(ctx iris.Context) {
	apiKey := ctx.Header("X-API-Key")
	if apiKey == "" {
		// The error code carries the 403 and stops the chain.
		errors.PermissionDenied.Message(ctx, "the X-API-Key header is required")
		return
	}

	rate.SetIdentifier(ctx, apiKey)
	ctx.Next()
}

func list(ctx iris.Context) {
	ctx.JSON(iris.Map{"key": "value"})
}
```

### Custom Exceed Handler

This example demonstrates how to set a custom handler to be executed when the rate limit is exceeded:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Set a custom exceed handler. errors.DefaultRateLimitExceededHandler is
	// the canonical one: 429 RESOURCE_EXHAUSTED with the wait in the detail.
	limit := rate.Limit(1, 5, rate.Options{
		ExceedHandler: func(ctx iris.Context) {
			errors.ResourceExhausted.Messagef(ctx, "slow down, %s", rate.Get(ctx).Delay)
		},
	})
	app.Use(limit)

	app.Get("/", index)
	app.Get("/other", other)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}
```

### Custom Client Data

This example demonstrates how to store custom data for each client:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rate"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Store custom data for each client.
	limit := rate.Limit(1, 5, rate.Options{
		ClientData: func(ctx iris.Context) any {
			return ctx.RemoteAddr()
		},
	})
	app.Use(limit)

	app.Get("/", index)
	app.Get("/other", other)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}
```

### Explanation

- **Rate Limiting**: The `rate.Limit` function is used to create a new rate limiter. It takes three parameters:
  - `limit`: The maximum number of requests allowed per second.
  - `burst`: The maximum burst size.
  - `opts`: a `rate.Options` value with the optional `ExceedHandler`, `ClientData`, `PurgeEvery` and `PurgeMaxLifetime` fields. The zero value is valid.

  There is always a purger, because a limiter without one grows forever: every new client identifier allocates an entry that is never reclaimed. `PurgeEvery` (how often to sweep, one minute by default) and `PurgeMaxLifetime` (how long an idle client is kept, derived from the rate by default) are independent, so setting one alone leaves the other at its default.

  `rate.New(limit, burst, opts)` returns the `*rate.Limiter` object instead of a handler, so that you can `Stop` its background purge goroutine; register `limiter.Handler()`. `rate.Limit` is the one-liner for a limiter that lives as long as the process.

- **Token Bucket Algorithm**: This algorithm controls the rate of requests by maintaining a bucket of tokens. Each request consumes a token, and tokens are added to the bucket at a fixed rate. If the bucket is empty, the request is denied.

## Token Bucket Algorithm

The token bucket algorithm is a simple and efficient way to control the rate of requests. It works as follows:

1. **Initialization**: A bucket is initialized with a certain number of tokens.
2. **Token Addition**: Tokens are added to the bucket at a fixed rate.
3. **Request Handling**: Each request consumes a token. If the bucket is empty, the request is denied.
4. **Burst Handling**: The bucket can hold a maximum number of tokens, allowing for bursts of traffic.

For more details, refer to the [Wikipedia article on Token Bucket](https://en.wikipedia.org/wiki/Token_bucket).

For more examples and detailed usage, refer to the [official Iris documentation](https://github.com/kataras/iris/tree/main/_examples/06-middleware/rate-limit).
