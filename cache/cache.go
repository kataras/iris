/* Package cache provides server-side caching capabilities with rich support of options and rules.

Use it for server-side caching, see the `iris#Cache304` for an alternative approach that
may fit your needs most.

Example code:


		 import (
		 	"time"

		 	"github.com/kataras/iris"
		 	"github.com/kataras/iris/cache"
		 )

		 func main(){
		 	app := iris.Default()
		 	middleware := cache.Handler(2 *time.Minute)
		 	app.Get("/hello", middleware, h)
		 	app.Run(iris.Addr(":8080"))
		 }

		 func h(ctx iris.Context) {
		 	ctx.HTML("<h1> Hello, this should be cached. Every 2 minutes it will be refreshed, check your browser's inspector</h1>")
		 }
*/

package cache

import (
	"time"

	"github.com/kataras/iris/cache/client"
	"github.com/kataras/iris/context"
)

// Cache accepts the cache expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns context.Handler, which you can use as your default router or per-route handler
//
// All types of response can be cached, templates, json, text, anything.
//
// Use it for server-side caching, see the `iris#Cache304` for an alternative approach that
// may fit your needs most.
//
// You can add validators with this function.
func Cache(expiration time.Duration) *client.Handler {
	return client.NewHandler(expiration)
}

// Handler accepts one single parameter:
// the cache expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns context.Handler.
//
// All types of response can be cached, templates, json, text, anything.
//
// Use it for server-side caching, see the `iris#Cache304` for an alternative approach that
// may fit your needs most.
//
// it returns a context.Handler which can be used as a middleware, for more options use the `Cache`.
//
// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/#caching
func Handler(expiration time.Duration) context.Handler {
	h := Cache(expiration).ServeHTTP
	return h
}

var (
	// NoCache disables the cache for a particular request,
	// can be used as a middleware or called manually from the handler.
	NoCache = client.NoCache
)
