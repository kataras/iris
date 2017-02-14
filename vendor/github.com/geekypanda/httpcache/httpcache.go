package httpcache

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/internal/nethttp"
	"github.com/geekypanda/httpcache/internal/server"
)

const (
	// Version is the release version number of the httpcache package.
	Version = "0.0.5-for-irisv6" // Edited version of geekypanda/httpcache to remove all fasthttp things, irisv6 users doesn't need to download that too.
)

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

// distributed

// ListenAndServe receives a network address and starts a server
// with a remote server cache handler registered to it
// which handles remote client-side cache handlers
// client should register its handlers with the RemoteCache functions
//
// Note: It doesn't starts the server,
func ListenAndServe(addr string) error {
	return server.New(addr, nil).ListenAndServe()
}

// CacheRemote receives a handler, its cache expiration and
// the remote address of the remote cache server(look ListenAndServe)
// returns a remote-cached handler
//
// You can add validators with this function
func CacheRemote(bodyHandler http.Handler, expiration time.Duration, remoteServerAddr string) *nethttp.ClientHandler {
	return nethttp.NewClientHandler(bodyHandler, expiration, remoteServerAddr)
}

// CacheRemoteFunc receives a handler function, its cache expiration and
// the remote address of the remote cache server(look ListenAndServe)
// returns a remote-cached handler function
//
// You CAN NOT add validators with this function
func CacheRemoteFunc(bodyHandler func(http.ResponseWriter, *http.Request), expiration time.Duration, remoteServerAddr string) http.HandlerFunc {
	return CacheRemote(http.HandlerFunc(bodyHandler), expiration, remoteServerAddr).ServeHTTP
}

var (
	// NoCache called when a particular handler is not valid for cache.
	// If this function called inside a handler then the handler is not cached
	// even if it's surrounded with the Cache/CacheFunc/CacheRemote wrappers.
	NoCache = nethttp.NoCache
)
