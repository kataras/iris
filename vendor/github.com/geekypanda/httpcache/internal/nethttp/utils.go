package nethttp

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/internal"
)

// GetMaxAge parses the "Cache-Control" header
// and returns a LifeChanger which can be passed
// to the response's Reset
func GetMaxAge(r *http.Request) internal.LifeChanger {
	return func() time.Duration {
		cacheControlHeader := r.Header.Get("Cache-Control")
		// headerCacheDur returns the seconds
		headerCacheDur := internal.ParseMaxAge(cacheControlHeader)
		return time.Duration(headerCacheDur) * time.Second
	}
}
