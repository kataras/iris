package client

import (
	"net/http/httptrace"
	"net/url"

	"github.com/kataras/httpclient"
)

// RequestHeader adds or sets (if overridePrev is true) a header to the request.
func RequestHeader(overridePrev bool, key string, values ...string) RequestOption {
	return httpclient.RequestHeader(overridePrev, key, values...)
}

// RequestAuthorization sets an Authorization request header.
func RequestAuthorization(value string) RequestOption {
	return httpclient.RequestAuthorization(value)
}

// RequestAuthorizationBearer sets an Authorization: Bearer $token request header.
func RequestAuthorizationBearer(accessToken string) RequestOption {
	return httpclient.RequestAuthorizationBearer(accessToken)
}

// RequestQuery adds a set of URL query parameters to the request.
func RequestQuery(query url.Values) RequestOption {
	return httpclient.RequestQuery(query)
}

// RequestParam sets a single URL query parameter to the request.
func RequestParam(key string, values ...string) RequestOption {
	return httpclient.RequestParam(key, values...)
}

// RequestRateLimit makes the request wait on the named limiter registered with
// RateLimitFor, RateLimitForPerMinute or RateLimiterFor. An unknown key is a
// no-op.
func RequestRateLimit(key string) RequestOption {
	return httpclient.RequestRateLimit(key)
}

// ClientTrace adds a client trace to the request.
func ClientTrace(clientTrace *httptrace.ClientTrace) RequestOption {
	return httpclient.ClientTrace(clientTrace)
}
