# Migrating from v12 to v14

v14 is the largest release in the project's history: the framework was rebuilt around Go generics, with a compile-enforced application builder, typed request helpers that validate for you, controllers resolved by the dependency container, and an error pipeline configured once. Most v12 code needs mechanical rewrites plus a handful of real behavior changes.

Use this file when converting existing v12 code. For writing new v14 code from scratch, start at [getting-started-tutorial.md](./getting-started-tutorial.md) instead.

## Import Path

The module major version moved, so every import path changes:

```go
import "github.com/kataras/iris/v12" // before
import "github.com/kataras/iris/v14" // after
```

```sh
find . -name '*.go' -exec sed -i 's|github.com/kataras/iris/v12|github.com/kataras/iris/v14|g' {} +
sed -i 's|github.com/kataras/iris/v12|github.com/kataras/iris/v14|g' go.mod
```

## Package Layout

Sub-packages were reorganized. The prefix rewrite above does not fix these relocations:

| v12 import | v14 import |
|------------|------------|
| `.../v12/sessions` | `.../v14/middleware/sessions` |
| `.../v12/cache` | `.../v14/middleware/cache` |
| `.../v12/websocket` | `.../v14/middleware/websocket` |
| `.../v12/i18n` | `.../v14/middleware/i18n` |
| `.../v12/view` | `.../v14/middleware/view` |
| `.../v12/versioning` | `.../v14/middleware/versioning` |
| `.../v12/middleware/logger` | removed; use `.../v14/middleware/accesslog` (v12 shipped both) |
| `.../v12/middleware/recover` | `.../v14/middleware/recovery` |
| `.../v12/mvc` | `.../v14/controller/mvc` |
| `.../v12/macro` | `.../v14/core/router/macro` |
| `.../v12/hero` | built in via `app.RegisterDependency`; helpers in `.../v14/core/dep` |
| `.../v12/x/client` | `.../v14/client` |
| `.../v12/x/jsonx` | `.../v14/rest/types` |
| `.../v12/x/errors` | `.../v14/rest/errors` |
| `.../v12/x/mathx` | `.../v14/rest/math` |
| `.../v12/x/pagination` | `.../v14/rest` |
| `.../v12/x/timex` | `.../v14/core/pkg/timex` |
| `.../v12/x/reflex` | `.../v14/core/pkg/reflex` |
| `.../v12/x/sqlx` | removed; use `github.com/kataras/pg` |
| `.../v12/core/host` proxy helpers (`host.ProxyHandler`, `host.ProxyHandlerRemote`, `host.NewProxy`, `host.NewProxyRemote`) | removed; build a standard `httputil.ReverseProxy` and mount it on a route: `app.Any("/{p:path}", iris.FromStd(proxy))` |

The `context` package keeps its path (`.../v14/context`). MVC was redesigned around generics (`mvc.UseController[C]`, `mvc.Websocket`); see [builtin-controllers.md](./builtin-controllers.md).

View engine implementations moved out of the framework entirely, to `github.com/iris-contrib/views`. The `Blocks` engine was removed because `github.com/kataras/blocks` already satisfies the new `view.Engine` interface directly.

## Mechanical Renames

Ordered find-and-replace list. Applying these in order clears most of a v12 codebase:

| Find | Replace |
|------|---------|
| `iris.Party` | `iris.Router` |
| `Configure(r iris.Party)` | `Routing(r iris.Router)` |
| `.PartyConfigure(` | `.Router(` |
| `.Party(` | `.Router(` |
| `.PartyFunc(` | `.Router(` |
| `func(p iris.Party)` | `func(r iris.Router)` |
| `ctx.URLParam(` | `ctx.Query().Get(` |
| `ctx.URLParamDefault(` | `ctx.Query().Get(` (drop the default arg, check manually) |
| `ctx.URLParamSlice(` | `ctx.Query().GetSlice(` |
| `ctx.URLParamInt(` | `ctx.Query().GetInt(` (returns `0` invalid, `-1` missing) |
| `ctx.URLParamIntDefault(` | `ctx.Query().GetInt(` (drop the default arg) |
| `ctx.URLParamUint64(` | `ctx.Query().GetUint64(` |
| `ctx.URLParamFloat64Default(` | `ctx.Query().GetFloat64(` (drop the default arg) |
| `ctx.URLParamBoolDefault(` | `ctx.Query().GetBool(` (drop the default arg) |
| `ctx.Params().Get(` | `ctx.Params().GetString(` |
| `ctx.Header("` (to **set**) | `ctx.SetHeader("` |
| `ctx.GetHeader("` (to **read a request** header) | `ctx.Header("` |
| `ctx.RecordRequestBody(true)` | `_ = recorder.RecordRequestBody(ctx)` |
| `ctx.IsRecordingBody()` | `recorder.IsRequestBodyRecorded(ctx)` |
| `ctx.Record()` | the `recorder.Response` middleware, registered before the handlers that write |
| `ctx.Recorder()` | `recorder.GetResponseRecorder(ctx)` (nil when the response is not recorded; it does not start recording) |
| `ctx.IsRecording()` | `recorder.IsRecordingResponse(ctx)`, or a nil check on `recorder.GetResponseRecorder(ctx)` |
| `cache.Cache304` | `middleware/cache.New(cache.DefaultOptions)` |
| `iris.WithDynamicHandler`, `Configuration.EnableDynamicHandler` | `iris.Options{Router: iris.RouterOptions{EnableDynamicHandler: true}}` |
| `app.RefreshRouter()` | `app.BuildRouter()`, with that option set before the first build (without it a second call returns an error) |
| `app.AddRouteUnsafe(route)` | register the route as usual (`app.Get`, `app.Handle`); on a running server follow it with `app.BuildRouter()` |
| `route.ChangeMethod(m)`, `route.SetStatusOffline()`, `route.RestoreStatus()` | set the exported `route.Method` (`iris.MethodNone` takes the route offline); on a running server follow it with `app.BuildRouter()` |

The `*Default` query variants are gone because `ctx.Query()` accessors return documented zeros. Replace the default with an explicit check:

```go
page := ctx.Query().GetInt("page") // -1 when the key is absent, 0 when unparseable
if page <= 0 {
    page = 1
}
```

In v12 a `{id:uuid}` path parameter stored the text it validated. In v14 it stores a parsed `uuid.UUID`: read it with `ctx.Params().GetUUID("id")` or `ctx.BindParam[uuid.UUID]("id")`, and drop the old `uuid.Parse` call on the text along with its error branch. `ctx.Params().GetString("id")` still works and returns the canonical lowercase form. `ctx.BindParam[string]("id")` on such a route now answers 400.

Nested groups lose the closure parameter shadowing:

```go
// before
r.PartyFunc("/{blog_id:uuid}/posts", func(p iris.Party) { p.Get("/", listPosts) })
// after
r.Router("/{blog_id:uuid}/posts", func(r iris.Router) { r.Get("/", listPosts) })
```

## Replaced APIs

Each removed v12 name has exactly one v14 replacement.

### Configuration is replaced by Options

`iris.Configuration`, `iris.WithConfiguration` and the `With*` configurators are gone. A partial `iris.Options` literal configures only what it sets and inherits the rest. `(*Options).Bind` loads YAML from the `SERVER_CONFIG` environment variable first, then from a file.

```go
app := iris.New(iris.Options{Name: "myapp", LogLevel: "debug"})
// or edit-the-defaults style:
app := iris.New(iris.Defaults(func(o *iris.Options) { o.LogLevel = "debug" }))
```

### Party is replaced by Router

`app.Party` and `PartyFunc` are removed. `Router` creates groups and also accepts controllers and constructor functions resolved by the dependency container.

```go
api := app.Router("/api")         // was: app.Party("/api")
api.Router("/users", NewUsersAPI) // nested group; constructor resolved by DI
```

### Done, DoneGlobal and UseGlobal are replaced by the Use family

The post-handler hooks and `UseGlobal` are removed. The same `Use` call behaves differently on the root application than on a child router:

| Method | Runs on |
|--------|---------|
| `UseRouter` | before routing: every request, including 404/405 |
| `Use` on the root app | all requests, including 404/405 |
| `Use` on a child router | matched routes of that router only |
| `UseRelevant` on the root app | matched routes only |
| `UseOnce` / `UseRelevantOnce` | as above, replacing an already-registered instance |

A child router's `Use` delegates to `UseRelevant`, so the two are the same call on anything below the root.

### OnErrorCode is replaced by HandleNoRoute, HandleNoMethod and the errors options

`OnErrorCode` and `OnAnyErrorCode` are removed. Router-level error cases have dedicated methods; API error rendering goes through `rest/errors` (the `Errors` builder step).

```go
app.HandleNoRoute(notFoundHandler)   // was: app.OnErrorCode(404, ...)
app.HandleNoMethod(methodNotAllowedHandler)
app.EnableStatusCodeResponse()       // render responses for handler-set codes
```

### RegisterView and ctx.View are replaced by the view middleware

`app.RegisterView` and `ctx.View` are removed. Register the engine as middleware and render through the package function:

```go
import (
    "github.com/iris-contrib/views"
    "github.com/kataras/iris/v14/middleware/view"
)

templates := views.HTML("./views", ".html")
app.Use(view.Handler(templates))
// inside a handler:
view.Render(ctx, "index.html", "main-layout", data)
```

Different engines can be scoped per router: `app.Router("/admin", view.Handler(adminTemplates))`.

### iris.Addr, iris.TLS, iris.AutoTLS and Runner are replaced by app.Run

The `Runner` type and its constructors are removed. Servers are described by `iris.ServerOptions`; `app.Listen(addr)` remains the plain-HTTP shortcut.

```go
app.Run(iris.ServerOptions{
    Addr:     ":443",
    CertFile: "cert.pem",
    KeyFile:  "key.pem",
})
```

A failed start is reported differently. v12's `Run` logged a build or listen error and returned it, so a bare `app.Listen(":8080")` exited with status 0 when the port was taken. v14 logs the failure at fatal level and exits with status 1; set `iris.Options{DisableFatalRun: true}` to have the error returned as before, for a process that runs more than one server.

### SetContextErrorHandler is replaced by the context error-handler option

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{ErrorHandler: myErrorHandler},
})
// per request:
app.Use(iris.WithErrorHandler(myErrorHandler))
```

### NewBuilder and the rest packages are the recommended stack

`iris.New` plus manual wiring keeps working. New code should use `iris.NewBuilder` with handlers built on `rest` and `rest/errors`. See [architecture-and-patterns.md](./architecture-and-patterns.md).

```go
iris.NewBuilder().
    Prefix("/api").
    AllowOrigin("*").
    Compression(true).
    LogRequests(true).
    Health(true, "production", "kataras").
    Errors(errors.NewOptions()).
    Services(NewRepository, NewService).
    RouterMiddlewares().
    Middlewares().
    API("/users", NewUsersAPI).
    Build().
    Listen(":8080")
```

## Behavior Changes

### Context.Header changes meaning: request reads lose the Get prefix

v12 read a request header with `ctx.GetHeader(name)` and set a response header with `ctx.Header(name, value)`. v14 gives the request accessors the plain names and prefixes the response ones with `Set`:

| v12 | v14 | Acts on |
|-----|-----|---------|
| `ctx.GetHeader(k)` | `ctx.Header(k)` | request |
| `ctx.Header(k, v)` | `ctx.SetHeader(k, v)` | response |
| n/a | `ctx.ResponseHeader(k)` | response (new in v14) |

`ctx.Header` is a two-argument response setter in v12 and a one-argument request reader in v14, so the arity difference turns every stale call site into a compile error rather than a silent misread.

Translate a v12 `GetHeader` to `Header`, never to `ResponseHeader`. `ResponseHeader` reads what this handler already wrote, so it returns empty for anything the client sent; using it for `Authorization` or an API key silently disables the check.

```go
apiKey := ctx.Header("X-API-Key")       // request  (what the client sent)
ctx.SetHeader("X-Cache", "HIT")         // response
cached := ctx.ResponseHeader("X-Cache") // response (what this handler set)
```

That read is the porting rule, not a recommendation. A v12 middleware that read the key and compared it is replaced wholesale by `apikey.New`; see [middleware-reference.md](./middleware-reference.md#apikey).

Content type splits the same way: v12 `ctx.GetContentTypeRequested()` is v14 `ctx.ContentType()`, v12 `ctx.ContentType(v)` is v14 `ctx.SetContentType(v)`, and `ctx.ResponseContentType()` / `ctx.ResponseMediaType()` are new. `ctx.MediaType()`, `ctx.Charset()` and `ctx.IsMediaType(...)` are new request-side helpers.

### SetLogoutFunc and Logout take a handler instead of loose arguments

v12 stored an arbitrary function plus persistence arguments and invoked it through `CallFunc`. v14 stores a typed handler, so both signatures change:

```go
// was: ctx.SetLogoutFunc(fn any, persistenceArgs ...any); err := ctx.Logout(args...)
ctx.SetLogoutFunc(func(ctx iris.Context) error {
    return sessionManager.Destroy(ctx)
})
err := ctx.Logout()
```

### ResponseWriter.Naive is renamed to Unwrap

```go
w := ctx.ResponseWriter().Unwrap() // was: .Naive()
rc := http.NewResponseController(ctx.ResponseWriter())
```

### Application.Validator and Validate become a per-context validator

`Application.Validator` and `app.Validate(v)` are gone. The validator is a context option now, set globally through `iris.Options` or edited per request:

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{RequestValidator: myValidator},
})

app.Use(func(ctx iris.Context) {
    ctx.Options().RequestValidator = myValidator
    ctx.Next()
})
```

It runs after the request body is decoded, on `ReadJSON`, `ReadForm`, `ReadQuery`, `ReadValues`, `ReadHeader`, `ReadParams`, `ReadURL`, `ReadMsgPack` and `UnmarshalBody`.

> The v14-only capabilities that have no v12 counterpart (`ctx.BindJSON` running `Validate`, `ctx.BindJSONOptional`, `errors.HandleErrorf`, `MapErrorFuncCtx`, the `rest/errors/pgerrors` subpackage, `iris.Initializer`, `ctx.WithoutCancel`, `Builder.LogRequestsWith`, `Options` merging with the defaults) are not migration steps. They are covered in [rest-helpers.md](./rest-helpers.md), [error-handling.md](./error-handling.md), [context-api.md](./context-api.md) and [architecture-and-patterns.md](./architecture-and-patterns.md).

## Fixed Behavior

Bugs corrected in v14. Nothing to do unless a test asserted the broken behavior.

| Fixed | Was in v12 |
|-------|------------|
| `ctx.ReadBody` with `*string`/`*[]byte` | Fell through to the decoders after reading and re-read a drained body; the `*[]byte` case used `copy` into a zero-length slice, so those targets stayed empty |
| `x/jsonx.KitchenTime.Scan` (now `rest/types`) | A shadowed `minute` inside an `if` left the outer value at 0, so every scanned database `TIME` landed on the hour: `13:45:00` became `1:00 PM`. Already-truncated stored values cannot be recovered |
| `context.GetApplications` | `make([]Application, 0, len(registeredApps))` followed by `copy` copies zero elements, so the slice was always empty |

Other bug fixes in the v14 line (`middleware/rate` enforcement, `middleware/recorder` buffering, `mvc.PathParam.Get`, `errors.Router.UseRouter`, `ContextOptions.BodyMaxMemory`, `middleware/httpcost`, `Context.Logout`, the numeric validation wording) repaired code that has no v12 counterpart: those packages, types or options either did not exist in v12 or were already correct there. They are not migration steps.

## Breaking Security Defaults

These change behavior without a compile error. Each has a one-line escape hatch.

### Session cookies are Secure and SameSite=Lax

v12 set `Secure` only when `Config.CookieSecureTLS` was enabled and never set `SameSite` at all. v14's `sessions.NewOptions` defaults to `Secure: true` and `SameSite: http.SameSiteLaxMode`. Restore the old behavior only for plain-HTTP local development:

```go
opts := sessions.NewOptions(secret)
opts.Secure = false
opts.SameSite = http.SameSiteNoneMode // browsers require Secure for None
```

A `Secure` cookie still works behind a TLS-terminating proxy: the browser connection is HTTPS even when the app sees HTTP.

Session cookies a v12 application already handed out do not carry over, even under the same cookie name and keys. The built-in stores treat a cookie they cannot decode like no cookie: a new session, and `sessions.Handler` replaces the cookie. Expect everyone to sign in once after the upgrade.

### JWT tokens are read from the Authorization header only

v12 defaulted `Extractors` to `{FromHeader, FromQuery}`. v14's `jwt.NewVerifier` defaults to the `Authorization` header only. The `?token=` query extractor is off, because tokens in URLs leak into access logs, proxy logs, browser history and `Referer` headers. Opt back in only when a client cannot set a header:

```go
verifier := jwt.NewVerifier(jwt.HS256, key).WithQueryToken()
```

### CORS no longer advertises credentials with a wildcard origin

v12 wrote `Access-Control-Allow-Credentials` on every allowed request, including the ones it answered with `Access-Control-Allow-Origin: *`. Browsers reject that combination, so v14 sends credentials with a wildcard origin only when they are enabled explicitly.

The practical reach is narrow. A request that carries an `Origin` header gets that origin echoed back rather than `*`, which is every browser request, so credentials are advertised exactly as before. The wildcard applies only to requests with no `Origin` header at all. Opt back in there with the builder's `Cors` step:

```go
iris.NewBuilder().
    Prefix("/api").
    Cors(cors.New().
        HandleErrorFunc(errors.FailedPrecondition.Err).
        AllowOrigin("*").
        AllowCredentials()).
    Compression(true)
```

### AddCookie defaults to SameSite=Lax

`Context.AddCookie` sets `SameSite=Lax` (and `HttpOnly`) by default. Override per call:

```go
ctx.AddCookie(name, value, iris.CookieSameSite(http.SameSiteStrictMode))
```

> `auth/oauth2` and `auth/identity` are new in v14; v12's `auth` package has neither, and its `SigninHandler` always answered with JSON rather than redirecting. Their behavior is described in [authentication-sdk.md](./authentication-sdk.md), not here, because there is no v12 behavior to migrate from.

## Recommended Opt-in Hardening

New capabilities, off by default. Turning them on is recommended for production.

| Knob | Effect |
|------|--------|
| `ContextOptions.BodyMaxMemory` (bytes) | Caps `Body`, `ReadJSON`, `ReadJSONStream`, `ReadMultipartRelated` and bounds decompression-bomb reads. `0` (default) is unlimited. Per-route: the `bodylimit` middleware |
| `session.RegenerateID(ctx)` | Session-fixation defense. Call right after a privilege change, most importantly sign-in |
| `fileserver.BlockHidden` as `Options.AssetValidator` | Stops serving `.git`, `.env` and other dotfiles. Note it also blocks `.well-known` |
| `requestid.ServerOnlyGenerator` | Ignores a client-supplied `X-Request-ID`, preventing log forgery |
| `password.Cost` (auth/identity) above `bcrypt.DefaultCost` | Stronger password hashing |
| `apikey.New` (`middleware/apikey`) | v12 shipped no API key middleware, so every service wrote its own `X-API-Key` check. v14 replaces it: constant-time comparison over a fixed-width digest, one principal per key, `Hash: apikey.SHA256` for hashed storage, `Lookup` for keys in a database, and deny by default |

Full treatment with code in [security-hardening.md](./security-hardening.md).

## Source Files

- `MIGRATION.md` (repo root) - The canonical upgrade guide this file condenses
- `HISTORY.md` (repo root) - The `Thu, 12 November 2026 | v14.0.0` section, including the ordered find-and-replace list
- [book/23-migration-guide.md](https://github.com/kataras/iris-book/blob/main/book/23-migration-guide.md) - Upgrade strategy and tips
- `options.go`, `context/context_options.go` - `Options`, `applyDefaults`, `ContextOptions.FillZero`
- `rest/errors/resolver.go`, `rest/errors/pgerrors/` - `RegisterErrorResolver` and the PostgreSQL resolver
- `iris_builder.go` - The builder chain, `Initializer`, `LogRequestsWith`
