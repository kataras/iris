// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/* Package cache provides cache capabilities with rich support of options and rules.

Example code:


		 import (
		 	"time"

		 	"github.com/kataras/iris"
		 	"github.com/kataras/iris/cache"
		 	"github.com/kataras/iris/context"
		 )

		 func main(){
		 	app := iris.Default()
		 	cachedHandler := cache.CacheHandler(h, 2 *time.Minute)
		 	app.Get("/hello", cachedHandler)
		 	app.Run(iris.Addr(":8080"))
		 }

		 func h(ctx context.Context) {
		 	ctx.HTML("<h1> Hello, this should be cached. Every 2 minutes it will be refreshed, check your browser's inspector</h1>")
		 }
*/

package cache

import (
	"time"

	"github.com/kataras/iris/cache/client"
	"github.com/kataras/iris/context"
)

// Cache accepts two parameters
// first is the context.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns context.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// You can add validators with this function
func Cache(bodyHandler context.Handler, expiration time.Duration) *client.Handler {
	return client.NewHandler(bodyHandler, expiration)
}

// CacheHandler accepts two parameters
// first is the context.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns context.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// it returns the context.Handler, for more options use the .Cache .
func CacheHandler(bodyHandler context.Handler, expiration time.Duration) context.Handler {
	return Cache(bodyHandler, expiration).ServeHTTP
}

var (
	// NoCache called when a particular handler is not valid for cache.
	// If this function called inside a handler then the handler is not cached
	// even if it's surrounded with the Cache/CacheFunc/CacheRemote wrappers.
	NoCache = client.NoCache
)
