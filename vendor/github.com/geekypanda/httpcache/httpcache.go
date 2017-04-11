// httpcache originally written by @geekypanda, maintained by @kataras,
// based on the, new, improved version: 5.0.0 by @esemplastic.
package httpcache

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/nethttp"
)

// func When(cachedHandler *nethttp.Handler, claimFuncs, validFuncs)
// | We could have something like this
//  but this wouldn't work for 'XXXFunc' & fasthttp because they just returns a function |

// Cache accepts two parameters
// first is the http.Handler which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.Handler, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// You can add validators with this function
func Cache(bodyHandler http.Handler, expiration time.Duration) *nethttp.Handler {
	return nethttp.NewHandler(bodyHandler, expiration)
}

// CacheFunc accepts two parameters
// first is the http.HandlerFunc which you want to cache its result
// the second is, optional, the cache Entry's expiration duration
// if the expiration <=2 seconds then expiration is taken by the "cache-control's maxage" header
// returns an http.HandlerFunc, which you can use as your default router or per-route handler
//
// All type of responses are cached, templates, json, text, anything.
//
// You CAN NOT add validators with this function
func CacheFunc(bodyHandler func(http.ResponseWriter, *http.Request), expiration time.Duration) http.HandlerFunc {
	return Cache(http.HandlerFunc(bodyHandler), expiration).ServeHTTP
}

var (
	// NoCache called when a particular handler is not valid for cache.
	// If this function called inside a handler then the handler is not cached
	// even if it's surrounded with the Cache/CacheFunc/CacheRemote wrappers.
	NoCache = nethttp.NoCache
)
