# Cache Middleware for Iris

This middleware sets the HTTP caching headers on your responses: `ETag`, `Cache-Control`, `Last-Modified`, `Expires`, `Pragma`, `Vary` and `Proxy-Authenticate`. It also answers a conditional request whose `If-None-Match` still matches with 304, so the body never goes out twice.

For a server-side response store rather than cache headers, see [httpcache](httpcache).

## Features of Cache Middleware

The cache middleware offers a variety of features to control HTTP caching behavior. These features include:

1. **ETag Headers**: Used for cache validation.
2. **Cache-Control Headers**: Directives to control caching behavior.
3. **Last-Modified Headers**: Used for cache validation.
4. **Expires Headers**: Specifies the expiration time of the cached resource.
5. **Pragma Headers**: HTTP/1.0 header for backward compatibility.
6. **Vary Headers**: Specifies how caches should treat the response based on request headers.
7. **Proxy-Authenticate Headers**: Specifies the authentication method for accessing a resource behind a proxy server.

## Installation

To use the `cache` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/cache"
```

## Usage

### Basic Setup

To use the cache middleware, you need to import the necessary packages and configure the middleware with your desired options. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cache"
)

func main() {
	app := iris.New()
	app.Use(cache.New(cache.DefaultOptions))

	app.Get("/foo", func(ctx iris.Context) {
		ctx.WriteString("Hello, world!")
	})

	app.Listen(":8080")
}
```

### Configuration Options

The `Options` struct controls which headers the middleware sets:

- **ETag**: Whether to use ETag headers for cache validation.
- **CacheControl**: The Cache-Control directives to include in the response headers.
- **LastModified**: Whether to use Last-Modified headers for cache validation.
- **Expires**: Duration for the Expires header.
- **Pragma**: The Pragma directive to include in the response headers.
- **Vary**: The Vary header specifies how caches should treat the response based on request headers.
- **ProxyAuthenticate**: The Proxy-Authenticate header specifies the authentication method for accessing a resource behind a proxy server.

### Example Configuration

Here is an example of how to configure the cache middleware with custom options:

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cache"
)

func main() {
	app := iris.New()
	options := cache.Options{
		ETag:         true,
		CacheControl: cache.NewCacheControlBuilder().Public().MaxAge(time.Hour).SMaxAge(2 * time.Hour).MustRevalidate().ProxyRevalidate().Build(),
		LastModified: true,
		Expires:      24 * time.Hour,
		Pragma:       cache.NoPragmaCache,
		Vary:         cache.NewVaryBuilder().AcceptEncoding().Build(),
		ProxyAuthenticate: cache.NewProxyAuthenticateBuilder().Basic("Restricted Area").Build(),
	}
	app.Use(cache.New(options))

	app.Get("/foo", func(ctx iris.Context) {
		ctx.WriteString("Hello, world!")
	})

	app.Listen(":8080")
}
```

### Cache-Control Directives

The `CacheControlBuilder` helps you build the `Cache-Control` header value. Here are some of the available methods:

- **Public**: Indicates that the response may be cached by any cache.
- **Private**: Indicates that the response is intended for a single user and must not be cached by shared caches.
- **NoCache**: Indicates that the response must not be used from the cache without revalidation with the origin server.
- **NoStore**: Indicates that the response must not be stored in any cache.
- **MaxAge**: Specifies the maximum amount of time a resource is considered fresh. Takes a `time.Duration`, written in whole seconds; a value above zero and below one second panics (`MaxAge(3600)` is 3600 nanoseconds, write `MaxAge(time.Hour)`).
- **SMaxAge**: Specifies the maximum amount of time a resource is considered fresh in shared caches. Takes a `time.Duration`, same rules as `MaxAge`.
- **MustRevalidate**: Indicates that once a resource becomes stale, it must be validated with the origin server before being used.
- **ProxyRevalidate**: Indicates that shared caches (e.g., proxies) must revalidate the response with the origin server once it becomes stale.

### Vary Header

The `VaryBuilder` helps you build the `Vary` header value. Here are some of the available methods:

- **Accept**: Sets the Vary directive to 'Accept'.
- **AcceptEncoding**: Sets the Vary directive to 'Accept-Encoding'.
- **AcceptLanguage**: Sets the Vary directive to 'Accept-Language'.
- **UserAgent**: Sets the Vary directive to 'User-Agent'.

### Proxy-Authenticate Header

The `ProxyAuthenticateBuilder` helps you build the `Proxy-Authenticate` header value. Here are some of the available methods:

- **Basic**: Sets the Proxy-Authenticate scheme to 'Basic'.
- **Digest**: Sets the Proxy-Authenticate scheme to 'Digest'.

## Testing

To check that your caching middleware behaves as expected, write tests with the `httptest` package that ships with Iris. Here is an example test:

```go
package cache_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cache"
)

func TestCache(t *testing.T) {
	app := iris.New()
	options := cache.DefaultOptions
	app.Use(cache.New(options))

	app.Get("/foo", func(ctx iris.Context) {
		ctx.WriteString("Hello, world!")
	})

	resp := httptest.New(t, app).GET("/foo").Expect().Status(http.StatusOK)

	if got, want := resp.Header("Cache-Control").Raw(), options.CacheControl; got != want {
		t.Errorf("Cache-Control header = %q, want %q", got, want)
	}

	if got, want := resp.Header("Pragma").Raw(), options.Pragma; got != want {
		t.Errorf("Pragma header = %q, want %q", got, want)
	}

	if got, want := resp.Header("Vary").Raw(), options.Vary; got != want {
		t.Errorf("Vary header = %q, want %q", got, want)
	}

	if got := resp.Header("ETag").Raw(); got == "" {
		t.Errorf("Expected ETag header to be set")
	}

	expires := time.Now().Add(options.Expires).Format(http.TimeFormat)
	if got, want := resp.Header("Expires").Raw(), expires; got != want {
		t.Errorf("Expires header = %q, want %q", got, want)
	}

	if got, want := resp.Header("Proxy-Authenticate").Raw(), options.ProxyAuthenticate; got != want {
		t.Errorf("Proxy-Authenticate header = %q, want %q", got, want)
	}
}
```
