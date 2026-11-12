# Routing and Macros

## Route Registration

### HTTP Method Functions

```go
app.Get(path, handler)
app.Post(path, handler)
app.Put(path, handler)
app.Delete(path, handler)
app.Patch(path, handler)
app.Head(path, handler)
app.Options(path, handler)
app.Query(path, handler)                  // HTTP QUERY (RFC 10008): a read with a body
app.Any(path, handler)                    // All methods, QUERY included
app.Handle("GET", path, handler)          // Generic, returns *RouteRegistry
app.Handle("GET POST", path, handler)     // Multiple methods (space-separated)
```

Method functions (`Get`, `Post`, ...) return a `*Route`. `Handle` and `Any` return a `*RouteRegistry` of the created routes; `registry.Query()` picks its QUERY route, like `registry.Get()` and `registry.Post()`.

`iris.MethodQuery` (also `router.MethodQuery`) is the `"QUERY"` constant, for `Handle("POST QUERY", "/list", h)` and other places that take a method string. QUERY is safe, idempotent and carries a body, so a search or a filtered list can be cached (see caching.md) and retried without being a POST. A struct registered with `Router(path, &T{})` that has a `Query(ctx iris.Context)` method answers QUERY on `/`, next to `Get`, `Post` and the other method-named methods (`Query(ctx) error` under `rest/errors`, whose `Router.Query(path, funcs...)` registers error-returning QUERY handlers). Send a QUERY body as JSON: on Go 1.27 net/http parses a url-encoded body only for POST, PUT and PATCH, so it never reaches `ctx.Form*`.

When a request is redirected to its trailing-slash twin, POST, PUT, PATCH and QUERY get 307 Temporary Redirect so the body survives; other methods get 301.

### Handler Signature

```go
type Handler = func(iris.Context)
```

### Route Groups

```go
// Using Router() for grouping
usersRouter := app.Router("/users")
usersRouter.Get("/", listUsers)
usersRouter.Post("/", createUser)
usersRouter.Get("/{id:uuid}", getUser)

// Nested groups
api := app.Router("/api")
v1 := api.Router("/v1")
v1.Router("/users", NewUserAPI)
v1.Router("/products", NewProductAPI)

// With middleware
admin := app.Router("/admin", basicauth.Default(creds))
admin.Get("/dashboard", dashboard)

// With controllers (recommended)
app.Router("/users", NewUserAPI)
```

### Route with Multiple Controllers

```go
app.Router("/api",
    iris.Middleware(authMiddleware),   // Middleware as controller
    iris.Dependencies(specialDep),    // Extra dependencies
    NewUserAPI,                       // Controller
    NewProductAPI,                    // Another controller
)
```

## Route Registration Rules

Control behavior when registering duplicate routes (same method, subdomain and path). The rule lives on the route registry and applies to the router and its children:

```go
app.Routes().SetRegisterRule(iris.RouteOverride)  // Replace the existing route (default)
app.Routes().SetRegisterRule(iris.RouteSkip)      // Keep the original route, skip the new one
app.Routes().SetRegisterRule(iris.RouteError)     // Log duplicates after Build, server never starts
```

| Rule | Behavior |
|------|----------|
| `iris.RouteOverride` | New route replaces the existing one. The default. |
| `iris.RouteSkip` | Existing route is kept, the new registration is ignored. |
| `iris.RouteError` | Duplicate is reported as an error after the Build state and the server does not start. |

## Dynamic Route Management

Routes can be registered offline, looked up and removed by name. Route names are unique per application. The default name is method + subdomain + path, for example `"GET/users/{id:uuid}"`.

```go
// Offline route: registered with the virtual method "NONE" (iris.MethodNone),
// not reachable over HTTP.
maintenance := app.None("/maintenance", handler)
_ = maintenance.IsOnline() // false

// Look up a route by its unique name.
r := app.Routes().GetByName("GET/users/{id:uuid}")

// A route is online when its Method is a real HTTP method.
// Toggle state by swapping the exported Method field.
r.Method = iris.MethodNone // offline
r.Method = "GET"           // online

// Remove a route by name, before Application.Listen.
// Reports whether it was found and removed.
ok := app.RemoveRoute("GET/users/{id:uuid}")
// Equivalent: app.Routes().RemoveByName("GET/users/{id:uuid}")
```

Changing routes after the server has started requires the `EnableDynamicHandler` router option (`Options.Router.EnableDynamicHandler = true`). It wraps the mux with an RWMutex so routes can change at serve-time, at a small performance cost. Defaults to false.

## Path Parameter Macros

Iris provides behavioral attributes for dynamic route path parameters while maintaining peak performance.

### Built-in Parameter Types

| Type | Path Syntax | Go Type | Description |
|------|------------|---------|-------------|
| `string` | `{name}` or `{name:string}` | `string` | Any non-empty string (default) |
| `int` | `{id:int}` | `int` | Integer (alias: `number`) |
| `int8` | `{n:int8}` | `int8` | 8-bit integer (-128 to 127) |
| `int16` | `{n:int16}` | `int16` | 16-bit integer |
| `int32` | `{n:int32}` | `int32` | 32-bit integer |
| `int64` | `{n:int64}` | `int64` | 64-bit integer (alias: `long`) |
| `uint` | `{id:uint}` | `uint` | Unsigned integer |
| `uint8` | `{n:uint8}` | `uint8` | 8-bit unsigned (0-255) |
| `uint16` | `{n:uint16}` | `uint16` | 16-bit unsigned |
| `uint32` | `{n:uint32}` | `uint32` | 32-bit unsigned |
| `uint64` | `{n:uint64}` | `uint64` | 64-bit unsigned |
| `bool` | `{active:bool}` | `bool` | Values accepted by strconv.ParseBool: 1, 0, t, f, true, false, ... (alias: `boolean`) |
| `alphabetical` | `{name:alphabetical}` | `string` | Letters and spaces (`^[a-zA-Z ]+$`) |
| `file` | `{file:file}` | `string` | Letters, numbers, underscore, dash, dot. No spaces |
| `path` | `{rest:path}` | `string` | Multi-segment wildcard, must be the last segment |
| `uuid` | `{id:uuid}` | `uuid.UUID` | UUID v4 (and v1) format (alias: `uuidv4`); `GetString` returns the canonical lowercase text |
| `mail` | `{addr:mail}` | `string` | Email address, syntax check only (mail.ParseAddress) |
| `email` | `{addr:email}` | `string` | Email address, syntax plus MX domain lookup (stricter than `mail`) |
| `date` | `{d:date}` | `time.Time` | Date in YYYY/MM/DD format |
| `weekday` | `{day:weekday}` | `time.Weekday` | Day of week name |

### Constraints

Constraints are applied after the type, separated by spaces:

| Constraint | Applies To | Example | Description |
|-----------|-----------|---------|-------------|
| `min(n)` | string, int, uint | `{name:string min(3)}` | Minimum length (string) or value (numbers) |
| `max(n)` | string, int, uint | `{name:string max(20)}` | Maximum length (string) or value (numbers) |
| `range(a,b)` | int, uint | `{age:int range(1,120)}` | Value between a and b |
| `regexp(expr)` | string | `{code:string regexp(^[A-Z]{3}$)}` | Regex match |
| `prefix(s)` | string | `{name:string prefix(user_)}` | Must start with s |
| `suffix(s)` | string | `{name:string suffix(.pdf)}` | Must end with s |
| `contains(s)` | string | `{name:string contains(admin)}` | Must contain s |
| `eq(v)` | string | `{state:string eq(active)}` | Exact equality with v |
| `eqor(v1,v2,...)` | string | `{flag:string eqor(on,off)}` | Equals any of the listed values |
| `else N` | all | `{id:int else 400}` | HTTP status on failure |

A typed macro that has no function of the requested name inherits the string one, which compares against the value's text: `{id:int prefix(4)}` and `{id:uuid prefix(a6)}` both work.

### Constraint Examples

```go
// Integer with range and error code
r.Get("/page/{page:int min(1) max(100) else 400}", handler)

// String with regex validation
r.Get("/users/{username:string min(3) max(20) regexp(^[a-zA-Z0-9_]+$)}", handler)

// Exact match or one-of
r.Get("/status/{state:string eq(active)}", handler)
r.Get("/toggle/{flag:string eqor(on,off) else 400}", handler)

// UUID parameter
r.Get("/users/{id:uuid}", handler)

// Wildcard path (matches remaining segments)
r.Get("/files/{filepath:path}", handler)

// Email parameter
r.Get("/lookup/{email:email}", handler)

// Combined constraints
r.Get("/code/{code:string min(3) max(10) prefix(PRD) else 400}", handler)
```

### Accessing Parameters

Typed accessors return the value directly, not a (value, error) tuple:

```go
func handler(ctx iris.Context) {
    // String parameter
    name := ctx.Params().GetString("name") // "" if missing

    // Typed parameters with their failure defaults
    id := ctx.Params().GetInt("id")          // -1 if missing or not an int
    id64 := ctx.Params().GetInt64("id")      // -1 on failure (all int sizes: -1)
    uid := ctx.Params().GetUint64("id")      // 0 on failure (all uint sizes: 0)
    active := ctx.Params().GetBool("active") // false on failure
    when := ctx.Params().GetDate("d")        // zero time.Time on failure
    sid := ctx.Params().GetUUID("sid")       // zero uuid.UUID (uuid.Nil()) on failure
    ttl := ctx.Params().GetDuration("ttl")   // 0 on failure
    at := ctx.Params().GetTime("at")         // RFC 3339; zero time.Time on failure
    ip := ctx.Params().GetIP("ip")           // zero netip.Addr (IsValid() false) on failure

    // Existence check
    if v, ok := ctx.Params().Lookup("name"); ok {
        _ = v // the evaluated value as any
    }
}
```

Failure defaults: `GetString` returns `""`, the `GetInt*` and `GetFloat*` family return `-1`, the `GetUint*` family returns `0`, `GetBool` returns `false`.

### Custom Macro Registration

Register custom parameter types or add functions to existing types:

```go
// Add a custom function to the string type
app.Macros().Get("string").RegisterFunc("equals", func(expected string) func(string) bool {
    return func(paramValue string) bool {
        return paramValue == expected
    }
})

// Usage: /status/{status:string equals(active)}
r.Get("/status/{status:string equals(active)}", handler)

// Register a completely new parameter type
app.Macros().Register("datetime", "", time.Time{}, false, false,
    func(paramValue string) (any, bool) {
        t, err := time.Parse("2006-01-02T15:04:05", paramValue)
        if err != nil {
            return nil, false
        }
        return t, true
    },
)

// Usage: /events/{when:datetime}
r.Get("/events/{when:datetime}", handler)
```

### Macro Error Handler

```go
// Set custom error handler for parameter validation failures
app.Macros().SetErrorHandler(func(ctx iris.Context, paramIndex int, err error) {
    errors.InvalidArgument.Detailf(ctx, "invalid path parameter",
        "parameter %d: %v", paramIndex, err)
})

// The builder automatically sets this to errors.DefaultPathParameterTypeErrorHandler
```

## Subdomain Routing

### Static Subdomains

```go
// Route to specific subdomain
admin := app.Subdomain("admin")
admin.Get("/", adminDashboard)
// Matches: admin.example.com/

api := app.Subdomain("api")
api.Get("/users", listUsers)
// Matches: api.example.com/users

// Nested: subdomain is prepended, this registers "panel.admin."
panel := app.Subdomain("admin").Subdomain("panel")
panel.Get("/", panelIndex)
// Matches: panel.admin.example.com/
```

### Wildcard Subdomains

```go
// Match any subdomain
wild := app.Subdomain("*") // or app.WildcardSubdomain()
wild.Get("/", func(ctx iris.Context) {
    subdomain := ctx.Subdomain()
    ctx.Writef("Subdomain: %s", subdomain)
})
// Matches: anything.example.com/
```

### Matching Semantics

Matching enforces dot boundaries. A request subdomain matches a registered subdomain only on exact match or as a dot-separated child:

| Request subdomain | Registered | Match? |
|-------------------|-----------|--------|
| `admin` | `admin` | Yes (exact) |
| `staff.admin` | `admin` | Yes (dot-separated child) |
| `badmin` | `admin` | No (suffix without a dot boundary) |

## Route Ordering

Route sorting at build time is stable (`slices.SortStableFunc`):

- Longer subdomains come first.
- For the same subdomain and method, paths with more slashes come first.
- Static routes (no parameters) sort before parameterized routes of equal depth.
- Parameterized routes of equal priority keep their registration order.

When several routes share the same path position with different parameter types (for example `/users/{id:int}` and `/users/{name:string}`), register the less specific fallback (`{string}`, `{path}`) first. The router evaluates the more specific types (`int`, `uuid`, ...) first at request time and falls back to `{string}`/`{path}` last.

## Named Routes and URL Generation

```go
// Name a route (default name: method + subdomain + path)
route := app.Get("/users/{id:uuid}", getUser)
route.SetName("user.get") // or: route.Name = "user.get"

// Look it up and build a concrete path from arguments
r := app.Routes().GetByName("user.get")
path := r.ResolvePath("42") // "/users/42"
```

## WrapRouter

A router wrapper runs before routing happens at all, so it can rewrite the path without a redirect. The signature is `func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)`, and skipping `next` short-circuits the whole router.

```go
app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
    if rest, ok := strings.CutPrefix(r.URL.Path, "/legacy"); ok {
        r.URL.Path = "/v2" + rest // clients keep the old URL, no 301
    }
    next(w, r)
})
```

Three registration methods, differing only in ordering. Getting this wrong is the usual cause of "my rewrite runs too late":

| Method | Ordering |
|--------|----------|
| `app.WrapRouter(fn)` | Last registered runs **first** |
| `app.AddRouterWrapper(fn)` | Deferred to build time; first registered runs **first** |
| `app.PrependRouterWrapper(fn)` | Always first, ahead of everything above |

Use `PrependRouterWrapper` for path normalization that everything else depends on (case folding, trailing slashes).

`app.Downgrade(http.HandlerFunc)` replaces the router entirely with a plain handler. Wrappers are bypassed, so it is an escape hatch rather than a composition tool.

Note the wrapper methods live on the embedded `*router.RouterBuilder` and take a `router.WrapperFunc`; `app.WrapRouter` reaches them through embedding.

## Converting Standard Handlers

```go
// Convert net/http handlers to Iris handlers
irisHandler := iris.FromStd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("from std"))
}))

app.Get("/std", irisHandler)

// Route parameters and the route template reach the standard handler:
app.Get("/users/{id:uuid}", iris.FromStd(func(w http.ResponseWriter, r *http.Request) {
    _ = r.PathValue("id") // the parameter
    _ = r.Pattern         // "/users/{id:uuid}"
}))
```

## Source Files

- `core/router/router_builder.go` - RouterBuilder: Handle, Router, Subdomain, None
- `core/router/route_registry.go` - RouteRegistry: register rules, name lookup, removal, sorting
- `core/router/route.go` - Route struct: Name, Method, IsOnline, ResolvePath
- `core/router/macro/macros.go` - Built-in parameter types and constraint functions
- `core/router/router_options.go` - RouterOptions struct (EnableDynamicHandler and friends)
- `context/encoding/entries.go` - Params() typed accessors
