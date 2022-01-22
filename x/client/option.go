package client

import (
	"time"

	"golang.org/x/time/rate"
)

// All the builtin client options should live here, for easy discovery.

type Option func(*Client)

// BaseURL registers the base URL of this client.
// All of its methods will prepend this url.
func BaseURL(uri string) Option {
	return func(c *Client) {
		c.BaseURL = uri
	}
}

// Timeout specifies a time limit for requests made by this
// Client. The timeout includes connection time, any
// redirects, and reading the response body.
// A Timeout of zero means no timeout.
//
// Defaults to 15 seconds.
func Timeout(d time.Duration) Option {
	return func(c *Client) {
		c.HTTPClient.Timeout = d
	}
}

// PersistentRequestOptions adds one or more persistent request options
// that all requests made by this Client will respect.
func PersistentRequestOptions(reqOpts ...RequestOption) Option {
	return func(c *Client) {
		c.PersistentRequestOptions = append(c.PersistentRequestOptions, reqOpts...)
	}
}

// RateLimit configures the rate limit for requests.
//
// Defaults to zero which disables rate limiting.
func RateLimit(requestsPerSecond int) Option {
	return func(c *Client) {
		c.rateLimiter = rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond)
	}
}
