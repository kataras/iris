# Architecture and Patterns

## Application Creation Modes

### iris.NewBuilder() (Recommended)

The builder provides a fluent, step-by-step API that enforces correct middleware ordering and includes production best practices (recovery, compression, CORS, error handling).

```go
iris.NewBuilder(configurators ...Configurator) Builder
```

#### Builder Step Interfaces

Each method returns the next step interface, enforcing order:

```
Builder          -> Prefix(path) / NoPrefix()
CorsBuilder      -> AllowOrigin(origins) / Cors(*cors.CORS)
CompressionBuilder -> Compression(bool)
RequestLoggingBuilder -> LogRequests(bool) / LogRequestsWith(*accesslog.AccessLog)
HealthBuilder    -> Health(bool, env, developer) / NoHealth()
ServiceBuilder   -> Errors(*errors.Options) / Deferrables(...any) / Services(...any)
MiddlewareBuilder -> RouterMiddlewares(...any) / Middlewares(...any)
APIBuilder       -> API(prefix, ...controllers) / Build() *Application
```

#### Full Builder Example

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/rest/errors"
    "github.com/kataras/iris/v14/middleware/basicauth"
    "database/sql"
)

func main() {
    iris.NewBuilder().
        Prefix("/api").                           // All API routes under /api
        AllowOrigin("https://myapp.com").         // CORS for specific domain
        Compression(true).                        // Gzip/Brotli compression
        LogRequests(true).                        // Access log to access_log.json
        Health(true, "production", "team@co.com").// GET /health endpoint
        Errors(errors.NewOptions().               // Error mapping
            MapErrors(errors.NotFound, sql.ErrNoRows)).
        Deferrables(func() { /* cleanup */ }).    // Shutdown hooks
        Services(                                 // Dependency injection
            NewRepoRegistry,
            NewUserService,
        ).
        RouterMiddlewares().                      // Global middleware (all requests)
        Middlewares().                             // API-only middleware
        API("/users", NewUserAPI).                // Register controllers
        API("/admin",
            basicauth.Default(map[string]string{"admin": "secret"}),
            NewAdminAPI,
        ).
        Build().                                  // Build the application
        Listen(":8080")                           // Start server
}
```

#### Builder Built-in Behaviors

The builder automatically registers:
- **Recovery middleware** with `errors.Internal.LogErr` error handler
- **Server header** (`Server: Iris`)
- **X-Developer header** (if developer string provided to Health)
- **Compression middleware** (if enabled)
- **`rest/errors.Handler`**, after compression, last of the built-in router middleware: a status of 400 or above with nothing written (stopped or not) answers the `rest/errors` JSON envelope
- **`rest/errors.Handler` as the 404 and 405 handler** at `Build`, unless the application registered its own `HandleNoRoute`/`HandleNoMethod` (which wins). The 405 needs `Options.Router.FireMethodNotAllowed`; off by default, a wrong method answers the 404
- **Rate limit exceed handler** bound to the errors package
- **Macro error handler** using `errors.DefaultPathParameterTypeErrorHandler`
- **Context error handler** using `errors.DefaultContextErrorHandler`

#### Builder Hardening

- The step interfaces enforce the documented call order at the type level: each method returns only the next legal steps.
- An internal `sync.Once` guards the one-time setup performed by `RouterMiddlewares` (macro error handler, errors options handler, recovery, default headers, compression, `errors.Handler`, rate-limit exceed handler). `Middlewares()` invokes `RouterMiddlewares()` first, so the built-ins apply exactly once regardless of call order or repeated calls. User middlewares passed to each call are still registered.
- `Services()` panics immediately on a dependency registration error: the app fails fast at startup instead of misbehaving later. It also routes values by shape: `func()` and `func() error` become shutdown closers (not dependencies), values with `Close()` or `Close() error` are registered as both dependency and closer, `Configurator` values are applied to the app, nil entries are skipped.
- Services (registered values or constructor outputs) implementing `iris.Initializer` (`Init(ctx context.Context) error`) are initialized exactly once inside `Build()`, after dependency injection, in registration order, the startup counterpart of `Deferrables`. An `Init` error aborts the build.
- `Deferrables(...)` forwards each value to `app.AddCloser`. Accepted types: `func()`, `func() error` (the error is preserved and joined into the shutdown error), `func(context.Context)`, `func(context.Context) error`, `io.Closer`, and the closer package's context-aware `Closer`. Any other type panics at registration time.
- `Build()` panics on a build error.

### iris.New()

Manual setup with full control. No middleware or configuration is applied automatically.

```go
app := iris.New()
app.Use(recovery.Handler)
app.Get("/", handler)
app.Listen(":8080")
```

With options:

```go
app := iris.New(iris.Options{
    LogLevel: "debug",
    Host:     "example.com",
    Router: iris.RouterOptions{
        EnablePathIntelligence: true,
    },
})
```

### iris.Default()

Quick prototyping with sensible defaults: debug logging, CORS `*`, compression, recovery, and request ID middleware.

```go
app := iris.Default()
app.Get("/", handler)
app.Listen(":8080")
```

Equivalent to:

```go
iris.NewBuilder().
    NoPrefix().
    AllowOrigin("*").
    Compression(true).
    LogRequests(false).
    NoHealth().
    Services().
    RouterMiddlewares(requestid.New()).
    Middlewares()
```

The chain stops before `Build()`: `Default` returns an **unbuilt** application, so routes registered after the call are served. Its compression is bounded by `compress.Safe`.

## Build Lifecycle Callbacks

`Application` exposes two exported hooks, both `func() error`:

- `app.OnBuild`: fired at the start of `Build()`, before the router is built. An error halts the build.
- `app.OnBuilt`: fired at the end of `Build()`, after the router is built.

Register them with `RegisterOnBuild(beforeBuild, afterBuild func() error)`. Either argument may be nil. Repeated calls chain: previously registered callbacks run first, in registration order, and the chain stops on the first error.

```go
app := iris.New()
app.RegisterOnBuild(
    func() error { // before router build
        app.Logger().Info("validating configuration")
        return nil
    },
    func() error { // after router build
        app.Logger().Info("routes are ready")
        return nil
    },
)
```

`Build()` runs automatically on `Listen`/`Run`, or call it manually (e.g. for tests). Internal features such as the cache middleware, sitemap, and apigraph controllers use these hooks too.

## Controller Pattern

Controllers are structs that implement the `Routing(r iris.Router)` method. They are registered via the builder's `API()` method or directly via `app.Router()`.

### Controller Interface

```go
type Controller interface {
    Routing(r Router)
}
```

### Constructor Injection

Controller constructors receive dependencies registered via `Services()`:

```go
type UserAPI struct {
    service  UserService
    logger   *log.Logger
}

func NewUserAPI(service UserService, logger *log.Logger) *UserAPI {
    return &UserAPI{service: service, logger: logger}
}

func (api *UserAPI) Routing(r iris.Router) {
    r.Get("/", api.list)
    r.Post("/", api.create)
    r.Get("/{id:uuid}", api.getByID)
    r.Put("/{id:uuid}", api.update)
    r.Delete("/{id:uuid}", api.delete)

    // Sub-groups
    r.Router("/settings", NewUserSettingsAPI)
}

func (api *UserAPI) list(ctx iris.Context) {
    users, err := api.service.ListUsers(ctx)
    ctx.List(users, err) // 200 with [] when empty; errors go through the map.
}
```

### Registration

```go
// Via builder
.API("/users", NewUserAPI)

// Via direct registration
app.Router("/users", NewUserAPI(service, logger))

// With middleware
.API("/admin", basicauth.Default(creds), NewAdminAPI)
```

### Controller Helpers

All helpers return a `Controller`, so they compose inside `Router()`/`API()` calls.

| Helper | Signature | Purpose |
|--------|-----------|---------|
| `iris.Middleware` | `Middleware(handlers ...iris.Handler) Controller` | Wraps plain handlers as a controller (registers them with `r.Use`) |
| `iris.Dependencies` | `Dependencies(deps ...any) Controller` | Overrides inherited dependencies for the controllers that follow in the same call |
| `iris.Join` | `Join(controllers ...any) Controller` | Combines multiple controllers into one |
| `iris.Prefix` | `Prefix(relativePath string, controllers ...any) Controller` | Mounts controllers under a relative path |
| `iris.ResolveController` | `ResolveController[T any]() Controller` | Reuses a `*T` controller instance already registered as a dependency, resolved from the router's DI container |

There is no conditional controller helper. Registration is ordinary Go: branch on the flag where you build the router.

```go
// Plain handlers as middleware inside a controller list.
app.Router("/admin", iris.Middleware(authHandler), NewAdminAPI)

// Per-group dependency override.
app.Router("/users", iris.Dependencies(mockService), NewUserAPI)

// Reuse an already-registered *AdminAPI dependency.
app.Router("/admin", iris.ResolveController[AdminAPI]())

// Register the billing API only when the feature flag is on.
if flags.BillingEnabled {
    app.Router("/billing", iris.ResolveController[BillingAPI]())
} else {
    app.Router("/billing", iris.Middleware(comingSoonHandler))
}
```

## Dependency Injection

### How Services() Works

Dependencies passed to `Services()` are resolved by type when constructing controllers:

```go
Services(
    config,                    // *AppConfig (value)
    NewLogger,                 // func() *log.Logger (factory)
    NewUserRepository,         // func(db *DB) UserRepository (factory with deps)
    NewUserService,            // func(repo UserRepository) *UserService (factory)
)
```

**Resolution rules:**
1. **Values** are registered directly by their type
2. **Factory functions** are called with their parameters resolved from registered dependencies
3. **Interfaces** are matched by the implementing type
4. **Closers** (types with `Close()` or `Close() error`) are automatically registered for graceful shutdown
5. **Configurators** (implementing `Configurator`) are applied to the app, not registered as dependencies
6. A registration error makes `Services()` panic at startup (fail fast)
7. **Initializers** (implementing `iris.Initializer`) get `Init(ctx) error` called inside `Build()` (see below)

### Startup Initialization

`Build()` walks the resolved singletons through `dep.Container.VisitSingletons` and calls `Init(ctx context.Context) error` on every one that implements `iris.Initializer`:

```go
type Initializer interface {
    Init(ctx context.Context) error
}
```

The contract:

- Runs **after** dependency injection and **before** the server starts, exactly once, in registration order.
- Only singletons are visited. Request-scoped dependencies are skipped: their values do not exist yet.
- Comparable values (pointers included) are deduplicated, so a value registered directly *and* returned by a constructor is initialized once.
- An error aborts the build (`Build()` panics with `iris: builder: init: <type>: <err>`).
- Implementations derive their own timeouts from the passed context.

It is the startup counterpart of `Deferrables`. Use it for schema creation, cache warmup and connection priming, instead of hiding that work in `Routing` where it runs at route-registration time and cannot fail cleanly.

```go
func (s *CatalogService) Init(ctx context.Context) error { return s.warmCache(ctx) }
func (s *CatalogService) Close() error                   { return s.pool.Close() }
```

A service with both methods needs no lifecycle code anywhere else: `Services(NewCatalogService)` wires startup and shutdown.

### Registering Dependencies Directly

```go
app.RegisterDependency(db, logger, config)
```

### Per-Route Dependencies

```go
app.Router("/users",
    iris.Dependencies(specialService),  // Only for this group
    NewUserAPI,
)
```

## Layered Architecture

### Recommended Structure

```
/cmd/server/main.go          # Entry point, builder setup
/api/                         # Controllers (API layer)
  user_api.go
  admin_api.go
/service/                     # Business logic (Service layer)
  user_service.go
/repository/                  # Data access (Repository layer)
  user_repository.go
  registry.go                 # Repository registry
/domain/                      # Domain models
  user.go
/pkg/                         # Shared utilities
```

### Repository Registry Pattern

```go
type RepoRegistry interface {
    Users() UserRepository
    Products() ProductRepository
    InTransaction(ctx context.Context, fn func(RepoRegistry) error) error
}

type sqlRepoRegistry struct {
    db       *DB
    users    UserRepository
    products ProductRepository
}

func NewSQLRepoRegistry(db *DB) RepoRegistry {
    return &sqlRepoRegistry{
        db:       db,
        users:    NewSQLUserRepository(db),
        products: NewSQLProductRepository(db),
    }
}
```

### Service Pattern

```go
type UserService struct {
    repos RepoRegistry
}

func NewUserService(repos RepoRegistry) *UserService {
    return &UserService{repos: repos}
}

func (s *UserService) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // Validation, business logic
    user := &User{Name: input.Name, Email: input.Email}
    return s.repos.Users().Create(ctx, user)
}
```

## Context Wrapper

Create custom context types that extend `iris.Context`. `NewContextWrapper` takes two type parameters: the struct type and its pointer type. The pointer type must implement `SetContext(iris.Context)`.

```go
type myContext struct {
    iris.Context
}

func (c *myContext) SetContext(ctx iris.Context) {
    c.Context = ctx
}

func (c *myContext) CurrentUser() *User {
    return c.Values().Get("user").(*User)
}

// Create wrapper: struct type and its pointer.
w := iris.NewContextWrapper[myContext, *myContext]()

// Use in routes
app.Get("/", w.Handler(func(ctx *myContext) {
    user := ctx.CurrentUser()
    ctx.JSON(user)
}))
```

The default pool acquires a zero value per request. For custom acquire/release behavior see `context.NewContextWrapperWithPool`.

### Manual Context Release

`app.ContextPool()` returns the sync pool that holds request contexts. Iris releases contexts automatically; it is only needed to manually release a context when a third-party middleware hijacks the connection and the HTTP handler returns early.

## Global Patches

`iris.Patches()` returns a process-wide singleton for overriding global behavior across all Iris applications. Apply patches once at startup, before serving.

| Patch | Effect |
|-------|--------|
| `Patches().Context().Writers().JSON/JSONP/XML(fn)` | Replaces the default body writer behind `ctx.JSON`, `ctx.JSONP`, `ctx.XML`. Signature: `func(ctx Context, v any, options *iris.JSON) error` (and the matching options type per writer) |
| `Patches().Context().Writers().YAML(fn)` | Signature differs: `func(ctx Context, v any, indentSpace int) error` |
| `Patches().Context().Writers().Markdown(fn)` | Signature differs: `func(ctx Context, v []byte, options *iris.Markdown) error` |
| `Patches().Context().GetDomain(fn)` | Changes how `Context.Domain()` derives the domain from host:port (subdomain redirect, i18n cookie, rewrite middleware) |
| `Patches().Context().SetDefaultCookieExpiration(d)` | Changes `context.DefaultCookieExpiration` (365 days), the lifetime `Context.AddCookie` gives a cookie with no `CookieExpires` option |
| `Patches().Context().ResolveHTTPFS(fn)` | Changes filesystem resolution for the file server |
| `Patches().Context().ResolveFS(fn)` | Changes filesystem resolution for the view engine |
| `Patches().Context().SetMaxPageSize(n)` | Sets `pagination.MaxPageSize`, the upper bound of `PageOptions.GetLimit` (default 100000) |
| `Patches().Context().SetDefaultPageSize(n)` | Sets `pagination.DefaultPageSize`, used when a list request sends no size (default 100) |

```go
import jsoniter "github.com/json-iterator/go"

iris.Patches().Context().Writers().JSON(
    func(ctx iris.Context, v any, options *iris.JSON) error {
        enc := jsoniter.ConfigCompatibleWithStandardLibrary.NewEncoder(ctx.ResponseWriter())
        enc.SetIndent("", options.Indent)
        return enc.Encode(v)
    })
```

## Generic Controllers

```go
type BaseController[T Entity, TCreate, TUpdate any] struct {
    service Service[T]
}

func (c *BaseController[T, TCreate, TUpdate]) Routing(r iris.Router) {
    r.Get("/", c.list)
    r.Post("/", c.create)
    r.Get("/{id:uuid}", c.getByID)
    r.Put("/{id:uuid}", c.update)
    r.Delete("/{id:uuid}", c.delete)
}
```

## Multi-App Composition (apps package)

`github.com/kataras/iris/v14/apps` is a registry of all Iris Application instances within the same program. It is not a collection of sample applications.

- Applications register themselves automatically on creation. `app.SetName("name")` gives one a unique name.
- `apps.Get(appName ...string) *iris.Application`: fetch by name from any package (no import cycles). With no argument it returns the last registered app, or creates a "Default" one if none exist.
- `apps.GetAll() []*iris.Application`: all registered apps.
- `apps.Switch(provider SwitchProvider, options ...SwitchOption) *iris.Application`: returns a new app that routes each request to the first matching registered app. Providers supply `SwitchCase{Filter, App}` values; `Hosts` matches host patterns, `Join` combines providers, and the `SetHost(...)` option rewrites the request host for the matched app.

```go
switcher := apps.Switch(apps.Hosts{
    {Pattern: "mydomain.com", Target: app},
    {Pattern: "test.mydomain.com", Target: testSubdomainApp},
    {Pattern: apps.AnyDomain, Target: catchAllApp},
})
switcher.Listen(":80")
```

`Target` accepts a `*iris.Application`, a registered app name, or a host string (which becomes a permanent redirect). `apps.OnApplicationRegistered(fn)` observes registrations from another package, and `context.GetApplications()` returns read-only `context.Application` values.

Switch mechanics worth knowing: each case is built on its own `Build` through `OnBuild`, the filters are installed as a `UseRouter` chain in registration order, and a match forwards to that app's `ServeHTTP` (its own context pool and configuration). When nothing matches, the request falls through to the switcher's own router, so you can give the switcher its own routes as the fallback.

## Source Files

- `iris.go` - Application struct (incl. OnBuild/OnBuilt), New(), Default(), Listen(), Run(), Build(), Shutdown(), RegisterOnBuild(), ContextPool(), NewContextWrapper()
- `iris_builder.go` - NewBuilder(), all builder interfaces and implementation
- `aliases.go` - Type aliases, constants, controller helpers, Patches()
- `options.go` - Options struct, Configurator interface, YAML loading
- `core/server/closer/closer.go` - Closer types accepted by AddCloser/Deferrables
- `apps/` - Multi-app registry, Get(), Switch(), Hosts provider
