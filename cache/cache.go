/* Package cache provides server-side caching capabilities with rich support of options and rules.

Use it for server-side caching, see the `iris#Cache304` for an alternative approach that
may fit your needs most.

Example code:


		 import (
		 	"time"

		 	"github.com/kataras/iris/v12"
		 	"github.com/kataras/iris/v12/cache"
		 )

		 func main(){
		 	app := iris.Default()
		 	middleware := cache.Handler(2 *time.Minute)
		 	app.Get("/hello", middleware, h)
		 	app.Listen(":8080")
		 }

		 func h(ctx iris.Context) {
		 	ctx.HTML("<h1> Hello, this should be cached. Every 2 minutes it will be refreshed, check your browser's inspector</h1>")
		 }
*/

package cache

import (
	"time"

	"github.com/kataras/iris/v12/cache/client"
	"github.com/kataras/iris/v12/context"
)

// WithKey sets a custom entry key for cached pages.
// Should be prepended to the cache handler.
//
// Usage:
// app.Get("/", cache.WithKey("custom-key"), cache.Handler(time.Minute), mainHandler)
func WithKey(key string) context.Handler {
	return func(ctx *context.Context) {
		client.SetKey(ctx, key)
		ctx.Next()
	}
}

// DefaultMaxAge is a function which returns the
// `context#MaxAge` as time.Duration.
// It's the default expiration function for the cache handler.
var DefaultMaxAge = func(ctx *context.Context) time.Duration {
	return time.Duration(ctx.MaxAge()) * time.Second
}

// MaxAge is a shortcut to set a simple duration as a MaxAgeFunc.
//
// Usage:
// app.Get("/", cache.Cache(cache.MaxAge(1*time.Minute), mainHandler)
func MaxAge(dur time.Duration) client.MaxAgeFunc {
	return func(*context.Context) time.Duration {
		return dur
	}
}

// Cache accepts the cache expiration duration.
// If the "maxAgeFunc" input argument is nil,
// then expiration is taken by the "cache-control's maxage" header.
// Returns a Handler structure which you can use to customize cache further.
//
// All types of response can be cached, templates, json, text, anything.
//
// Use it for server-side caching, see the `iris#Cache304` for an alternative approach that
// may be more suited to your needs.
//
// You can add validators with this function.
func Cache(maxAgeFunc client.MaxAgeFunc) *client.Handler {
	if maxAgeFunc == nil {
		maxAgeFunc = DefaultMaxAge
	}

	return client.NewHandler(maxAgeFunc)
}

// Handler like `Cache` but returns an Iris Handler to be used as a middleware.
// For more options use the `Cache`.
//
// Examples can be found at: https://github.com/kataras/iris/tree/main/_examples/response-writer/cache
func Handler(expiration time.Duration) context.Handler {
	maxAgeFunc := func(*context.Context) time.Duration {
		return expiration
	}

	h := Cache(maxAgeFunc).ServeHTTP
	return h
}
