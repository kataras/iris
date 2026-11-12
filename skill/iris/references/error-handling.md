# Error Handling

## Overview

The `rest/errors` package provides a structured, canonical error handling system for HTTP APIs. It follows a standardized JSON error response format with canonical error codes mapped to HTTP status codes.

Import:

```go
import "github.com/kataras/iris/v14/rest/errors"
```

## Error Response Format

All error responses follow this JSON structure:

```json
{
    "id": "unique-error-id",
    "http_error_code": {
        "canonical_name": "NOT_FOUND",
        "status": 404
    },
    "message": "short summary of the error",
    "detail": "detailed explanation specific to this occurrence",
    "data": { "additional": "context-specific data" },
    "validation": [
        {
            "field": "field_name",
            "value": "invalid_value",
            "reason": "explanation of why invalid"
        }
    ]
}
```

Fields:
- `id` - Unique error instance identifier (auto-generated)
- `http_error_code` - Canonical name and HTTP status code
- `message` - Short, human-readable summary
- `detail` (optional) - Specific explanation for this occurrence
- `data` (optional) - Additional context data
- `validation` (optional) - Array of field validation errors

## Canonical Error Codes

### Common Codes

| Code | HTTP Status | Constant | When to Use |
|------|-------------|----------|-------------|
| `INVALID_ARGUMENT` | 400 | `errors.InvalidArgument` | Bad client input, validation failures |
| `FAILED_PRECONDITION` | 400 | `errors.FailedPrecondition` | System not in required state |
| `OUT_OF_RANGE` | 400 | `errors.OutOfRange` | Value past valid range |
| `UNAUTHENTICATED` | 401 | `errors.Unauthenticated` | Missing/invalid credentials |
| `PAYMENT_REQUIRED` | 402 | `errors.PaymentRequired` | Payment needed |
| `PERMISSION_DENIED` | 403 | `errors.PermissionDenied` | Insufficient permissions |
| `NOT_FOUND` | 404 | `errors.NotFound` | Resource not found |
| `METHOD_NOT_ALLOWED` | 405 | `errors.MethodNotAllowed` | Unsupported HTTP method |
| `ALREADY_EXISTS` | 409 | `errors.AlreadyExists` | Duplicate resource |
| `ABORTED` | 409 | `errors.Aborted` | Concurrency conflict |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | `errors.UnsupportedMediaType` | Bad content type |
| `RESOURCE_EXHAUSTED` | 429 | `errors.ResourceExhausted` | Rate limited, quota exceeded |
| `TOO_LARGE` | 413 | `errors.TooLarge` | Request entity too large |
| `INTERNAL` | 500 | `errors.Internal` | Internal server error |
| `DATA_LOSS` | 500 | `errors.DataLoss` | Unrecoverable data loss |
| `UNKNOWN` | 500 | `errors.Unknown` | Unknown error |
| `UNIMPLEMENTED` | 501 | `errors.Unimplemented` | Not implemented |
| `BAD_GATEWAY` | 502 | `errors.BadGateway` | An upstream this service depends on failed |
| `UNAVAILABLE` | 503 | `errors.Unavailable` | Service unavailable |
| `DEADLINE_EXCEEDED` | 504 | `errors.DeadlineExceeded` | Timeout |

### Extended Codes

| Code | HTTP Status | Constant |
|------|-------------|----------|
| `NOT_ACCEPTABLE` | 406 | `errors.NotAcceptable` |
| `PROXY_AUTH_REQUIRED` | 407 | `errors.ProxyAuthRequired` |
| `REQUEST_TIMEOUT` | 408 | `errors.RequestTimeout` |
| `CANCELLED` | 408 | `errors.Cancelled` |
| `GONE` | 410 | `errors.Gone` |
| `LENGTH_REQUIRED` | 411 | `errors.LengthRequired` |
| `PRECONDITION_FAILED` | 412 | `errors.PreconditionFailed` |
| `REQUEST_URI_TOO_LONG` | 414 | `errors.RequestURITooLong` |
| `REQUESTED_RANGE_NOT_SATISFIABLE` | 416 | `errors.RequestedRangeNotSatisfiable` |
| `EXPECTATION_FAILED` | 417 | `errors.ExpectationFailed` |
| `IM_A_TEAPOT` | 418 | `errors.ImATeapot` |
| `MISDIRECTED_REQUEST` | 421 | `errors.MisdirectedRequest` |
| `UNPROCESSABLE_ENTITY` | 422 | `errors.UnprocessableEntity` |
| `LOCKED` | 423 | `errors.Locked` |
| `FAILED_DEPENDENCY` | 424 | `errors.FailedDependency` |
| `TOO_EARLY` | 425 | `errors.TooEarly` |
| `UPGRADE_REQUIRED` | 426 | `errors.UpgradeRequired` |
| `PRECONDITION_REQUIRED` | 428 | `errors.PreconditionRequired` |
| `REQUEST_HEADER_FIELDS_TOO_LARGE` | 431 | `errors.RequestHeaderFieldsTooLarge` |
| `UNAVAILABLE_FOR_LEGAL_REASONS` | 451 | `errors.UnavailableForLegalReasons` |
| `HTTP_VERSION_NOT_SUPPORTED` | 505 | `errors.HTTPVersionNotSupported` |
| `VARIANT_ALSO_NEGOTIATES` | 506 | `errors.VariantAlsoNegotiates` |
| `INSUFFICIENT_STORAGE` | 507 | `errors.InsufficientStorage` |
| `LOOP_DETECTED` | 508 | `errors.LoopDetected` |
| `NOT_EXTENDED` | 510 | `errors.NotExtended` |
| `NETWORK_AUTHENTICATION_REQUIRED` | 511 | `errors.NetworkAuthenticationRequired` |

### Iris-specific Codes

Non-standard status codes registered by the package (status constants live in `context/status.go`):

| Code | HTTP Status | Constant |
|------|-------------|----------|
| `PAGE_EXPIRED` | 419 | `errors.PageExpired` |
| `BLOCKED_BY_WINDOWS_PARENTAL_CONTROLS` | 450 | `errors.BlockedByWindowsParentalControls` |
| `INVALID_TOKEN` | 498 | `errors.InvalidToken` |
| `TOKEN_REQUIRED` | 499 | `errors.TokenRequired` |
| `BANDWIDTH_LIMIT_EXCEEDED` | 509 | `errors.BandwidthLimitExceeded` |
| `INVALID_SSL_CERTIFICATE` | 526 | `errors.InvalidSSLCertificate` |
| `SITE_OVERLOADED` | 529 | `errors.SiteOverloaded` |
| `SITE_FROZEN` | 530 | `errors.SiteFrozen` |
| `NETWORK_READ_TIMEOUT` | 598 | `errors.NetworkReadTimeout` |

## Error Methods

All methods are defined on `ErrorCodeName` (e.g. `errors.NotFound`). Each one stops request processing and sends the response immediately, except `Wrap`.

| Method | Signature | Purpose |
|--------|-----------|---------|
| `Message` | `(ctx, message string)` | Message only |
| `Messagef` | `(ctx, format string, args ...any)` | Printf-formatted message |
| `Detail` | `(ctx, msg, detail string)` | Message + detail |
| `Detailf` | `(ctx, msg, detail string, detailArgs ...any)` | Message + printf-formatted detail |
| `Data` | `(ctx, msg string, data any)` | Message + structured data |
| `DataWithDetail` | `(ctx, msg, detail string, data any)` | Message + detail + structured data |
| `Validation` | `(ctx, validationErrors ...ValidationError)` | Field-level validation failures |
| `Err` | `(ctx, err error)` | Sends err.Error() as the message |
| `ErrPrivate` | `(ctx, err error)` | Logs err server-side, sends a generic message |
| `LogErr` | `(ctx, err error)` | Logs err; `Internal` sends a generic message, other codes send the error text |
| `Problem` | `(ctx, title, detail, typeURI, instanceURI string, extra ...any)` | RFC 7807 problem details |
| `Wrap` | `(err error) error` | Returns a wrapped error for later processing, sends nothing |

### Detail / Detailf

```go
errors.NotFound.Detail(ctx, "user not found", "no user with id '123'")
errors.NotFound.Detailf(ctx, "user not found", "no user with id %q in org %q", userID, orgID)
```

Response:
```json
{
    "http_error_code": {"canonical_name": "NOT_FOUND", "status": 404},
    "message": "user not found",
    "detail": "no user with id '123'"
}
```

### Message / Messagef

```go
errors.Unauthenticated.Message(ctx, "invalid credentials")
errors.NotFound.Messagef(ctx, "user with ID %s not found", userID)
```

### Err

Send error based on an error value. The error's text becomes the client-visible message. Validation errors are auto-routed to `Validation()`, and errors wrapped with `iris.PrivateError` are hidden from the client:

```go
errors.Internal.Err(ctx, err)
```

### ErrPrivate

Log the full error server-side and send only a generic message to the client. Use for errors that may leak schema, paths, or credentials. It holds for every error type: a `*errors.Error` or a validation error, which `Err` renders as they are, gets the generic message here too:

```go
user, err := db.Query("SELECT * FROM users WHERE secret_token = $1", token)
if err != nil {
    errors.Internal.ErrPrivate(ctx, err) // logs full error, client sees "server error"
    return
}
```

### LogErr

Log the error and send an error response. For `errors.Internal` the client gets a generic message (same as `ErrPrivate`); for other codes the error text is sent:

```go
errors.Internal.LogErr(ctx, err)
```

### Report (no render)

`errors.Report(ctx, err)` runs the whole pipeline and returns the `*Error` **without writing anything to the response**. It exists for a response that has already committed a status code, which is every streaming response after its first byte: the render path would write a JSON body into the middle of an event stream, and skipping the pipeline instead means the logger and the collector never fire, so the failure reaches no sink at all.

```go
reported := errors.Report(ctx, err) // resolves through the route's error map
// ...or under a code you already know:
reported = errors.BadGateway.Report(ctx, err)
```

It resolves the error against the route's error-code map, honours `SkipCanceled`, calls the logger with the **unredacted** error, generates the correlation id, redacts a private or internal error down to `"server error"`, and calls the collector. It returns nil when there is nothing to report: a nil error, or a cancellation while `SkipCanceled` is on. The returned value is the client-safe copy.

The `sse` package renders its error frame with this, so a streaming handler gets the same treatment as any other without extra wiring, and an `InitPatrol`-style setup that replaces `errors.LogError` sees stream failures too. See [sse.md](./sse.md) and [ai.md](./ai.md).

### Data / DataWithDetail

```go
errors.FailedPrecondition.Data(ctx, "insufficient funds", iris.Map{
    "current_balance":  50.00,
    "required_balance": 100.00,
})

errors.InvalidArgument.DataWithDetail(ctx,
    "validation failed",
    "multiple fields contain invalid values",
    iris.Map{"invalid_fields": []string{"email", "phone"}},
)
```

Response:
```json
{
    "http_error_code": {"canonical_name": "FAILED_PRECONDITION", "status": 400},
    "message": "insufficient funds",
    "data": {"current_balance": 50.00, "required_balance": 100.00}
}
```

### Problem (RFC 7807)

Send an RFC 7807 "Problem Details" response, rendered as `application/problem+json` by default:

```go
errors.InvalidArgument.Problem(ctx,
    "Validation Failed",                                // title
    "The email field contains an invalid email address", // detail
    "https://example.com/problems/validation-failed",    // type URI
    "/users/create")                                     // instance URI
```

Response:
```json
{
    "type": "https://example.com/problems/validation-failed",
    "title": "Validation Failed",
    "status": 400,
    "detail": "The email field contains an invalid email address",
    "instance": "/users/create"
}
```

Optional trailing `extra ...any` values are included as data (a single value directly, multiple values as an array; nils are filtered).

## Validation Errors

`errors.ValidationError` is an interface, not a struct:

```go
type ValidationError interface {
    error
    GetReason() string
}
```

Any type implementing it (plus `errors.ValidationErrors`, a slice of them) can be passed to `Validation()` or `Err()`. The framework ships ready implementations in the validation subpackage:

```go
import "github.com/kataras/iris/v14/rest/errors/validation"
```

### Constructors and Rules

| Constructor | Type | Chainable rules |
|-------------|------|-----------------|
| `validation.Field(name, value)` | `*FieldError[T]`, generic | `Func(fns ...func(T) string)` |
| `validation.String(name, value)` | `*StringError` | `NotEmpty()`, `Fullname()`, `Length(min, max)`, `Email()` |
| `validation.Number(name, value)` | `*NumberError[T]` | `Positive()`, `Negative()`, `Zero()`, `NonZero()`, `InRange(min, max)` |
| `validation.Slice(name, value)` | `*SliceError[T, V]` | length/emptiness rules (`NotEmptySlice`, `SliceLength`) |

Rule functions take the value and return an empty string on success or a reason on failure. Package-level rules (`validation.NotEmpty`, `validation.Email`, `validation.StringLength(min, max)`, `validation.InRange(min, max)`, etc.) plug into `Func`, as do custom closures.

### Usage

```go
name := validation.String("name", input.Name).NotEmpty().Length(3, 50)
email := validation.String("email", input.Email).Email()
age := validation.Number("age", input.Age).InRange(18, 120)

// Join filters out fields that passed; returns nil when all are valid.
if err := validation.Join(name, email, age); err != nil {
    errors.InvalidArgument.Err(ctx, err) // auto-routed to Validation()
    return
}
```

Generic fields with custom rules:

```go
username := validation.Field("username", input.Username).Func(
    validation.NotEmpty,
    func(value string) string {
        if strings.Contains(value, " ") {
            return "must not contain spaces"
        }
        return ""
    },
)

if err := username.Get(); err != nil { // Get() returns nil when valid
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

Or pass field errors to `Validation` directly:

```go
errors.InvalidArgument.Validation(ctx, name, email)
```

Response:
```json
{
    "http_error_code": {"canonical_name": "INVALID_ARGUMENT", "status": 400},
    "message": "validation failure",
    "detail": "fields were invalid",
    "validation": [
        {"field": "name", "value": "jo", "reason": "must be between 3 and 50 characters"},
        {"field": "email", "value": "bad", "reason": "must be a valid email address"}
    ]
}
```

## Error Mapping

Map Go errors to canonical HTTP error codes for automatic handling.

### Default Map

`errors.NewOptions()` starts with a default error code map (not empty), built by `makeDefaultErrorCodeMap` in `rest/errors/options.go`:

- `database/sql.ErrNoRows` maps to `NotFound`
- `github.com/kataras/jwt` errors (`jwt.ErrExpired`, `jwt.ErrMissing`, `jwt.ErrTokenSignature`, `jwt.ErrBlocked`, etc.) map to `Unauthenticated`
- `view.ErrNoEngine` maps to `Internal`
- hex decode errors ("encoding/hex: invalid byte") map to `InvalidArgument`

PostgreSQL (`github.com/kataras/pg`) mappings are opt-in: a blank import of
`rest/errors/pgerrors` restores `pg.ErrNoRows` -> `NotFound`, duplicate-key
-> `AlreadyExists` (generic client detail, raw error logged), foreign-key
violations (SQLSTATE `23503`) -> `InvalidArgument` ("invalid reference",
naming the constraint when the driver reports one; without it they fell through to
a 500) and SQLSTATE text trimming to `InvalidArgument`. The subpackage uses
`errors.RegisterErrorResolver`, the extension point for driver-specific
mapping that runs after the per-route code map.

`MapErrors` / `MapErrorFunc` / `MapErrorTypes` calls extend this map; they do
not replace it. `MapErrorFuncCtx` registers a context-aware mapper
(`func(ctx *context.Context, err error) error`) so the transformed error can
carry request detail; `ErrorCodeMap.Filter(ctx, err)` takes the context as
its first argument (nil for direct calls outside a request). For call-site
enrichment inside handlers use `errors.HandleErrorf(ctx, err, "plan %s", id)`
which wraps with `%w` so mappings still apply.

### MapErrorAs (typed) and MapErrorContains (text)

Two shapes of `MapErrorFunc` closure have their own call.

```go
// Finds *QuotaError anywhere in the error tree (errors.AsType) and builds the
// response from its fields. A nil result means "not mine": the next mapping
// gets a turn, like MapErrorFunc returning nil.
opts.MapErrorAs(errors.ResourceExhausted,
    func(e *QuotaError) *errors.Error {
        return &errors.Error{
            Message: "quota exceeded",
            Detail:  fmt.Sprintf("the limit is %d", e.Limit),
        }
    })

// Classify without transforming: name E, there is nothing left to infer from.
opts.MapErrorAs[*QuotaError](errors.ResourceExhausted, nil)
```

`MapErrorAs` is a **generic method** on both mappers, and it returns the receiver so the call keeps chaining after `MapErrors` and the rest. `E` is inferred from `render`:

```go
func (opts *Options) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) *Options
func (m ErrorCodeMap) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) ErrorCodeMap
```

For the same lookup in your own code, `errors.AsType[E](err) (E, bool)` is the standard library's `errors.AsType`, forwarded by `rest/errors` next to `Is`, `As`, `Unwrap` and `Join`: `if quota, ok := errors.AsType[*QuotaError](err); ok { ... }`.

`MapErrorContains` is a real method pair on both types, for the SQL and vendor errors that arrive as text only:

```go
opts.MapErrorContains(errors.InvalidArgument,
    "violates check constraint", "invalid value")
```

The match is case-sensitive and `message` **replaces** the original text, so the driver's words never reach the client. Prefer a typed mapping (`MapErrorAs`, `MapErrorTypes`, `MapErrors`) wherever one exists: a driver is free to reword its text in the next release, and a substring mapping fails silently when it does. Log before mapping when those words are the only diagnostic, the way `pgerrors` does with `errors.LogError`. An empty `substr` registers nothing at all rather than a mapping that matches every error; an empty `message` keeps the original text.

### Error Resolvers (driver-specific mapping)

`RegisterErrorResolver` is the extension point for classifying errors from a driver you do not want linked into every binary. It is how `rest/errors/pgerrors` works, and the same pattern fits MongoDB, Redis or any other client.

```go
type ErrorResolver func(ctx *context.Context, err error) bool

func RegisterErrorResolver(fn ErrorResolver)
```

The ordering contract:

1. The per-route `Options` error code map runs **first**.
2. Registered resolvers run next, **in registration order**, only for errors the code map did not claim.
3. Built-in classification runs **last**.

A resolver returns `true` when it produced a client response, which **stops the chain**. Register at package `init` time or before serving; a nil function is ignored.

```go
func init() {
    errors.RegisterErrorResolver(func(ctx iris.Context, err error) bool {
        if errors.Is(err, mongo.ErrNoDocuments) {
            errors.NotFound.Message(ctx, "resource not found")
            return true
        }
        return false
    })
}
```

Keep driver text out of the client response. `pgerrors` models this: on a duplicate key it logs the raw pgx error server-side and sends only a generic "a resource with the given value already exists" detail.

### In Builder

```go
iris.NewBuilder().
    // ...
    Errors(errors.NewOptions().
        // Map specific errors to codes
        MapErrors(errors.NotFound, sql.ErrNoRows, ErrUserNotFound).
        MapErrors(errors.InvalidArgument, ErrValidation, ErrBadInput).
        MapErrors(errors.AlreadyExists, ErrDuplicateEmail).

        // Map with custom function
        MapErrorFunc(errors.Internal, func(err error) error {
            // Custom error processing
            log.Printf("Internal error: %v", err)
            return err
        }),
    ).
    // ...
```

### How Mapping Works

When an error is returned from a handler or passed to `Err()`, the error options middleware checks if the error matches any mapped error. If so, it automatically sends the appropriate canonical error response.

```go
func (api *UserAPI) getUser(ctx iris.Context) {
    user, err := api.service.GetUser(ctx, id)
    if err != nil {
        // If err == sql.ErrNoRows, automatically sends 404 NOT_FOUND
        // Otherwise sends 500 INTERNAL
        errors.Internal.Err(ctx, err)
        return
    }
    ctx.JSON(user)
}
```

### Handle and ctx.HandleError: the guard forms

`ctx.HandleError(err) bool` is `errors.HandleError(ctx, err)` as a context method: the error half that `OK`, `Created`, `NoContent`, `List`, `Count` and `Paginated` each run before rendering their success body, on its own for a step that has an error but nothing to render on success. A nil error writes nothing and reports false. A handler whose only use of `rest/errors` was that call drops the import.

`errors.<Code>.Handle(ctx, err) bool` is the code-forcing counterpart: it renders the error as that code, whatever the central map would have made of it, and reports whether it did. Use it for a step whose failure means one thing at that call site and nothing in general.

```go
if ctx.HandleError(api.identity.ChangeEmail(ctx, input)) {
    return
}

if errors.Unavailable.Handle(ctx, api.users.UpdateEmail(ctx, old, new)) {
    return
}
```

`Handle` is `Err` plus the answer to "did it fire". `Err`, `ErrPrivate` and `LogErr` keep their `func(*context.Context, error)` signature deliberately: that is what `LogErrorFunc` and `cors.HandleErrorFunc` take, so `cors.New().HandleErrorFunc(errors.FailedPrecondition.Err)` passes one as a bare method value. Reach for `LogErr` rather than `Handle` when the failure is a server fault worth a log line; `Handle` renders without logging, like `Err`.

### NotFoundID

`errors.NotFoundID(ctx, resource string, id any) bool` renders `NOT_FOUND` with the detail `"<resource> <id> not found"`. The id is formatted with `%v`, so a string, an integer and a `uuid.UUID` all read the way they print. Its result follows the `Context.OK` convention (true means the handler succeeded) so it is always false, the opposite of `HandleError`.

```go
plan, err := svc.GetPlan(ctx, id)
if errors.HandleError(ctx, err) {
    return
}

if plan == nil {
    errors.NotFoundID(ctx, "plan", id)
    return
}

ctx.OK(plan, nil)
```

## Private Errors

Two complementary mechanisms hide sensitive error details from clients:

- **`errors.<Code>.ErrPrivate(ctx, err)`** (rest/errors): logs the full error server-side and sends a generic message under the given code. Choose this at the response site.
- **`errors.Private(err)`** (rest/errors): wraps an error so it is never shown to the client, no matter where it surfaces later in the rest/errors pipeline. It delegates to `context.PrivateErrorf`, which appends the `context.ErrPrivate` marker to the wrap chain while keeping the original message for server-side logs. `Err()` detects the marker by unwrapping the error chain. Choose this at the error-creation site (e.g. in a service layer that does not know how the error will be rendered).

```go
// Service layer: mark the error as private once.
return errors.Private(fmt.Errorf("query failed on host %s: %w", host, err))

// Handler: errors.Internal.Err(ctx, err) will log it and send a generic message.
```

**A private error is never rendered, on any path.** Private means stored with `iris.PrivateError` or `ctx.SetErrPrivate`, or carrying the mark anywhere in its chain (`errors.Private`, or a private error wrapped again with `%w`). `errors.Handler` answers such an error before any mapping, resolver or wrapped `ErrorCodeName` is consulted, keeping the status the handler set: below 500 the status's canonical code with the status text as the message and no detail; at 500 and above the generic `"server error"`, logged. A private `*errors.Error` is not sent as it is, and a private validation error does not render its fields. The private text still reaches the collector.

`ctx.StopWithError` reads the chain the same way: with a private error (`iris.PrivateError(err)`, `errors.Private(err)`, or a rewrap) it sets the status, writes nothing and stores the error as private (`ctx.ErrPublic()` reports it so; `ctx.GetErr()` returns what you passed). With a non-private error it writes the text itself, so `errors.Handler` sees a body and leaves that response alone.

The `jwt` middleware stops every bad token with `ctx.StopWithError(401, iris.PrivateError(err))`, so its 401 reads `"message":"Unauthorized"` with or without `Errors(errors.NewOptions())`: a client cannot read from the body why its token was refused. The `auth` package's own error handler is a different path.

## Default Handlers

The builder automatically registers these handlers:

- **`errors.DefaultPathParameterTypeErrorHandler`** - Handles macro parameter validation errors
- **`errors.DefaultContextErrorHandler`** - Handles context-level errors
- **`errors.DefaultRateLimitExceededHandler`** - Handles rate limit exceeded responses
- **`errors.Handler`** - With `UseRouter`, as the last of the builder's built-in router middleware (after compression), and as the 404 and 405 handler when the application registered none of its own at `Build`

### errors.Handler: bare failure statuses, 404 and 405

After the chain returns, `errors.Handler` renders a status of 400 or above that has nothing written, whether or not the chain was stopped (`ctx.StopWithStatus(403)` and `ctx.NotFound()` are alike). What the client reads:

- no stored error: the status text is the message (`"message":"Forbidden"`);
- a private stored error, of any type: answered first, before any mapping; the status the handler set, with the status text below 500 and the generic `"server error"` (logged) at 500 and above;
- a non-private stored error a mapping or resolver recognises: that mapping. A JSON decode error stored next to a status below 500 (`ctx.StopWithPlainError(400, err)` with the error of `ctx.ReadJSON`) answers 400 `INVALID_ARGUMENT`, `"unable to parse request body"`; `bodylimit`'s error answers 413 `TOO_LARGE`, `"request body too large"`, for both the declared-length and the streamed case;
- a non-private stored `*errors.Error` at 500 and above: sent as it is, because that type exists to be sent to clients (logged; the collector is not called for it);
- any other stored error at 500 and above: logged, and the client reads `"server error"`; the status the handler set stays (a 503 is a 503);
- any other stored error below 500: its text is the message.

It writes nothing when a body exists, when the response declares a `Content-Length` (a standard handler mounted with `iris.FromStd` that sets `Content-Length: 0` and `w.WriteHeader(401)`, as `httputil.ReverseProxy` relays an upstream's empty 401, keeps its empty answer; one that neither writes, flushes nor declares a length gets the envelope), when the status line is already on the wire, or when the connection was hijacked.

Where it is registered decides what it covers:

| Registration | Covers |
|---|---|
| `app.Use(errors.Handler)` on the root app | the routes and the router's 404 and 405; a `HandleNoRoute`/`HandleNoMethod` registered after it never runs |
| `app.UseRouter(errors.Handler)` | the routes and everything registered after it with `UseRouter`; **not** 404 and 405, because the router's default handler has written `Not Found` before the chain returns |
| `app.HandleNoRoute(errors.Handler)`, `app.HandleNoMethod(errors.Handler)` | 404 and 405 only |

An `iris.New()` application that wants the builder's behaviour registers the last three together:

```go
app.UseRouter(errors.Handler)
app.HandleNoRoute(errors.Handler)
app.HandleNoMethod(errors.Handler) // only reached when Options.Router.FireMethodNotAllowed is on
```

On an `iris.NewBuilder()` (and `iris.Default()`) application this is already done. A `HandleNoRoute` or `HandleNoMethod` of your own, registered from a `Configurator` given to `NewBuilder` or `Services`, or on the value `Default` returns, wins: the builder fills those slots only when they are empty at `Build`. Two failures are answered ahead of it and keep their bare status: a rejection from a handler a `Configurator` registered with `UseRouter`, and the 415 of the compression middleware for a request body that does not match its `Content-Encoding`. `FireMethodNotAllowed` is off by default, so a wrong method answers the 404 envelope until you turn it on.

A collector set with `SetCollector` therefore also receives every 404, 405 and bare 401 or 403 of a builder application.

## Usage Patterns

### Controller Error Handling

```go
// The payload validates itself, so ctx.BindJSON runs Validate after
// decoding and renders the field-level 400 without a handler branch.
func (in CreateUserInput) Validate() error {
    return validation.Join(
        validation.String("name", in.Name).NotEmpty().Length(3, 50),
        validation.String("email", in.Email).Email(),
    )
}

func (api *UserAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[CreateUserInput]()
    if !ok {
        return // 400 already written: malformed JSON, or a failed Validate.
    }

    user, err := api.service.CreateUser(ctx, input)
    ctx.Created(user, err) // 201, or the centrally mapped error.
}
```

### Service Layer Error Returns

```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err // Let the error mapping handle it
    }
    if user == nil {
        return nil, sql.ErrNoRows // Maps to NOT_FOUND
    }
    return user, nil
}
```

## Source Files

- `rest/errors/` - Error types, canonical codes, options, handlers
- `rest/errors/stream.go` - `Report` and `ErrorCodeName.Report`, the pipeline without a render
- `rest/errors/map_error_as.go` - `Options.MapErrorAs`, `ErrorCodeMap.MapErrorAs`, `MapErrorContains`
- `rest/errors/handler.go` - `Handler`, and what each registration point covers
- `rest/errors/not_found_id.go` - `NotFoundID`
- `rest/errors/pgerrors/pgerrors.go` - the opt-in PostgreSQL resolver, including the 23503 branch
- `rest/errors/validation/` - Field validation helpers (Field, String, Number, Slice, Join)
- `rest/errors/README.md` - RFC specification for HTTP Wire Errors
