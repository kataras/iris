# HTTP Client

## Overview

The `client` package is an HTTP client for calling external services from your Go code: REST APIs, webhooks, other microservices. It wraps `net/http` with a base URL, persistent request options, rate limiting, retries, request/response hooks, secret redaction and helpers that marshal and unmarshal JSON, form and multipart bodies.

It is a thin layer over `github.com/kataras/httpclient`, the author's standalone client. Every type, option and function of that module is re-exported here under the same name, so an Iris application imports only this package. What the Iris package adds is the framework's defaults: bodies follow the shared JSON policy of `core/pkg/jsonutil`, and `Debug(nil)` logs through golog.

Import:

```go
import "github.com/kataras/iris/v14/client"
```

The default content type to send and receive data is JSON. Every request carries an `Accept: application/json` header unless you override it.

## Creating a Client

`client.New(opts ...Option)` returns a `*client.Client`:

```go
c := client.New(
    client.BaseURL("https://api.example.com"),
    client.Timeout(15*time.Second),
    client.RateLimit(10), // 10 requests per second
    client.PersistentRequestOptions(
        client.RequestAuthorizationBearer("YOUR_TOKEN"),
    ),
)
```

### Options

| Option | Description |
|--------|-------------|
| `BaseURL(uri string)` | Prepended to every request path (plain string concatenation, a trailing `/` on the base is trimmed, so start paths with `/`) |
| `Timeout(d time.Duration)` | Total request timeout (connection, redirects, body read). Zero, the default, means no timeout |
| `DialTimeout(d time.Duration)` | Connection-establishment timeout. Adjusts the `*http.Transport`, cloning `http.DefaultTransport` when there is none, so `HTTP_PROXY` and pool defaults are kept |
| `Transport(rt http.RoundTripper)` | The transport to send through, for instrumentation or an `httptest.NewTestServer` |
| `Handler(h http.Handler)` | In-process transport for testing against an `http.Handler`, an `*iris.Application` included. The response is recorded in full first, so streaming handlers need `Transport` |
| `OAuth2(src oauth2.TokenSource)` | Wraps the transport with `golang.org/x/oauth2` token handling. Give `Transport`, `Handler` or `DialTimeout` before it |
| `PersistentRequestOptions(reqOpts ...RequestOption)` | Request options applied to every request |
| `RateLimit(requestsPerSecond int)` | Client-wide rate limiter. Zero disables limiting. A `Clone` builds its own |
| `RateLimitPerMinute(requestsPerMinute int)` | The same, per minute, with a burst of the whole minute |
| `RateLimiter(l Limiter)` | A limiter you own, shared by every client given the same one. Nil disables |
| `RateLimitFor(key string, requestsPerSecond int)` | A named limiter for one endpoint; requests select it with `RequestRateLimit(key)` |
| `RateLimitForPerMinute(key string, requestsPerMinute int)` | The same, per minute |
| `RateLimiterFor(key string, l Limiter)` | A named limiter you own. Nil removes it |
| `Retry(policy RetryPolicy)` | Automatic retries with exponential backoff. `MaxAttempts` below 2 disables |
| `RedactQueryParams(keys ...string)` | Query parameter values replaced by `REDACTED` in `APIError` messages and `Debug` output |
| `RedactHeaders(names ...string)` | Extra headers scrubbed from `Debug` output. `Authorization`, `Proxy-Authorization`, `Cookie` and `Set-Cookie` always are |
| `JSONOptions(opts ...json.Options)` | Replaces the `encoding/json/v2` options for both encoding and decoding |
| `JSONMarshalOptions(opts ...json.Options)`, `JSONUnmarshalOptions(opts ...json.Options)` | The same for one direction |
| `Debug(logger DebugLogger)` | Logs full request and response dumps. `Debug(nil)` uses golog; any logger with `Debugf(string, ...any)` fits, `app.Logger()` included |

`NewRateLimiter(requestsPerSecond int) Limiter` and `NewRateLimiterPerMinute(requestsPerMinute int) Limiter` build a limiter to hand to `RateLimiter` or `RateLimiterFor`. A `Limiter` is any type with `Wait(ctx context.Context) error`.

Exported fields on `Client`: `HTTPClient *http.Client`, `BaseURL string`, `PersistentRequestOptions []RequestOption`. You can tweak `HTTPClient` directly after `New`.

### Clone

`Clone` derives a new client that keeps the original options and appends more. Pass `client.NoOption` first to discard the inherited ones; the Iris JSON policy survives it:

```go
adminClient := c.Clone(client.PersistentRequestOptions(
    client.RequestAuthorizationBearer(adminToken),
))
```

A clone builds its own limiters for `RateLimit` and `RateLimitFor`, and shares the instance you passed to `RateLimiter` or `RateLimiterFor`.

## Request Options

`RequestOption = func(*http.Request) error`. They run after the request is constructed, before it is sent. Pass them per call (the trailing variadic on every request method) or client-wide via `PersistentRequestOptions`.

| Builder | Description |
|---------|-------------|
| `RequestHeader(overridePrev bool, key string, values ...string)` | Add (false) or replace (true) a header |
| `RequestAuthorization(value string)` | Set the `Authorization` header |
| `RequestAuthorizationBearer(accessToken string)` | Set `Authorization: Bearer <token>` |
| `RequestQuery(query url.Values)` | Merge URL query parameters |
| `RequestParam(key string, values ...string)` | Set a single query parameter |
| `RequestRateLimit(key string)` | Wait on the named limiter registered with `RateLimitFor`, on every attempt. An unknown key is a no-op |
| `ClientTrace(t *httptrace.ClientTrace)` | Attach an `httptrace.ClientTrace` to the request |

```go
resp, err := c.Do(ctx, http.MethodGet, "/search", nil,
    client.RequestParam("q", "iris"),
    client.RequestHeader(true, "X-Request-ID", reqID),
    client.RequestRateLimit("search"),
)
```

## Sending Requests

There are no per-verb shorthand methods (no `c.Get`, `c.Post`, etc.). The HTTP verb is always the `method` argument: pass `http.MethodGet`, `http.MethodPost`, `http.MethodPut`, `http.MethodDelete` and so on, or `client.MethodQuery` (`"QUERY"`, RFC 10008) for a read that carries a body. An empty method defaults to GET.

### Do

```go
func (c *Client) Do(ctx context.Context, method, urlpath string, payload any, opts ...RequestOption) (*http.Response, error)
```

The payload is encoded based on its type:

| Payload type | Body |
|--------------|------|
| `io.Reader` | Streamed as-is (sent once, never replayed by a retry) |
| `[]byte`, `json.RawMessage` | Raw bytes |
| `string` | Raw text |
| `url.Values` | URL-encoded string |
| any other value (struct, map, slice) | JSON-encoded |

`Do` does not turn non-2xx responses into errors; check `resp.StatusCode` yourself (the typed readers below do this for you). The caller owns the response body:

```go
resp, err := c.Do(ctx, http.MethodGet, "/todos/42", nil)
if err != nil {
    return err
}
defer c.DrainResponseBody(resp) // drain + close, lets the transport reuse the connection

if resp.StatusCode >= http.StatusBadRequest {
    return client.ExtractError(resp) // wraps status + body in an APIError
}
```

### JSON and Form

`JSON` and `Form` are `Do` with the right `Content-Type` set:

```go
resp, err := c.JSON(ctx, http.MethodPost, "/users", User{Name: "John"})

resp, err = c.Form(ctx, http.MethodPost, "/login", url.Values{
    "username": {"john"},
    "password": {"secret"},
})
```

### Multipart File Uploads

```go
uploader := c.NewUploader()
uploader.AddField("description", "monthly report")  // plain form field
uploader.AddFile("file", "./report.pdf")            // local file by path
uploader.AddFileSource("data", "data.csv", reader)  // any io.Reader

resp, err := uploader.Upload(ctx, http.MethodPost, "/upload")
if err != nil {
    return err
}
defer c.DrainResponseBody(resp)
```

An `Uploader` sends once. A second `Upload` returns `client.ErrUploaderClosed`.

## Reading Responses

The typed readers close the body for you and return an `APIError` when the status code is 400 or above. They come in two shapes, matching `iris.Context`: `Bind` returns a new value, `Read` fills a pointer you hold.

### BindJSON and ReadJSON

`BindJSON` is a generic method (Go 1.27): the type parameter is the response type, the arguments are the request method, URL path, an optional payload (sent as JSON when non-nil) and any request options:

```go
todos, err := c.BindJSON[[]Todo](ctx, http.MethodGet, "/todos", nil)
```

`ReadJSON` fills a value you already hold, which is what you want for a long-lived field or a pooled struct. The destination must be a non-nil pointer:

```go
var todos []Todo
err := c.ReadJSON(ctx, &todos, http.MethodGet, "/todos", nil)
```

An empty response body comes back as `io.EOF`, which `client.IsErrEmptyJSON(err)` also recognises. Several APIs answer a successful write with no content, so that is a normal outcome rather than a failure.

Encoding and decoding use the framework's shared JSON policy (`core/pkg/jsonutil`), the same one `ctx.JSON` and `ctx.ReadJSON` use on the server side. Three consequences worth knowing: a `time.Duration` field travels as int64 nanoseconds in both directions, map keys are sorted on encode, and member matching on decode is case-insensitive while duplicate members are rejected. `Bind`, `BindError` and `DecodeError` follow the same policy. `JSONOptions`, `JSONMarshalOptions` and `JSONUnmarshalOptions` replace it.

### BindPlain and ReadPlain

Read the body as plain text, typed as `string`, `[]byte`, `int`, `int64` or `float64` (named types over those work too):

```go
version, err := c.BindPlain[string](ctx, http.MethodGet, "/version", nil)

var count int
err = c.ReadPlain(ctx, &count, http.MethodGet, "/count", nil)
```

`GetPlainUnquote(ctx, method, urlpath, payload, opts...)` returns the body as a string and strips JSON quotes if the server mistakenly quoted a plain value.

### Call

`Call(ctx, method, urlpath, payload, opts...) error` sends a request whose success body carries nothing: it returns the `APIError` on a status of 400 or above, nil otherwise, and drains the body:

```go
if err := c.Call(ctx, http.MethodDelete, "/todos/42", nil); err != nil {
    return err
}
```

### WriteTo

Copies the response body to any `io.Writer`. If the writer is an `http.ResponseWriter`, it forwards `Content-Type` and `Content-Length`, which makes it a one-liner proxy inside an Iris handler:

```go
func proxyReport(ctx iris.Context) {
    _, err := c.WriteTo(ctx, ctx.ResponseWriter(), http.MethodGet, "/reports/latest", nil)
    if err != nil {
        errors.BadGateway.Err(ctx, err)
    }
}
```

### Bind

Package-level helpers for when you already hold a `*http.Response` (for example after `Do`). They bind based on the response `Content-Type` (`application/json`, or plain text into a string or byte slice) and fail fast on anything else. Closing the body is up to the caller:

```go
resp, err := c.Do(ctx, http.MethodGet, "/users/42", nil)
if err != nil {
    return err
}
defer c.DrainResponseBody(resp)

u, err := client.Bind[User](resp)
```

There is no pointer form: `client.BindResponse` was removed. Use `client.Bind[T]`.

## Paginated Lists

An Iris list endpoint answers a `Page[T]` envelope (see `rest-helpers.md`). The client decodes it into the same type the server built: `client.PageOptions`, `client.Page[T]`, `client.CountResponse[N]` and `client.ListFilter` alias the std-only `core/pkg/pagination` types, as `iris.Page[T]` does on the server.

| Function / option | Description |
|-------------------|-------------|
| `List[T](ctx, c *Client, urlpath string, page PageOptions, filter any, opts ...RequestOption) (*Page[T], error)` | One page. Sends QUERY with `page`, `size` and `details` in the URL query and `filter` as the JSON body |
| `Items[T](ctx, c *Client, urlpath string, filter any, opts ...RequestOption) iter.Seq2[T, error]` | Walks every page, item by item, until `HasNextPage` is false, a page is empty, or the loop breaks. A failed request yields its error once, with the zero `T`, and ends the walk |
| `PageSize(n int) RequestOption` | Sets the `size` URL query parameter. The server still clamps it to its own maximum |
| `ListMethod(method string) RequestOption` | Replaces the request method, e.g. `ListMethod(http.MethodPost)` for a server that lists over POST |
| `MethodQuery` | The `"QUERY"` method constant |

```go
type BookFilter struct {
    client.ListFilter
    Author string `json:"author,omitzero"`
}

page, err := client.List[Book](ctx, c, "/books",
    client.PageOptions{Page: 1, Size: 20}, BookFilter{Author: "Kurt Vonnegut"})

for book, err := range client.Items[Book](ctx, c, "/books", BookFilter{}, client.PageSize(50)) {
    if err != nil {
        return err
    }
    fmt.Println(book.Title)
}
```

Options passed to `List` and `Items` run after the page options, so they win. The Iris list endpoints generated by `controller/mvc` and the identity server's `/u/list` answer both POST and QUERY, so either method works against them.

## Retries

Retrying is opt-in. Pass a `RetryPolicy` and failed attempts are repeated with exponential backoff; a `Retry-After` response header wins when present. By default only network errors on idempotent methods and the 429, 502, 503 and 504 statuses are retried, and buffered request bodies (`[]byte`, `string`, `url.Values`, JSON-encoded values) are replayed:

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

`RetryPolicy` fields: `MaxAttempts`, `InitialBackoff` (default 500ms), `MaxBackoff` (default 10s), `Multiplier` (default 2), `Jitter`, `StatusCodes`, `RetryOn func(req, resp, err) bool` (replaces the built-in decision), `MaxRetryAfter`, `RetryNonIdempotent`, `OnRetry func(attempt int, req, resp, err, wait)`. Every attempt waits on the rate limiters and is visible to the request handlers, so `Debug` output shows the failed attempts as well.

## Redacting Secrets

When an API key travels in the query string it would otherwise be printed by `APIError.Error()` and by the `Debug` dumps. Register the parameter names once and their values are replaced with `REDACTED` in all rendered text; the request itself is not modified:

```go
c := client.New(
    client.BaseURL("https://api.example.com"),
    client.PersistentRequestOptions(client.RequestParam("apiKey", key)),
    client.RedactQueryParams("apiKey"),
    client.Debug(nil),
)
```

`client.RedactURL(u, "apiKey")` renders a URL the same way for your own log lines. Request and response bodies are never scrubbed.

## Error Handling

The typed readers return a `client.APIError` for responses with status 400 or above:

```go
type APIError struct {
    Response *http.Response
    Body     json.RawMessage // response body, already read and closed
    URL      string          // the request URL with redacted query values
    ReadErr  error           // set when the body could not be read in full
}
```

Helpers:

| Function | Description |
|----------|-------------|
| `(APIError).Code() int` | The status code, 0 when there is no response |
| `ExtractError(resp *http.Response) APIError` | Read the body and wrap the response in an `APIError` |
| `GetError(err error) (APIError, bool)` | Find an `APIError` in `err`, wrapped or not (`errors.As`). False for one without a response |
| `GetErrorCode(err error) int` | Status code of an `APIError`. Returns 0 for nil, non-API errors and errors without a response |
| `BindError[T](err error) (T, error)` | Decode the `APIError` body into a new `T`. Hands back `err` itself when it is not an `APIError` |
| `DecodeError(err error, destPtr any) error` | The pointer form of `BindError` |

```go
result, err := c.BindJSON[Order](ctx, http.MethodGet, "/orders/42", nil)
if err != nil {
    if client.GetErrorCode(err) == http.StatusNotFound {
        return ErrOrderNotFound
    }

    body, decErr := client.BindError[struct {
        Message string `json:"message"`
    }](err)
    if decErr == nil {
        return fmt.Errorf("orders api: %s", body.Message)
    }
    return err
}
```

When the remote server is an Iris application using `rest/errors`, the body is its wire envelope, so decode straight into `errors.Error` with `client.BindError[errors.Error](err)`. Inside an Iris handler, `errors.HandleAPIError(ctx, err)` forwards the remote envelope to your own client. `client.APIError` is the same type as `httpclient.APIError`, so both also work for errors produced by code that uses `github.com/kataras/httpclient` directly.

## Request Handlers (Hooks)

A `RequestHandler` runs around every request. `BeginRequest` fires after the request is constructed and right before it is sent; `EndRequest` fires right after the response arrives. A non-nil error from either aborts the call; an `EndRequest` handler that returns the error it was given passes it through:

```go
type RequestHandler interface {
    BeginRequest(context.Context, *http.Request) error
    EndRequest(context.Context, *http.Response, error) error
}
```

Register per client with `c.RegisterRequestHandler(h)` right after `New`, or for all future clients with the package-level `client.RegisterRequestHandler(h)`. The `Debug` option is implemented as such a handler.

## End-to-End Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/kataras/iris/v14/client"
)

type Todo struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
}

type apiErrorBody struct {
    Message string `json:"message"`
}

func main() {
    c := client.New(
        client.BaseURL("https://api.example.com"),
        client.Timeout(15*time.Second),
        client.RateLimit(10),
        client.Retry(client.RetryPolicy{MaxAttempts: 3}),
        client.PersistentRequestOptions(
            client.RequestAuthorizationBearer("YOUR_TOKEN"),
        ),
    )

    ctx := context.Background()

    // GET /todos?completed=false and bind the JSON array.
    todos, err := c.BindJSON[[]Todo](ctx, http.MethodGet, "/todos", nil,
        client.RequestParam("completed", "false"))
    if err != nil {
        if body, decErr := client.BindError[apiErrorBody](err); decErr == nil {
            fmt.Printf("list failed (%d): %s\n", client.GetErrorCode(err), body.Message)
        } else {
            fmt.Println("list failed:", err)
        }
        return
    }
    fmt.Printf("%d open todos\n", len(todos))

    // POST /todos with a JSON body and bind the created entity.
    created, err := c.BindJSON[Todo](ctx, http.MethodPost, "/todos",
        Todo{Title: "write the client docs"})
    if err != nil {
        fmt.Println("create failed:", err)
        return
    }
    fmt.Println("created todo", created.ID)

    // DELETE /todos/{id}: no response body to bind, use Call.
    if err := c.Call(ctx, http.MethodDelete, fmt.Sprintf("/todos/%d", created.ID), nil); err != nil {
        fmt.Println("delete failed:", err)
    }
}
```

## Source Files

- `client/client.go` - `Client`, `New`, `NoOption` and the type aliases over `github.com/kataras/httpclient`
- `client/option.go` - Client options, `Debug` with its golog default
- `client/request_option.go` - Request options
- `client/bind.go` - `Bind`, `BindError`, `DecodeError` under the Iris JSON policy
- `client/error.go` - `ExtractError`, `GetError`, `GetErrorCode`
- `client/list.go` - `MethodQuery`, `List`, `Items`, `PageSize`, `ListMethod` and the `PageOptions`, `Page`, `CountResponse`, `ListFilter` aliases

The methods (`Do`, `JSON`, `Form`, `Call`, `BindJSON`, `ReadJSON`, `BindPlain`, `ReadPlain`, `WriteTo`, `NewUploader`, `Clone`) and the retry, rate limit and redaction machinery live in `github.com/kataras/httpclient`.
