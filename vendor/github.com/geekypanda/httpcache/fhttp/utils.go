package fhttp

import (
	"time"

	"github.com/geekypanda/httpcache/entry"
	"github.com/valyala/fasthttp"
)

// GetMaxAge parses the "Cache-Control" header
// and returns a LifeChanger which can be passed
// to the response's Reset
func GetMaxAge(reqCtx *fasthttp.RequestCtx) entry.LifeChanger {
	return func() time.Duration {
		cacheControlHeader := string(reqCtx.Request.Header.Peek("Cache-Control"))
		// headerCacheDur returns the seconds
		headerCacheDur := entry.ParseMaxAge(cacheControlHeader)
		return time.Duration(headerCacheDur) * time.Second
	}
}
