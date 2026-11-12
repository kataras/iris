package client

import (
	"net/http"
	"net/url"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/httpclient"
	"golang.org/x/oauth2"

	json "encoding/json/v2"
)

// All the builtin client options live here, for easy discovery. Each one is
// its github.com/kataras/httpclient namesake; the doc there carries the details.

// BaseURL registers the base URL of this client. All of its methods prepend
// this URL to the given path, with a trailing "/" trimmed from the base.
func BaseURL(uri string) Option { return httpclient.BaseURL(uri) }

// Timeout specifies a time limit for requests made by this Client, including
// connection time, redirects and reading the body. Zero, the default, means no
// timeout. Set one for anything talking to the internet.
func Timeout(timeout time.Duration) Option { return httpclient.Timeout(timeout) }

// DialTimeout specifies a time limit for establishing connections, name
// resolution included. It adjusts the Client's *http.Transport, cloning
// http.DefaultTransport when there is none yet, so proxy settings from the
// environment and the connection pool defaults are kept.
func DialTimeout(timeout time.Duration) Option { return httpclient.DialTimeout(timeout) }

// Transport sets the http.RoundTripper the Client sends through, for an
// instrumented transport or the transport of an httptest.NewTestServer.
// Transport, DialTimeout and Handler all write the same field, the last wins.
func Transport(rt http.RoundTripper) Option { return httpclient.Transport(rt) }

// Handler specifies an iris.Application or any http.Handler that can be tested
// through this Client: requests are served in-process, without a network. The
// response is recorded in full before the Client sees it, so a streaming
// handler is better tested through Transport and an httptest.NewTestServer.
func Handler(h http.Handler) Option { return httpclient.Handler(h) }

// OAuth2 sends every request with a token from "src", wrapping the Client's
// transport in an oauth2.Transport. Give Transport, Handler or DialTimeout
// before it.
//
//	c := client.New(client.OAuth2(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})))
//
// or with a configuration that refreshes tokens on its own:
//
//	config := &oauth2.Config{...}
//	c := client.New(client.OAuth2(config.TokenSource(ctx, token)))
func OAuth2(src oauth2.TokenSource) Option { return httpclient.OAuth2(src) }

// PersistentRequestOptions adds one or more request options that every
// request made by this Client will respect.
func PersistentRequestOptions(reqOpts ...RequestOption) Option {
	return httpclient.PersistentRequestOptions(reqOpts...)
}

// RateLimit configures how many requests per second this Client may send.
// Every attempt, retries included, waits on the limiter. A Clone builds its
// own limiter. Zero or less disables rate limiting.
func RateLimit(requestsPerSecond int) Option { return httpclient.RateLimit(requestsPerSecond) }

// RateLimitPerMinute is RateLimit expressed per minute, with a burst of the
// whole minute's worth.
func RateLimitPerMinute(requestsPerMinute int) Option {
	return httpclient.RateLimitPerMinute(requestsPerMinute)
}

// RateLimiter sets a Limiter you own as the client-wide limiter, so several
// Clients can share one budget. A Clone shares it too. Nil disables limiting.
func RateLimiter(l Limiter) Option { return httpclient.RateLimiter(l) }

// RateLimitFor registers a named limiter on top of the client-wide one.
// Requests select it with the RequestRateLimit request option:
//
//	c := client.New(client.RateLimit(20), client.RateLimitFor("search", 2))
//	results, err := c.BindJSON[Results](ctx, "GET", "/search", nil, client.RequestRateLimit("search"))
func RateLimitFor(key string, requestsPerSecond int) Option {
	return httpclient.RateLimitFor(key, requestsPerSecond)
}

// RateLimitForPerMinute is RateLimitFor expressed per minute.
func RateLimitForPerMinute(key string, requestsPerMinute int) Option {
	return httpclient.RateLimitForPerMinute(key, requestsPerMinute)
}

// RateLimiterFor registers a Limiter you own under a name. Nil removes it.
func RateLimiterFor(key string, l Limiter) Option { return httpclient.RateLimiterFor(key, l) }

// NewRateLimiter returns a Limiter allowing requestsPerSecond requests per
// second with a burst of the same size, to hand to RateLimiter or
// RateLimiterFor. Zero or less returns nil.
func NewRateLimiter(requestsPerSecond int) Limiter {
	return httpclient.NewRateLimiter(requestsPerSecond)
}

// NewRateLimiterPerMinute is NewRateLimiter expressed per minute.
func NewRateLimiterPerMinute(requestsPerMinute int) Limiter {
	return httpclient.NewRateLimiterPerMinute(requestsPerMinute)
}

// Retry enables automatic retries according to the policy: network errors on
// idempotent methods and the 429, 502, 503 and 504 statuses by default, with
// exponential backoff, a Retry-After header winning when present. Buffered
// request bodies are replayed. A policy with MaxAttempts below 2 disables it.
func Retry(policy RetryPolicy) Option { return httpclient.Retry(policy) }

// RedactQueryParams registers query parameter names (e.g. "apiKey") whose
// values must never appear in APIError messages or Debug output. The request
// itself is not modified.
func RedactQueryParams(keys ...string) Option { return httpclient.RedactQueryParams(keys...) }

// RedactHeaders registers extra header names whose values are replaced by
// "REDACTED" in Debug output. Authorization, Proxy-Authorization, Cookie and
// Set-Cookie are always redacted.
func RedactHeaders(names ...string) Option { return httpclient.RedactHeaders(names...) }

// RedactURL returns the string form of "u" with the values of the given query
// parameters replaced by "REDACTED" and any userinfo password masked, for your
// own log lines.
func RedactURL(u *url.URL, keys ...string) string { return httpclient.RedactURL(u, keys...) }

// JSONOptions replaces the encoding/json/v2 options this Client uses to encode
// payloads and decode bodies. The default is the Iris policy of
// core/pkg/jsonutil; options given here replace it for both directions, they
// are not merged with it. Join them with json.JoinOptions to keep both.
func JSONOptions(opts ...json.Options) Option { return httpclient.JSONOptions(opts...) }

// JSONMarshalOptions replaces the options used to encode request payloads
// (default jsonutil.DefaultMarshalOptions), leaving decoding alone.
func JSONMarshalOptions(opts ...json.Options) Option { return httpclient.JSONMarshalOptions(opts...) }

// JSONUnmarshalOptions replaces the options used to decode response bodies
// (default jsonutil.DefaultUnmarshalOptions), leaving encoding alone.
func JSONUnmarshalOptions(opts ...json.Options) Option {
	return httpclient.JSONUnmarshalOptions(opts...)
}

// DefaultDebugLogger is the logger Debug uses when handed nil: a golog child
// named "Iris HTTP Client" at debug level. Replace it before creating clients
// to route the dumps elsewhere.
var DefaultDebugLogger DebugLogger = golog.Child("Iris HTTP Client: ").SetLevel("debug")

// Debug enables the client's debug logger. It dumps each request right before
// it is sent and each response right after it is received, with redacted
// query parameters and credential headers replaced by "REDACTED". A nil
// logger uses DefaultDebugLogger.
//
//	c := client.New(client.Debug(nil))         // golog
//	c := client.New(client.Debug(app.Logger())) // the application's logger
func Debug(logger DebugLogger) Option {
	if logger == nil {
		logger = DefaultDebugLogger
	}

	return httpclient.Debug(logger)
}
