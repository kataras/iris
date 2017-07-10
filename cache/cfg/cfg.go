package cfg

import "time"

// The constants be used by both client and server
var (
	FailStatus            = 400
	SuccessStatus         = 200
	ContentHTML           = "text/html; charset=utf-8"
	ContentTypeHeader     = "Content-Type"
	StatusCodeHeader      = "Status"
	QueryCacheKey         = "cache_key"
	QueryCacheDuration    = "cache_duration"
	QueryCacheStatusCode  = "cache_status_code"
	QueryCacheContentType = "cache_content_type"
	RequestCacheTimeout   = 5 * time.Second
)

// NoCacheHeader is the static header key which is setted to the response when NoCache is called,
// used inside nethttp and fhttp Skippers.
var NoCacheHeader = "X-No-Cache"

// MinimumCacheDuration is the minimum duration from time.Now
// which is allowed between cache save and cache clear
var MinimumCacheDuration = 2 * time.Second
