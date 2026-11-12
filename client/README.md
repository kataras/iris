# Iris HTTP Client

The [`client`](https://github.com/kataras/iris/tree/main/client) package calls other HTTP services. You configure a base URL, timeouts, rate limits and a retry policy once, then make requests against it. Request and response handling is pluggable, so you can add your own middleware to every call.

It is a thin layer over [`github.com/kataras/httpclient`](https://github.com/kataras/httpclient), the author's standalone client. Every type, option and function of that module is available here under the same name, so an Iris application imports only this package. What this package adds is the framework's defaults: request and response bodies follow the Iris JSON policy (`core/pkg/jsonutil`, the same one `ctx.JSON` and `ctx.ReadJSON` use on the server side), and `Debug(nil)` logs through golog.

## Features

- Base URL, timeouts, persistent request options
- Typed responses through generic methods (`BindJSON[T]`, `BindPlain[T]`)
- Rate limits per client, per endpoint, or one budget shared by several clients
- Retries with exponential backoff, `Retry-After` honoured
- Query parameter and header redaction in error messages and debug output
- OAuth2 tokens through `golang.org/x/oauth2`
- Request handlers (middleware) around every call
- JSON, form and multipart file uploads
- Errors that `rest/errors.HandleAPIError` forwards to your own client

## Installation

```go
import "github.com/kataras/iris/v14/client"
```

## Configuration

### Base URL and timeouts

```go
c := client.New(
    client.BaseURL("https://api.example.com"),
    client.Timeout(15 * time.Second),      // the whole request, body included
    client.DialTimeout(10 * time.Second),  // connecting only
)
```

There is no default timeout. Set one for anything talking to the internet.

### Persistent request options

Options that apply to every request:

```go
c := client.New(
    client.PersistentRequestOptions(
        client.RequestAuthorizationBearer("YOUR_TOKEN"),
        client.RequestParam("apiKey", key),
    ),
)
```

### Rate limiting

```go
c := client.New(client.RateLimit(10)) // 10 requests per second
```

`RateLimit` covers the whole API. When one endpoint has a tighter budget, register a named limiter and tag the calls that use it. Every call sharing the key shares the budget, retries included:

```go
c := client.New(
    client.RateLimit(20),                  // the whole API
    client.RateLimitFor("search", 2),      // this endpoint
)

results, err := c.BindJSON[Results](ctx, http.MethodGet, "/search", nil,
    client.RequestRateLimit("search"))
```

A `Clone` builds its own limiters. When several clients must share one upstream quota, build the limiter yourself and hand the same one to each with `RateLimiter(l)` or `RateLimiterFor(key, l)`:

```go
var hostLimiter = client.NewRateLimiter(20)

content := client.New(client.BaseURL(host+"/rest"), client.RateLimiter(hostLimiter))
catalogue := client.New(client.BaseURL(host), client.RateLimiter(hostLimiter))
```

A `Limiter` is any type with `Wait(ctx) error`, so a distributed limiter of your own fits too.

### Retries

Retrying is opt-in. By default only network errors on idempotent methods (GET, HEAD, OPTIONS, TRACE, PUT, DELETE and QUERY) and the 429, 502, 503 and 504 statuses are retried; buffered request bodies are replayed.

```go
c := client.New(
    client.BaseURL("https://api.example.com"),
    client.Retry(client.RetryPolicy{
        MaxAttempts:    3,
        InitialBackoff: 500 * time.Millisecond,
        MaxBackoff:     10 * time.Second,
        Jitter:         true,
    }),
)
```

Every attempt waits on the rate limiter and is visible to the request handlers, so `Debug` output shows the failed attempts as well.

### OAuth2

```go
c := client.New(client.OAuth2(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})))
```

Or with an `oauth2.Config` that refreshes tokens on its own: `client.OAuth2(config.TokenSource(ctx, token))`. Give `Transport`, `Handler` or `DialTimeout` before it, since it wraps whatever transport is set at that point.

### Request handlers

Run code before and after each request:

```go
type MyRequestHandler struct{}

func (h *MyRequestHandler) BeginRequest(ctx context.Context, req *http.Request) error {
    return nil
}

func (h *MyRequestHandler) EndRequest(ctx context.Context, resp *http.Response, err error) error {
    return nil
}

c := client.New()
c.RegisterRequestHandler(&MyRequestHandler{})
```

A non-nil error from either method stops the call. An `EndRequest` handler that returns the error it was given passes it through.

### Debugging

```go
c := client.New(client.Debug(nil))          // golog, "Iris HTTP Client" child at debug level
c := client.New(client.Debug(app.Logger())) // the application's logger
```

Both dumps, request and response, go to the logger. Values of query parameters registered with `RedactQueryParams`, and the `Authorization`, `Proxy-Authorization`, `Cookie` and `Set-Cookie` headers, are replaced by `REDACTED`:

```go
c := client.New(
    client.PersistentRequestOptions(client.RequestParam("apiKey", key)),
    client.RedactQueryParams("apiKey"),
    client.Debug(nil),
)
```

The same redaction applies to `APIError` messages. Request and response bodies are never scrubbed.

### Testing against a handler

```go
c := client.New(client.Handler(app)) // any http.Handler, an *iris.Application included
```

Requests are served in-process. The response is recorded in full before the client sees it, so for a streaming handler use `client.Transport(srv.Client().Transport)` with an `httptest.NewTestServer` instead.

## Usage

### Sending requests

```go
type Payload struct {
    Name string `json:"name"`
}

resp, err := c.JSON(context.Background(), http.MethodPost, "/endpoint", Payload{Name: "John"})
if err != nil {
    panic(err)
}
c.DrainResponseBody(resp)
```

```go
resp, err := c.Form(context.Background(), http.MethodPost, "/endpoint", url.Values{
    "field1": {"value1"},
    "field2": {"value2"},
})
if err != nil {
    panic(err)
}
c.DrainResponseBody(resp)
```

`Do`, `JSON`, `Form` and `Uploader.Upload` hand you the `*http.Response` with its body still open and do not inspect the status. Closing it is your job, and `DrainResponseBody` is how to do it so the connection can be reused. Everything under "Reading responses" closes it for you.

### Reading responses

`BindJSON` names the response type at the call site and returns it:

```go
type Response struct {
    Data string `json:"data"`
}

respData, err := c.BindJSON[Response](context.Background(), http.MethodGet, "/endpoint", nil)
```

The type parameter is the response type; the arguments are the request method, URL path, an optional payload (sent as JSON when non-nil) and any request options. A status of 400 or above comes back as an `APIError`.

`ReadJSON` fills a value you already hold, which is what you want for a long-lived field or a pooled struct:

```go
var respData Response
err := c.ReadJSON(context.Background(), &respData, http.MethodGet, "/endpoint", nil)
```

`BindPlain[T]` and `ReadPlain` do the same for plain text, into a `string`, `[]byte`, `int`, `int64` or `float64`:

```go
respText, err := c.BindPlain[string](context.Background(), http.MethodGet, "/endpoint", nil)
```

`Call` is for endpoints whose success body carries nothing. It returns the `APIError` on a failure status and nil otherwise:

```go
err := c.Call(context.Background(), http.MethodDelete, "/endpoint/42", nil)
```

### Paginated lists

An Iris list endpoint answers a `Page[T]` (`CurrentPage`, `TotalPages`, `TotalItems`, `HasNextPage`, `Items`), reads the page and size from the URL and the filter from the body. `List[T]` fetches one page. It sends a QUERY request (RFC 10008), which the client retries like a GET:

```go
page, err := client.List[Book](ctx, c, "/books", client.PageOptions{Page: 1, Size: 20}, filter)
```

`Items[T]` walks every page and hands you one item at a time. It stops when the server has no next page or when you break out of the loop:

```go
for book, err := range client.Items[Book](ctx, c, "/books", filter, client.PageSize(50)) {
    if err != nil {
        return err
    }
    fmt.Println(book.Title)
}
```

A server that still lists over POST takes `client.ListMethod(http.MethodPost)` as one more option. `PageOptions`, `Page[T]`, `CountResponse[N]` and `ListFilter` are the same types the server uses (`iris.Page[T]` and so on), from the std-only `core/pkg/pagination` package.

An empty JSON body comes back as `io.EOF`, which `IsErrEmptyJSON` also recognises. `Bind[T](resp)` decodes a response you already hold, choosing by `Content-Type`.

### File uploads

```go
uploader := c.NewUploader()
uploader.AddFile("file", "path/to/file.txt")
resp, err := uploader.Upload(context.Background(), http.MethodPost, "/upload")
if err != nil {
    panic(err)
}
c.DrainResponseBody(resp)
```

An `Uploader` sends once; a second `Upload` returns `ErrUploaderClosed`.

## Error handling

### Extracting errors

`Do` returns the response as-is, whatever its status. `ExtractError` wraps a failed response (reading and closing its body) into an `APIError`:

```go
resp, err := c.Do(context.Background(), http.MethodGet, "/endpoint", nil)
if err != nil {
    panic(err) // transport failure, there is no response to extract from.
}

if resp.StatusCode >= 400 {
    apiErr := client.ExtractError(resp)
    fmt.Println(apiErr.Error()) // "https://api.example.com/endpoint: 404 Not Found: {...}"
}
```

The typed readers (`BindJSON`, `ReadJSON`, `BindPlain`, `ReadPlain`, `Call`) do this for you and return the `APIError` as their error. `APIError.Code()` is the status, `APIError.Body` the raw body.

### Decoding errors

`BindError` decodes the body of an `APIError` into any JSON type. When the remote server is an Iris application using the `rest/errors` package, the body is its wire error envelope, so decode straight into `errors.Error`:

```go
import "github.com/kataras/iris/v14/rest/errors"

_, err := c.BindJSON[Response](context.Background(), http.MethodGet, "/endpoint", nil)
if err != nil {
    remoteErr, decodeErr := client.BindError[errors.Error](err)
    if decodeErr != nil {
        panic(decodeErr) // not an APIError, or the body is not JSON.
    }
    fmt.Println(remoteErr.ErrorCode.CanonicalName, remoteErr.Message)
}
```

`DecodeError(err, &dest)` is the pointer form. Inside an Iris handler, `errors.HandleAPIError(ctx, err)` forwards the remote envelope to your own client instead. Because `client.APIError` is the same type as `httpclient.APIError`, that also works for errors produced by services that use `github.com/kataras/httpclient` directly.

### Status codes

`GetErrorCode` reads the status of an `APIError`. It returns 0 for a nil error, for any error that is not an `APIError` (a transport failure from `Do`, for example) and for an `APIError` built without a response, so check for zero before using the result as a status:

```go
_, err := c.BindJSON[Response](context.Background(), http.MethodGet, "/endpoint", nil)
if client.GetErrorCode(err) == http.StatusNotFound {
    // ...
}
```

`GetError(err)` returns the `APIError` itself, found through `errors.As`, so an error wrapped with `%w` is still recognised.

## Building against a local checkout

This package requires a published version of `github.com/kataras/httpclient`. To develop the two together, use a Go workspace from this repository's root; `go.work` is ignored by git and by dependents:

```sh
go work init . ../httpclient
```

## References

- [HTTP - Wikipedia](https://en.wikipedia.org/wiki/Hypertext_Transfer_Protocol)
- [Rate Limiting - Wikipedia](https://en.wikipedia.org/wiki/Rate_limiting)
- [HTTP Status Codes - IANA](https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml)
- [RFC 6749 - The OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749)
