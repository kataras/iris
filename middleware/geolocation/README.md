# Geolocation

This middleware resolves the client's IP address to a location. It calls [ip-api.com](http://ip-api.com) for the geolocation data and, when you ask for it, [restcountries.com](https://restcountries.com) for the country's primary language. Results are cached in memory, so a repeat visitor costs no external call.

## Features

- Fetches geolocation information based on the client's IP address.
- Optionally fetches the primary language of the country.
- Customizable error handling.
- In-memory caching for geolocation information.

### Installation

To use the geolocation middleware, you need to import it into your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/geolocation"
```

## Usage

### Middleware Usage

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/geolocation"
)

func main() {
	app := iris.New()
	app.Use(geolocation.New(geolocation.Options{}))

	app.Get("/", func(ctx iris.Context) {
		info := geolocation.Get(ctx)
		ctx.JSON(info)
	})

	app.Listen(":8080")
}
```

### Options

`geolocation.New` takes a `geolocation.Options` value. The zero value is valid; every field below is optional.

- `APIKey`: the API key for the ip-api.com API. When set (and `LocationClient` is not), the paid HTTPS endpoint is used. Defaults to an empty string.
- `LocationClient`: a custom HTTP client for the location provider (ip-api.com by default). Useful for a self-hosted provider or a mock server in tests.
- `LanguageClient`: a custom HTTP client for the language provider (restcountries.com by default).
- `DisableLanguage`: skips fetching the primary language of the country.
- `Cache`: the cache used to store the geolocation information. Defaults to an in-memory cache.
- `DisableCache`: looks up every request instead of caching.

`iris.WithErrorHandler` is a middleware, not a geolocation option: it registers an error handler for the requests that follow it. Errors that may occur during the fetching of the geolocation information are reported through it.

#### API Key

The default rate limit for the free version is 45 requests per minute (1,440 requests daily). For more information on pricing and to see the available plans, visit [IP-API Pricing](https://members.ip-api.com/#pricing). The paid plan offers unlimited requests and is very affordable compared to other services (e.g., ipgeolocation.io).

You can provide a custom API key for the ip-api.com API by setting the `APIKey`.

```go
app.Use(geolocation.New(geolocation.Options{APIKey: "my-api-key"}))
```

#### Disable Language Fetching

You can disable fetching the primary language of the country by setting `DisableLanguage`.

```go
app.Use(geolocation.New(geolocation.Options{DisableLanguage: true}))
```

#### Custom Error Handler

You can customize the error handling by implementing the `func(iris.Context, error)` function and registering it using the Application's or a Router's or a Route's Hadler's `ContextOptions`. All geolocation's middleware errors are of type `geolocation.Err` and can be checked using the `errors.Is` function.

`errors` below is `github.com/kataras/iris/v14/rest/errors`, which re-exports the standard `errors.Is` alongside the canonical error codes, so one import covers the check and the response.

```go
errorHandler:= func(ctx iris.Context, err error) {
    if errors.Is(err , geolocation.Err) {
        // The error code carries the 401 and stops the chain.
        errors.Unauthenticated.Message(ctx, "geolocation lookup is not authorized")
        return
	}

    // [...rest of your error handling code...]
}
app.ContextOptions().ErrorHandler = errorHandler
app.Use(geolocation.New(geolocation.Options{}))
// [...]
```

Per-Router:

```go
usersAPI := app.Router("/users")
usersAPI.ContextOptions().ErrorHandler = errorHandler
usersAPI.Use(geolocation.New(geolocation.Options{}))
usersAPI.Get("/", ...)
// [...]
```

Per-Route using the `iris.WithErrorHandler` middleware:

```go
app.Get("/", iris.WithErrorHandler(errorHandler), geolocation.New(geolocation.Options{}), mainHandler)
// [...]
```

### Cache

The geolocation middleware supports caching to store geolocation information and reduce the number of external API requests.

#### Default Memory Cache

By default, the middleware uses an in-memory cache with a **dynamic maximum count based on the system's available memory**. You can configure the cache with a maximum count of items. When the maximum count is reached, the cache will clean up the oldest items.

#### Custom Memory Cache

You can provide a custom maximum count for the memory cache:

```go
app.Use(geolocation.New(geolocation.Options{Cache: cache.NewMemoryCache[geolocation.Info](11000)}))
```

where `cache` is `github.com/kataras/iris/v14/core/pkg/cache`.

#### Disable Cache

You can disable the cache by setting `DisableCache`:

```go
app.Use(geolocation.New(geolocation.Options{DisableCache: true}))
```

### Fetch Usage

You can also skip the middleware registration and fetch the geolocation information manually using the `Fetch` function:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/geolocation"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		info, err := geolocation.Fetch(ctx)
		if err != nil { // here you handle the error.
			errors.NotFound.Err(ctx, err)
			return
		}

		ctx.OK(info, nil)
	})

	app.Listen(":8080")
}
```

### Output Example

Both examples will output the geolocation information in JSON format:

```json
{
    "status": "success",
    "country": "Greece",
    "countryCode": "GR",
    "region": "G",
    "regionName": "North Greece",
    "city": "Xanthi",
    "zip": "",
    "lat": 38.6214,
    "lon": 21.4078,
    "timezone": "Europe/Athens",
    "isp": "Cosmote Mobile Telecommunication S.A",
    "org": "Cosmote Mobile Telecommunications S.A.",
    "as": "AS29247 Ote SA (Hellenic Telecommunications Organisation)",
    "query": "107.178.202.219",
    "language": "Greek"
}
```
