# Testing Guide

## httptest Package

The `httptest` package provides end-to-end testing utilities for Iris applications, built on top of the httpexpect library.

Import:

```go
import "github.com/kataras/iris/v14/httptest"
```

## Basic Testing

### Testing an Application

```go
func TestMyAPI(t *testing.T) {
    // Build the app (same as production setup, without Listen)
    app := iris.NewBuilder().
        NoPrefix().
        AllowOrigin("*").
        Compression(false).
        LogRequests(false).
        NoHealth().
        Services(NewUserService).
        RouterMiddlewares().
        Middlewares().
        API("/users", NewUserAPI).
        Build()

    // Create test client
    e := httptest.New(t, app)

    // Make requests and assert responses
    e.GET("/users").Expect().
        Status(httptest.StatusOK).
        JSON().Array().NotEmpty()
}
```

`httptest.New` calls `app.Build()` internally and reports build failures through `t.Fatalf` (it integrates with the testing framework instead of panicking).

### Testing with iris.New()

```go
func TestSimpleApp(t *testing.T) {
    app := iris.New()
    app.Get("/hello", func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "hello"})
    })

    e := httptest.New(t, app)
    e.GET("/hello").Expect().
        Status(httptest.StatusOK).
        JSON().Object().
        HasValue("message", "hello")
}
```

## httptest.New Options

```go
// Default
e := httptest.New(t, app)

// With custom URL
e := httptest.New(t, app, httptest.URL("https://myapp.com"))

// With debug output
e := httptest.New(t, app, httptest.Debug(true))

// With custom log level
e := httptest.New(t, app, httptest.LogLevel("debug"))

// With strict mode (fail on the first error instead of collecting all)
e := httptest.New(t, app, httptest.Strict(true))
```

## HTTP Method Testing

### GET Requests

```go
e.GET("/users").Expect().
    Status(httptest.StatusOK).
    JSON().Array().Length().IsEqual(3)

e.GET("/users/{id}", "123").Expect().
    Status(httptest.StatusOK).
    JSON().Object().
    HasValue("id", "123").
    HasValue("name", "John")

// With query parameters
e.GET("/users").WithQuery("page", 2).WithQuery("per_page", 10).
    Expect().Status(httptest.StatusOK)

// With headers
e.GET("/protected").WithHeader("Authorization", "Bearer token123").
    Expect().Status(httptest.StatusOK)
```

### POST Requests

```go
user := map[string]any{
    "name":  "John",
    "email": "john@example.com",
}

e.POST("/users").WithJSON(user).Expect().
    Status(httptest.StatusCreated).
    JSON().Object().
    HasValue("name", "John").
    ContainsKey("id")
```

### PUT Requests

```go
update := map[string]any{
    "name": "John Updated",
}

e.PUT("/users/{id}", "123").WithJSON(update).Expect().
    Status(httptest.StatusOK)
```

### DELETE Requests

```go
e.DELETE("/users/{id}", "123").Expect().
    Status(httptest.StatusNoContent)
```

### PATCH Requests

```go
patch := map[string]any{"name": "New Name"}
e.PATCH("/users/{id}", "123").WithJSON(patch).Expect().
    Status(httptest.StatusOK)
```

## Response Assertions

### Status Codes

```go
e.GET("/").Expect().Status(httptest.StatusOK)           // 200
e.POST("/").Expect().Status(httptest.StatusCreated)      // 201
e.DELETE("/").Expect().Status(httptest.StatusNoContent)   // 204
e.GET("/missing").Expect().Status(httptest.StatusNotFound) // 404
```

### JSON Assertions

```go
resp := e.GET("/users").Expect().Status(httptest.StatusOK)

// Array assertions
resp.JSON().Array().
    NotEmpty().
    Length().IsEqual(3)

// Object assertions
resp.JSON().Object().
    HasValue("name", "John").
    HasValue("age", 30).
    ContainsKey("id").
    NotContainsKey("password")

// Nested object
resp.JSON().Object().
    Value("address").Object().
    HasValue("city", "Athens")

// Value checks
resp.JSON().Object().
    Value("count").Number().IsEqual(42)

resp.JSON().Object().
    Value("active").Boolean().IsTrue()
```

### Header Assertions

```go
e.GET("/").Expect().
    Header("Content-Type").IsEqual("application/json; charset=utf-8").
    Header("X-Request-Id").NotEmpty()
```

### Body Assertions

```go
e.GET("/text").Expect().
    Body().IsEqual("Hello, World!")

e.GET("/html").Expect().
    Body().Contains("<h1>")
```

## Testing Individual Handlers

Test a handler without building a full application:

```go
func TestHandler(t *testing.T) {
    handler := func(ctx iris.Context) {
        ctx.JSON(iris.Map{"status": "ok"})
    }

    w := httptest.NewRecorder()
    r := httptest.NewRequest("GET", "/", nil)
    httptest.Do(w, r, handler)

    // Assert on w (http.ResponseRecorder)
}
```

## Testing with Authentication

### Basic Auth

```go
e.GET("/admin").WithBasicAuth("admin", "password").
    Expect().Status(httptest.StatusOK)

e.GET("/admin").WithBasicAuth("wrong", "creds").
    Expect().Status(httptest.StatusUnauthorized)
```

### Bearer Token

```go
e.GET("/protected").
    WithHeader("Authorization", "Bearer valid-token").
    Expect().Status(httptest.StatusOK)
```

## Testing Form Data

```go
e.POST("/form").
    WithFormField("name", "John").
    WithFormField("email", "john@example.com").
    Expect().Status(httptest.StatusOK)
```

## Testing File Uploads

```go
e.POST("/upload").
    WithMultipart().
    WithFile("file", "testfile.txt").
    Expect().Status(httptest.StatusOK)
```

## Testing with a Real Network Listener

`httptest.New` routes requests in-memory (no network). When a test needs a real listening server (for example, testing an HTTP client against a live endpoint), use `httptest.NewServer`, which builds the app and wraps it in a standard `net/http/httptest.Server`:

```go
func TestWithRealServer(t *testing.T) {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.Text("ok")
    })

    srv := httptest.NewServer(t, app) // *net/http/httptest.Server
    defer srv.Close()

    resp, err := http.Get(srv.URL + "/")
    // ...
}
```

`NewServer` calls `app.Build()` and reports failures through `t.Fatal`.

## Testing TLS Servers

```go
// For testing running TLS servers with insecure skip verification
e := httptest.NewInsecure(t,
    httptest.URL("https://localhost:8443"),
)
```

## Testing Error Responses

```go
// Test canonical error responses
e.GET("/users/{id}", "nonexistent").Expect().
    Status(httptest.StatusNotFound).
    JSON().Object().
    Value("http_error_code").Object().
    HasValue("canonical_name", "NOT_FOUND").
    HasValue("status", 404)

// Test validation errors
e.POST("/users").WithJSON(map[string]any{"name": "x"}).Expect().
    Status(httptest.StatusBadRequest).
    JSON().Object().
    Value("validation").Array().NotEmpty()
```

## Testing Middleware

```go
func TestRateLimiting(t *testing.T) {
    app := iris.New()
    app.Use(rate.Limit(1, 1, rate.Options{})) // 1 req/sec, burst 1
    app.Get("/", func(ctx iris.Context) {
        ctx.Text("ok")
    })

    e := httptest.New(t, app)

    // First request succeeds
    e.GET("/").Expect().Status(httptest.StatusOK)

    // Second request is rate limited
    e.GET("/").Expect().Status(httptest.StatusTooManyRequests)
}
```

## Testing Controllers in Isolation

```go
func TestUserAPI(t *testing.T) {
    // Create mock service
    mockService := &MockUserService{
        users: []User{{ID: "1", Name: "Test"}},
    }

    // Create controller with mock
    api := NewUserAPI(mockService)

    // Build minimal app
    app := iris.New()
    app.Router("/users", api)

    e := httptest.New(t, app)
    e.GET("/users").Expect().
        Status(httptest.StatusOK).
        JSON().Array().Length().IsEqual(1)
}
```

`httptest.NewController` is that scaffold as one call, because the unit under test is a `Routing` value while `httptest.New` takes a built application:

```go
func NewController(t IrisTesty, prefix string, controller any,
    mw ...context.Handler) *httpexpect.Expect
```

```go
func TestUserAPI(t *testing.T) {
    e := httptest.NewController(t, "/users", NewUserAPI(&MockUserService{}))

    e.GET("/users").Expect().Status(httptest.StatusOK).
        JSON().Array().Length().IsEqual(1)
}
```

- The variadic middleware is registered on the controller's **own group**, so it runs for its routes and not for the 404 and 405 chains.
- A failure answers what an `iris.NewBuilder` application answers, so the test asserts the production body: the path parameter type and context error handlers are installed, `errors.Handler` runs before the router (a bare failure status such as a middleware's 403 renders the `rest/errors` envelope, not an empty body), and `errors.Handler` is the 404 handler (an unknown path under the prefix renders the `NOT_FOUND` envelope). A method the controller does not serve answers that 404, not a 405, because `Options.Router.FireMethodNotAllowed` is off by default and `NewController` takes no options.
- Nothing else the builder chain adds is there: no CORS, no access log, no recovery, no compression.
- It takes an `IrisTesty` (which `*testing.T` satisfies) rather than a `*testing.T`, for the same reason `New` does. It takes no `OptionSetter`: Go allows one variadic parameter and the middleware is the one a controller test needs. A test that needs `httptest.URL` or a log level builds the application itself and calls `New`.

## Status Code Constants

The `httptest` package re-exports all standard HTTP status codes:

```go
httptest.StatusOK                    // 200
httptest.StatusCreated               // 201
httptest.StatusNoContent             // 204
httptest.StatusBadRequest            // 400
httptest.StatusUnauthorized          // 401
httptest.StatusForbidden             // 403
httptest.StatusNotFound              // 404
httptest.StatusConflict              // 409
httptest.StatusTooManyRequests       // 429
httptest.StatusInternalServerError   // 500
```

## Network Testing Utilities

```go
// Create local TCP listener for testing
listener := httptest.NewLocalListener()

// Create TLS listener with self-signed certificate.
// Takes a net.Listener; pass nil to create one via NewLocalListener.
tlsListener := httptest.NewLocalTLSListener(nil)

// Or wrap an existing listener
tlsListener := httptest.NewLocalTLSListener(httptest.NewLocalListener())

// Generated-on-first-use localhost certificate (ECDSA P-256, SANs for
// localhost/127.0.0.1/::1, valid one year from process start).
cert := httptest.LocalhostCert()
key := httptest.LocalhostKey()
```

## Source Files

- `httptest/` - Testing utilities, configuration, status codes
- `httptest/controller.go` - `NewController`
