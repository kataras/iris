---
name: iris
description: "Development guide for the Iris v14 Go web framework.
  Use this skill when building web applications, REST APIs, or microservices with
  Iris. Covers architecture patterns, routing, middleware, error handling, dependency
  injection, streaming and AI backends, migration from v12, security defaults,
  testing, and deployment."
tags: [go, iris, web_framework, webserver]
---

# Iris Web Framework v14

## Quick Start

```go
package main

import "github.com/kataras/iris/v14"

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "Hello, World!"})
    })
    app.Listen(":8080")
}
```

- Module path: `github.com/kataras/iris/v14`
- Go 1.27+

For a complete guided walkthrough of the recommended stack (builder,
controllers, rest helpers, error mapping, DI, testing), see
[getting-started-tutorial.md](./references/getting-started-tutorial.md).

## Reference Index

Load the one that matches the task; each is self-contained.

| Reference | Load it when |
|-----------|--------------|
| [getting-started-tutorial.md](./references/getting-started-tutorial.md) | Building a v14 app from scratch, end to end |
| [migration-v12-to-v14.md](./references/migration-v12-to-v14.md) | Converting v12 code, or a v12 API does not compile |
| [architecture-and-patterns.md](./references/architecture-and-patterns.md) | Builder chain, DI, service lifecycle, layering, patches |
| [routing-and-macros.md](./references/routing-and-macros.md) | Route registration, path macros, subdomains, router wrappers |
| [context-api.md](./references/context-api.md) | Reading a request or writing a response |
| [rest-helpers.md](./references/rest-helpers.md) | `ctx.BindJSON`, `OK`/`Created`/`NoContent`, pagination |
| [sse.md](./references/sse.md) | Server-Sent Events: streaming a response, or reading somebody else's |
| [ai.md](./references/ai.md) | AI backends: model streams, relays, upstream failure classification |
| [error-handling.md](./references/error-handling.md) | Canonical codes, error mapping, validation, privacy |
| [middleware-reference.md](./references/middleware-reference.md) | Any of the 31 bundled middleware packages |
| [builtin-controllers.md](./references/builtin-controllers.md) | Generic MVC controllers, file server, API graph, sitemap |
| [authentication-sdk.md](./references/authentication-sdk.md) | JWT, identity server, OAuth2, RBAC |
| [security-hardening.md](./references/security-hardening.md) | Secure defaults, CSRF, headers, production checklist |
| [database-and-persistence.md](./references/database-and-persistence.md) | Repository wiring, `kataras/pg`, driver error mapping |
| [caching.md](./references/caching.md) | Cache headers, 304s, server-side response cache |
| [i18n.md](./references/i18n.md) | Translations, pluralization, language detection |
| [observability.md](./references/observability.md) | Access logs, request IDs, monitor, pprof, profiling |
| [performance.md](./references/performance.md) | Streaming, benchmarks, what the framework already does |
| [http-client.md](./references/http-client.md) | Outbound HTTP calls from a service |
| [testing-guide.md](./references/testing-guide.md) | `httptest`, handler and controller tests |
| [configuration-and-deployment.md](./references/configuration-and-deployment.md) | `Options`, TLS, tunnels, graceful shutdown |

## Install and Run

```sh
mkdir myapp && cd myapp
go mod init myapp
go get github.com/kataras/iris/v14@latest
go run .
```

There is no built-in CLI and no live reload. Restart the process on change, under whatever file watcher you already use. See [configuration-and-deployment.md](./references/configuration-and-deployment.md).

## Architecture Overview

Three application creation modes:

| Mode | When to Use |
|------|-------------|
| `iris.NewBuilder()` | **Recommended.** Fluent step-by-step builder with built-in best practices |
| `iris.New()` | Manual setup with full control |
| `iris.Default()` | Quick prototyping (debug logging, CORS `*`, compression, recovery) |

### Builder Pattern (Recommended)

```go
iris.NewBuilder().
    Prefix("/api").
    AllowOrigin("*").
    Compression(true).
    LogRequests(true).
    Health(true, "production", "team@company.com").
    Errors(errors.NewOptions().MapErrors(errors.NotFound, sql.ErrNoRows)).
    Deferrables(db).
    Services(NewRepoRegistry, NewUserService).
    RouterMiddlewares(requestid.New()).
    Middlewares().
    API("/users", NewUserAPI).
    API("/admin", basicauth.Default(creds), NewAdminAPI).
    Build().
    Listen(":8080")
```

Builder step order: **Prefix/NoPrefix → AllowOrigin/Cors → Compression → LogRequests/LogRequestsWith → Health/NoHealth → Errors → Deferrables → Services → RouterMiddlewares/Middlewares → API → Build**

Services implementing `iris.Initializer` (`Init(ctx context.Context) error`)
are initialized once inside `Build()`, after dependency injection. Services
implementing `iris.Worker` (`Run(ctx) error` / `Stop(ctx) error`) are started
after the initializers and stopped before the closers, and `Shutdown` waits
for `Run` to return.

See [architecture-and-patterns.md](./references/architecture-and-patterns.md) for full details.

### Controller Pattern

```go
type UserAPI struct {
    service UserService
}

func NewUserAPI(service UserService) *UserAPI {
    return &UserAPI{service: service}
}

func (api *UserAPI) Routing(r iris.Router) {
    r.Get("/", api.list)
    r.Post("/", api.create)
    r.Get("/{id:uuid}", api.getByID)
    r.Put("/{id:uuid}", api.update)
    r.Delete("/{id:uuid}", api.delete)
}
```

Built-in controllers ship in the `controller/` package: FileServer (`controller/fileserver`), MVC (`controller/mvc`), APIGraph (`controller/apigraph`), Sitemap (`controller/sitemap`).

`mvc.UseController[*C](opts...)` and `mvc.UseRepository(repo, opts...)` both generate the six resource routes and both take `mvc.Guard`, `mvc.Without`, `mvc.IDParam`, `mvc.BeforeWrite` and `mvc.Pagination`. `UseRepository` needs no controller type at all: the repository's five methods over a standard `context.Context` are the whole resource.

### Layered Architecture

```
Handler/Controller -> Service -> Repository -> Domain
```

Register services via `Services()` in the builder. Controllers receive them through constructor injection.

## Core Concepts

| Concept | Type/Signature | Description |
|---------|---------------|-------------|
| Handler | `func(iris.Context)` | Request handler function |
| Context | `iris.Context` (alias for `*context.Context`) | Request/response object |
| Router | `iris.Router` | Route group interface |
| Controller | `iris.Controller` | Interface with `Routing(iris.Router)` method |
| Options | `iris.Options` | Application configuration |
| Configurator | `iris.Configurator` | Interface for configuring the app |
| Map | `iris.Map` | Alias for `map[string]any` |

## Routing

HTTP methods: `Get`, `Post`, `Put`, `Delete`, `Patch`, `Head`, `Options`, `Any`, `Handle`

### Path Parameter Macros

| Macro | Example | Description |
|-------|---------|-------------|
| `{name:string}` | `/users/{name:string}` | String (default) |
| `{id:int}` | `/users/{id:int}` | Integer (all sizes: int8-64) |
| `{id:uint}` | `/users/{id:uint}` | Unsigned integer (uint8-64) |
| `{id:uuid}` | `/users/{id:uuid}` | UUID format |
| `{active:bool}` | `/filter/{active:bool}` | Boolean |
| `{name:alphabetical}` | `/org/{name:alphabetical}` | Letters only |
| `{file:file}` | `/assets/{file:file}` | Filename with extension |
| `{rest:path}` | `/files/{rest:path}` | Multi-segment wildcard |
| `{email:email}` | `/lookup/{email:email}` | Email address |
| `{d:date}` | `/report/{d:date}` | Date (YYYY/MM/DD) |
| `{day:weekday}` | `/schedule/{day:weekday}` | Day of week |

Constraints: `min(n)`, `max(n)`, `range(min,max)`, `regexp(expr)`, `prefix(s)`, `suffix(s)`, `contains(s)`, `else 400`

```go
r.Get("/users/{id:int min(1) else 400}", handler)
r.Get("/files/{path:path}", handler)
```

See [routing-and-macros.md](./references/routing-and-macros.md) for full reference.

## Error Handling

Use the `rest/errors` package with canonical error codes:

| Code | HTTP Status | Usage |
|------|-------------|-------|
| `errors.InvalidArgument` | 400 | Bad client input |
| `errors.NotFound` | 404 | Resource not found |
| `errors.Unauthenticated` | 401 | Missing/invalid credentials |
| `errors.PermissionDenied` | 403 | Insufficient permissions |
| `errors.AlreadyExists` | 409 | Duplicate resource |
| `errors.Internal` | 500 | Server error |
| `errors.ResourceExhausted` | 429 | Rate limited |
| `errors.BadGateway` | 502 | An upstream this service depends on failed |

Key methods:

```go
errors.NotFound.Detail(ctx, "user not found", "no user with id 123")
errors.Internal.Err(ctx, err)              // Log error, send generic message
errors.Internal.LogErr(ctx, err)           // Log + send error response
errors.InvalidArgument.Data(ctx, "bad input", dataMap)
errors.InvalidArgument.Validation(ctx, validationErrors...)
errors.InvalidArgument.Message(ctx, "custom message")
```

Error mapping in builder:

```go
Errors(errors.NewOptions().
    MapErrors(errors.NotFound, sql.ErrNoRows).
    MapErrors(errors.InvalidArgument, ErrValidation).
    MapErrorFunc(errors.Internal, func(err error) error { ... }).
    MapErrorContains(errors.InvalidArgument, "violates check constraint", "invalid value"))

// Typed: a generic method, E inferred from the function.
opts.MapErrorAs(errors.ResourceExhausted, func(e *QuotaError) *errors.Error { ... })
```

`errors.NotFoundID(ctx, "plan", id)` renders `NOT_FOUND` with the detail
`"plan <id> not found"`; it always returns false (the `Context.OK` convention),
the opposite of `HandleError`.

See [error-handling.md](./references/error-handling.md) for full reference.

## REST Response Helpers

The Context helpers combine the success response and error mapping into one call. Each returns false when it wrote an error response instead.

```go
ctx.OK(resp, err)            // 200 JSON, or mapped error response
ctx.Created(resp, err)       // 201 JSON, id scalars wrapped as {"id": ...}
ctx.NoContent(modified, err) // 204 when modified, 304 when not
ctx.Count(n, err)            // 200 {"count": N}, zero included
ctx.List(items, err, tolerate...) // 200 JSON array; nil and empty both render []
ctx.JSONOrError(resp)        // JSON now, with encode failures rendered as errors
ctx.Paginated(collectFunc, opts...) // paginated list responses; iris.AllowSort/DenySort
ctx.HandleError(err)         // the error branch alone, for a step with no success body
errors.Unavailable.Handle(ctx, err) // same, forcing the code instead of mapping it
```

See [rest-helpers.md](./references/rest-helpers.md) for full reference.

## Context API

### Request Reading

The `Bind*` family is what handlers call: it decodes, validates and writes the 4xx itself. The `Read*` family below is the layer under it, for a target the binder cannot produce.

```go
ctx.BindJSON[T]()          // JSON body, validated; false means the 4xx was sent
ctx.BindJSONOptional[T]()  // same, but an empty body yields the zero T
ctx.BindParam[T]("id")     // one path parameter, typed
ctx.BindParams[T]()        // every path parameter, into a struct
ctx.ReadJSON(&v)           // JSON body
ctx.ReadXML(&v)            // XML body
ctx.ReadYAML(&v)           // YAML body
ctx.ReadForm(&v)           // Form data
ctx.ReadQuery(&v)          // Query parameters
ctx.ReadBody(&v)           // Auto-detect content type
ctx.ReadProtobuf(&v)       // Protobuf binary
ctx.ReadMsgPack(&v)        // MsgPack binary
ctx.Params().Get("id")     // Path parameter
ctx.Query().Get("key")     // Query parameter
ctx.Params().GetUUID("id") // also GetDuration, GetTime, GetIP on Params/Query/Form; zero value on a miss
ctx.BindUpdate[T]()        // JSON body + the validated ?columns= list
ctx.QueryDate("from", iris.QueryRequired()) // strict readers: a bad value is a 400
ctx.QueryDate("date day from")             // space-separated keys, tried in order
ctx.QueryInt("limit", iris.QueryDefault("25"))
ctx.QueryBool("detailed")
ctx.Header("key")          // Request header (ResponseHeader reads response headers)
ctx.Body()                 // Raw body bytes
ctx.FormFile("upload")     // File upload
```

### Response Writing

The response helpers above (`ctx.OK`, `ctx.Created`, `ctx.NoContent`, `ctx.Count`, `ctx.List`, `ctx.Paginated`) carry the status and the error branch. These writers are what they are built on.

```go
ctx.JSON(v)                // JSON response
ctx.XML(v)                 // XML response
ctx.YAML(v)                // YAML response
ctx.HTML("<h1>Hi</h1>")   // HTML response
ctx.Markdown([]byte("# Title"))
ctx.Text("plain text")    // Plain text
ctx.MsgPack(v)             // MsgPack binary
ctx.Protobuf(v)            // Protobuf binary
ctx.StatusCode(iris.StatusCreated)
ctx.SetHeader("key", "val")
ctx.Redirect("/path", iris.StatusFound)
ctx.SendFile("path/to/file", "download.pdf")
ctx.RenderComponent(component)
```

### Request-Scoped Values

```go
ctx.Values().Set("key", value)
ctx.Values().Get("key")
ctx.SetUser(user)
ctx.User()
```

See [context-api.md](./references/context-api.md) for full reference.

## Middleware

| Package | Description |
|---------|-------------|
| `accesslog` | Structured request/response logging (JSON, CSV, template) |
| `apikey` | Shared-secret authentication for services, cron jobs and webhook senders |
| `basicauth` | HTTP Basic Authentication with flexible backends |
| `bodylimit` | Request body size limits |
| `cache` | HTTP caching headers, ETag, Cache-Control |
| `compress` | Gzip/Brotli/Deflate response compression |
| `cors` | Cross-Origin Resource Sharing |
| `counter` | Request counter metrics |
| `geolocation` | IP-based geolocation |
| `grpc` | gRPC integration |
| `hcaptcha` | hCaptcha verification |
| `httpcost` | Request cost tracking |
| `i18n` | Internationalization and localization |
| `ipaccess` | IP allowlist/blocklist |
| `jwt` | JSON Web Token auth (sign, verify, blocklist) |
| `methodoverride` | HTTP method override via header/form |
| `modrevision` | Build info / health endpoint |
| `monitor` | Application monitoring |
| `pprof` | Go profiling endpoints |
| `rate` | Token bucket rate limiting |
| `recaptcha` | Google reCAPTCHA verification |
| `recorder` | Response recorder for middleware chains |
| `recovery` | Panic recovery with stack traces |
| `referrer` | Referrer tracking and parsing |
| `requestid` | Unique request ID generation |
| `rewrite` | URL rewriting rules |
| `servertiming` | Server-Timing header |
| `sessions` | Server-side sessions (cookie, PostgreSQL, Redis) |
| `versioning` | API versioning (header, query, subdomain) |
| `view` | Template engine integration |
| `websocket` | WebSocket via neffos library |

**Global middleware** (all requests including 404s): `app.UseRouter(handler)` or `RouterMiddlewares(...)`
**Route middleware** (matched routes only): `app.Use(handler)` or `Middlewares(...)`

See [middleware-reference.md](./references/middleware-reference.md) for full reference.

## Authentication SDK

The `auth/` package provides a complete authentication and identity system:

- **JWT Authentication**: EdDSA-signed access + refresh tokens
- **Identity Server**: User management with PostgreSQL, OAuth2, 2FA
- **RBAC**: Resource-action policies and scope validation
- **Session Management**: Secure cookie storage with AES-GCM encryption
- **OAuth2**: Authorization Code, Device Code, Password Grant flows
- **Client SDK**: Multi-language identity SDK for consuming the auth API
- **Request scope**: `auth.Scope(derive)` after `VerifyHandler` derives a tenant id once; `auth.ScopeID[ID](ctx)` reads it. A missing user, or one of another type, is a 401 (`auth.ErrMissingUser`); a derived zero id is stored as it is

See [authentication-sdk.md](./references/authentication-sdk.md) for full reference.

## HTTP Client

The `client/` package provides the HTTP client for outbound calls from your services. See [http-client.md](./references/http-client.md) for full reference.

## Testing

```go
func TestAPI(t *testing.T) {
    app := iris.NewBuilder().
        // ... builder config ...
        Build()

    e := httptest.New(t, app)
    e.GET("/api/users").Expect().Status(httptest.StatusOK).
        JSON().Array().NotEmpty()
    e.POST("/api/users").WithJSON(user).Expect().Status(httptest.StatusCreated)
}
```

Key functions:
- `httptest.New(t, app, ...options)` - Create test client from app
- `httptest.NewController(t, prefix, controller, mw...)` - A throwaway app around one controller, which is what `Routing` values need
- `httptest.NewInsecure(t, ...options)` - Test running TLS servers
- `httptest.Do(w, r, handler)` - Test individual handlers

See [testing-guide.md](./references/testing-guide.md) for full reference.

## Configuration and Deployment

- **Options**: `iris.Options{Name, LogLevel, Host, Router, Context}`
- **YAML config**: `iris.New(iris.YAML("config.yml"))`
- **Environment**: `opts.Bind("config.yml")` checks `SERVER_CONFIG` env var first
- **Whole-service config**: `iris.LoadConfiguration[T](envKey, filename)` with `iris.BaseConfiguration` embedded inline; the variable may hold the document *or* a path to it, `BindEnv()` runs after decoding, and `c.Addr()` feeds `app.Listen`. There is no `app.Serve`
- **Server**: `app.Listen(":8080")` or `app.Run(iris.ServerOptions{...})`, called as statements: both block, return nil after a graceful shutdown, and on a failed start log at fatal level and exit with status 1. `iris.Options{DisableFatalRun: true}` logs at error level and returns the error instead (several servers in one process, or a test with a real listener)
- **TLS**: Certificate files, custom `tls.Config`, or automatic Let's Encrypt
- **Tunneling**: Ngrok (`NgrokOptions`, opt-in: needs `import _ "github.com/kataras/iris/v14/core/server/ngrok"` or the server refuses to start) or SSH reverse tunnel
- **Graceful shutdown**: `app.Shutdown(ctx)`
- **Build info**: `iris.PrintBuildInfo(w)`, `iris.Version`, `iris.BuildRevision`

See [configuration-and-deployment.md](./references/configuration-and-deployment.md) for full reference.

## Streaming and AI

`sse.Open(ctx)` writes nothing: the protocol headers commit on the first `Send`, so a check that fails after opening still returns an ordinary status code. That first `Send` removes buffering middleware, clears the server's absolute `WriteTimeout` and sets the anti-buffering headers. `Close` writes the terminal frame on success too, so its absence is how a client detects truncation. `sse.Read` consumes an upstream stream as an `iter.Seq2`. `Timeout.HandlerTimeout` makes streaming impossible, which is what `sse.ErrNotStreamable` reports.

`ai.Open(ctx, opts)` serves the streaming and the non-streaming shape from one handler (`Options.Stream` is a `*bool`, so nil negotiates on `Accept`). Seven `Part` kinds, three neutral encoders, `ai.Relay` with a `Decoder` you write, and `ai.Classify(resp, err)` for a typed `*ai.UpstreamError` whose `Class` says whether to retry, switch credential or stop. Never retry after `Committed()`.

A failure after the first byte cannot be rendered, so it goes out as an error frame. `errors.Report(ctx, err)` runs the `rest/errors` pipeline (mapping, logging, collection, id, redaction) and returns the `*Error` without writing it, which is what `sse.Options.ErrorFrame` defaults to. Never forward a provider's own message to your caller: `(*ai.UpstreamError).Report(ctx)` logs it and gives the client a canonical code instead.

See [sse.md](./references/sse.md) and [ai.md](./references/ai.md) for full reference.

## Migrating from v12

v14 renamed a lot. The traps that compile fine but misbehave, or that the model most often gets wrong:

| v12 | v14 |
|-----|-----|
| `iris.Party`, `app.Party`, `PartyFunc` | `iris.Router`, `app.Router` |
| `Configure(r iris.Party)` | `Routing(r iris.Router)` |
| `ctx.URLParam*` (whole family) | `ctx.Query().Get*` |
| `ctx.Header("k", "v")` (set) | `ctx.SetHeader("k", "v")` |
| `ctx.GetHeader("k")` (read request) | `ctx.Header("k")` |
| `iris.Configuration`, `With*` | `iris.Options`, `iris.Defaults` |
| `app.RegisterView`, `ctx.View` | `view.Handler(engine)`, `view.Render(ctx, ...)` |
| `app.OnErrorCode` | `app.HandleNoRoute`, `HandleNoMethod`, `Errors(...)` |

Security defaults also changed (session cookies `Secure`+`Lax`, JWT header-only, no wildcard CORS credentials). See [migration-v12-to-v14.md](./references/migration-v12-to-v14.md).

## Security

v14 is secure by default in most places, with two deliberate gaps: **Iris ships no CSRF middleware and injects no security headers**. Two permissive defaults worth flipping: `ContextOptions.BodyMaxMemory` is `0` (unlimited), and the file server serves dotfiles.

```go
iris.Options{Context: iris.ContextOptions{BodyMaxMemory: 10 * iris.MB}}
fileserver.Options{AssetValidator: fileserver.BlockHidden}
```

See [security-hardening.md](./references/security-hardening.md) for full reference.

## Database

No ORM. Register the store through `Services(...)` and the builder handles the lifecycle: a `Close()` method becomes a shutdown closer, and `Init(ctx) error` (`iris.Initializer`) runs inside `Build()` after DI. Translate driver errors to domain sentinels at the store boundary, then map those once in `Errors(...)`. With `github.com/kataras/pg`, add `import _ ".../rest/errors/pgerrors"` to keep 404/409 mapping.

See [database-and-persistence.md](./references/database-and-persistence.md) for full reference.

## Caching

Three layers: `middleware/cache` writes headers (its validators are clock-derived, **not** content-derived), `middleware/cache/httpcache` stores responses in Redis/DragonflyDB with `Pair`-based invalidation, and `core/pkg/cache` is an in-process map. Identity must be in the cache key (`Headers`, `Cookies`) or do not response-cache the route.

See [caching.md](./references/caching.md) for full reference.

## Internationalization

`middleware/i18n` needs no third-party dependency. `NewRouter` on `UseRouter` strips a language path prefix or subdomain before routing; `NewHandler` on `Use` resolves the locale. Then `i18n.GetMessage(ctx, key, args...)`.

See [i18n.md](./references/i18n.md) for full reference.

## Observability

`middleware/accesslog` on `UseRouter` (so 404s are logged) with `Async = true` and bodies off. Add `requestid` before it. Gate `monitor` and `pprof` behind auth. Two notes: the monitor's `RefreshInterval` defaults to `monitor.DefaultRefreshInterval` (2 seconds) when unset, and `httpcost` writes its metric only after the chain returns.

See [observability.md](./references/observability.md) for full reference.

## Performance

The framework already pools contexts, stores params in slices rather than maps, and resolves routes through prefix trees built at startup. Do not rebuild those. The hard rule: **never retain `iris.Context` past the handler's return**; pass `ctx.StdContext()` to consumers that keep the context (HTTP clients, database drivers) and use `ctx.WithoutCancel()` for background work. For large payloads use `ctx.StreamWriter`, `ctx.ReadJSONStream` and `ctx.Paginated`.

See [performance.md](./references/performance.md) for full reference.

## Key File Paths

| Area | Path |
|------|------|
| Application | `iris.go`, `iris_builder.go`, `aliases.go`, `options.go` |
| Context | `context/context.go` |
| Router | `core/router/` |
| Errors | `rest/errors/` |
| REST helpers | `rest/` (response helpers; `rest/errors/` for canonical codes) |
| HTTP client | `client/` |
| Built-in controllers | `controller/` (FileServer, MVC, APIGraph, Sitemap) |
| Scalar payload types | `rest/types/` (dates, durations, `Number`, `Weekday`) |
| Multi-app registry | `apps/` |
| Middleware | `middleware/*/` |
| Auth | `auth/`, `auth/identity/`, `auth/oauth2/` |
| Streaming | `sse/` (transport), `ai/` (model streams, relays, classification) |
| Testing | `httptest/` |
| Server | `core/server/` |
| Book | [kataras/iris-book](https://github.com/kataras/iris-book) (24 chapters, separate repository) |
| Tutorial | `TUTORIAL.md` (repo root), `references/getting-started-tutorial.md` |
| Migration | `MIGRATION.md` (repo root), `HISTORY.md` |
| Examples | `_examples/` (16 numbered topic dirs; `01-getting-started` is a 13-step tutorial, then `02-routing` … `16-advanced`) |

## Conventions

- **Constructors**: `NewXxx` returning interface or pointer; used for DI
- **Controller receivers**: `(api *XxxAPI)` or `(c *XxxController)`
- **Error handling**: Use `rest/errors` canonical codes, not raw HTTP status codes
- **DI**: Register services in builder's `Services()`, consumed by controller constructors
- **Middleware**: Always call `ctx.Next()` to continue the chain
- **Response methods**: `ctx.JSON()`, `ctx.XML()`, `ctx.HTML()`, etc.
- **Query and path params**: Read query parameters via `ctx.Query()` and path parameters via `ctx.Params()`; both return typed entries with direct accessors (`Get`, `GetInt64`, `GetBool`, etc.), no tuple returns
- **Byte sizes**: Use `iris.B`, `iris.KB`, `iris.MB`, `iris.GB` constants
- **Context as context.Context**: `iris.Context` implements `context.Context`; pass directly to calls that return before the handler does. Never retain it past the handler's return: it is pooled. Pass `ctx.StdContext()` to HTTP clients and anything else that keeps the context; use `ctx.WithoutCancel()` for background work
- **Route groups**: Use `r.Router("/sub", controllers...)` for nested groups
- **Request headers**: `ctx.Header(k)` reads the request, `ctx.ResponseHeader(k)` reads the response, `ctx.SetHeader(k, v)` replaces and `ctx.AddHeader(k, v)` appends. There is no `GetHeader`
- **Validation**: Give the request type a `Validate() error`; `ctx.BindJSON` runs it and renders the 400 for you
