// Package client is an HTTP client for sending requests to remote services and
// decoding their responses.
//
// It is a thin layer over github.com/kataras/httpclient: every type, option and
// function of that package is available here under the same name, so an Iris
// application never imports it directly. What this package adds is the
// framework's defaults. Request payloads and response bodies follow the Iris
// JSON policy of core/pkg/jsonutil (time.Duration as int64 nanoseconds, sorted
// map keys, case-insensitive member matching, duplicate members rejected), the
// same policy the server side uses in ctx.JSON and ctx.ReadJSON, and
// Debug(nil) logs through golog.
//
// The central type is Client, created with New and configured through Options
// such as a base URL, a timeout, persistent request options, rate limiters, a
// retry policy and before/after request handlers:
//
//	c := client.New(
//		client.BaseURL("https://api.example.com/v1"),
//		client.Timeout(30*time.Second),
//		client.PersistentRequestOptions(client.RequestAuthorizationBearer(token)),
//	)
//
// # Reading responses
//
// BindJSON names the response type at the call site and returns it:
//
//	user, err := c.BindJSON[User](ctx, http.MethodGet, "/users/42", nil)
//
// ReadJSON fills a value you already hold, which is what you want for a
// long-lived field or a pooled struct:
//
//	var user User
//	err := c.ReadJSON(ctx, &user, http.MethodGet, "/users/42", nil)
//
// BindPlain and ReadPlain do the same for plain text bodies. Call is for
// endpoints whose success body carries nothing. WriteTo copies the body to a
// writer, forwarding Content-Type and Content-Length when that writer is an
// http.ResponseWriter. All of them close the response body and turn a status of
// 400 or above into an APIError, which rest/errors.HandleAPIError knows how to
// forward to your own client.
//
// # Owning the response body
//
// Do, JSON, Form and Uploader.Upload hand you the *http.Response with its body
// still open. Close it with DrainResponseBody so the connection can be reused.
//
// # Options
//
// Client options, passed to New or Clone:
//
//	BaseURL, Timeout, DialTimeout, Transport, Handler, OAuth2,
//	PersistentRequestOptions,
//	RateLimit, RateLimitPerMinute, RateLimitFor, RateLimitForPerMinute,
//	RateLimiter, RateLimiterFor,
//	Retry, RedactQueryParams, RedactHeaders,
//	JSONOptions, JSONMarshalOptions, JSONUnmarshalOptions, Debug, NoOption
//
// Request options, passed to any request method:
//
//	RequestHeader, RequestAuthorization, RequestAuthorizationBearer,
//	RequestQuery, RequestParam, RequestRateLimit, ClientTrace
package client
