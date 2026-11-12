# Migrating from Iris v12 to v14

This guide is the canonical reference for upgrading an application from
`github.com/kataras/iris/v12` to `github.com/kataras/iris/v14`. It covers the
module import path, the package layout, the replacements for removed v12
APIs, the `Context` method changes, the v12 bugs whose fixes change what you
observe, and the secure-by-default behavior changes. The v14 APIs themselves
are documented in the [book](https://iris-go.com/book/), whose chapters are
free to read at [iris-go.com/docs](https://iris-go.com/docs/), and hands-on in
[TUTORIAL.md](TUTORIAL.md). Start there if you are new to Iris rather than
upgrading.

Every removed v12 name fails at compile time. Work through the compiler's
error list with the tables below and the mechanical part of the upgrade is
done. If you would rather have an AI agent do that part, the next section is a
prompt written for exactly this migration.

## Automated migration with an AI agent

Most of this guide is mechanical: rename, move, re-import, adjust an argument.
Coding agents are good at that and bad at knowing which Iris name replaced
which, so the prompt below carries the mapping with it. Open your agent in the
repository you are upgrading and paste the whole block.

It is deliberately conservative. It changes only what the compiler would have
rejected anyway, and it reports the security-default changes instead of
reverting them, because those are decisions about your application rather than
syntax. Read that report before you accept the diff.

````text
You are upgrading this Go repository from the Iris web framework v12
(github.com/kataras/iris/v12) to v14 (github.com/kataras/iris/v14). Work
through the phases in order and do not start a phase before the previous one
builds as far as it can. Do not reformat, restructure or "improve" code you
were not asked to change.

PHASE 1: import path.
Rewrite every occurrence of github.com/kataras/iris/v12 to
github.com/kataras/iris/v14, in go.mod and in every .go file. Run
`go mod tidy`. Iris v14 requires Go 1.27 or later; if the go directive is
lower, raise it and say so.

PHASE 2: relocated packages.
The prefix rewrite above does not fix these. Apply each mapping:
  .../sessions            -> .../middleware/sessions
  .../cache               -> .../middleware/cache
  .../websocket           -> .../middleware/websocket
  .../i18n                -> .../middleware/i18n
  .../view                -> .../middleware/view
  .../versioning          -> .../middleware/versioning
  .../middleware/recover  -> .../middleware/recovery
  .../mvc                 -> .../controller/mvc
  .../macro               -> .../core/router/macro
  .../x/client            -> .../client
  .../x/jsonx             -> .../rest/types
  .../x/errors            -> .../rest/errors
  .../x/mathx             -> .../rest/math
  .../x/pagination        -> .../core/pkg/pagination (ListOptions -> PageOptions,
      List[T] -> Page[T], NewList -> NewPage; also iris.PageOptions, iris.Page[T]).
      NewPage takes the same four arguments in another order:
      NewList(items, totalCount, filter, opts) -> NewPage(opts, filter, items, totalCount)
  .../x/timex             -> .../core/pkg/timex
  .../x/reflex            -> .../core/pkg/reflex
The context package keeps its path (.../v14/context).
Removed with no drop-in replacement, so report each use rather than guessing:
middleware/logger (use middleware/accesslog), hero (use the built-in DI
container), x/sqlx (use github.com/kataras/pg). View engine implementations
moved to github.com/iris-contrib/views; the Blocks engine was dropped in
favour of github.com/kataras/blocks.

Four relocated packages also changed their entry point:
  cache.Handler(expiration) / cache.Cache(fn) -> app.Use(cache.New(cache.DefaultOptions))
  recover.New()                               -> app.UseRouter(recovery.Handler)
  websocket.Handler(server, idGen...)         -> websocket.New(server, websocket.Options{})

PHASE 3: removed APIs.
  iris.Configuration, iris.WithConfiguration, iris.With*  -> iris.Options (a struct literal;
      it is itself a Configurator, so a partial literal sets only what it names)
  iris.Party, app.Party(prefix)                           -> iris.Router, app.Router(prefix)
  PartyConfigure                                          -> Router; Configure -> Routing
  Done, DoneGlobal, UseGlobal                             -> the Use family
  OnErrorCode                                             -> HandleNoRoute, HandleNoMethod,
      or the rest/errors options
  RegisterView, ctx.View                                  -> the view middleware
  iris.Addr, iris.TLS, iris.AutoTLS, iris.Runner          -> app.Run(iris.ServerOptions{...})
  SetContextErrorHandler                                  -> Options.Context.ErrorHandler
  HandleDir, Favicon                                      -> the fileserver controller
  WWW, SubdomainRedirect                                  -> the rewrite middleware
  Application.Validator, Validate                         -> Options.Context.RequestValidator
  mvc.GRPC                                                -> controller/mvc.GRPC
  Configuration.Tunneling, iris.WithTunneling             -> ServerOptions.Ngrok plus
      import _ "github.com/kataras/iris/v14/core/server/ngrok"
  auth.Must(auth.New[T](cfg))                             -> auth.Must[T](cfg)
  jwt.Get(ctx).(*Claims)                                  -> jwt.Get[*Claims](ctx),
      which returns the claims and an ok bool
  ctx.Values().Get(hcaptcha.ResponseContextKey)           -> hcaptcha.Get(ctx)
  client.BindResponse(resp, &v)                           -> v, err := client.Bind[T](resp)
  context.IsErrPathCRSFToken                              -> context.IsErrPathCSRFToken
  context.SetCookieKVExpiration                           -> context.DefaultCookieExpiration
  Patches().Context().SetCookieKVExpiration(d)            -> SetDefaultCookieExpiration(d)

PHASE 4: Context method changes. This is the largest phase.
Query values: all eighteen URLParam* methods are gone. Use ctx.Query():
  ctx.URLParam(k)          -> ctx.Query().Get(k)
  ctx.URLParamExists(k)    -> ctx.Query().Exists(k)
  ctx.URLParamSlice(k)     -> ctx.Query().GetSlice(k)
  ctx.URLParamInt(k)       -> ctx.Query().GetInt(k)
  ctx.URLParamInt64(k)     -> ctx.Query().GetInt64(k)
  ctx.URLParamUint64(k)    -> ctx.Query().GetUint64(k)
  ctx.URLParamFloat64(k)   -> ctx.Query().GetFloat64(k)
  ctx.URLParamBool(k)      -> ctx.Query().GetBool(k)
  ctx.URLParams()          -> ctx.Query().ToMap()
  ctx.URLParamsSorted()    -> ctx.Query()
The *Default forms lose their default argument: keep the caller's fallback as
an explicit check. The numeric getters return -1 when the key is absent and 0
when it is present but unparseable, so `if v <= 0 { v = fallback }` preserves
v12 behavior for positive defaults.

Path parameters are stored in their macro type and ctx.Params().Get returns
any. Use the typed getters: ctx.Params().GetString("id"), GetInt("n").

Headers split by side. ctx.Header changes meaning between versions, so check
every call by its arity:
  ctx.GetHeader(k)               -> ctx.Header(k)          (reads the request)
  ctx.Header(k, v)               -> ctx.SetHeader(k, v)    (writes the response)
  ctx.GetContentTypeRequested()  -> ctx.ContentType()
  ctx.ContentType(v)             -> ctx.SetContentType(v)
ctx.SetHeader(k, v) replaces any value already set for that name, so a
middleware registered twice cannot emit a duplicate Access-Control-Allow-Origin
that browsers reject; ctx.AddHeader(k, v) appends, for a list-valued header
such as Vary.

Renamed:
  ctx.GetCookie(name)          -> ctx.CookieValue(name)
  ctx.GetRequestCookie(name)   -> ctx.Cookie(name)
  ctx.GetBody()                -> ctx.Body()
  ctx.SetCookieKV(k, v, ...)   -> ctx.AddCookie(k, v, ...)
  ctx.UpsertCookie(c, ...)     -> ctx.SetCookie(c, ...)
  ctx.ReadHeaders(ptr)         -> ctx.ReadHeader(ptr)
  ctx.RequestContentLength()   -> ctx.ContentLength()
  ctx.ResetResponseWriter(w)   -> ctx.SetResponseWriter(w)
  ctx.ResetRequest(r)          -> ctx.SetRequest(r)
  ctx.SetMaxRequestBodySize(n) -> bodylimit.New(n), or Options.Context.BodyMaxMemory
  ctx.GetReferrer()            -> the middleware/referrer middleware
  ctx.Problem(v, opts)         -> errors.<ErrorCodeName>.Problem(ctx, ...)
  ctx.CompressWriter(true)     -> the middleware/compress middleware
Removed with no replacement, so report each call site: VisitAllCookies,
ReflectValue, Controller, RegisterDependency, UnregisterDependency, SetFunc,
GetFunc, CallFunc.
Removed, with another way to do the job:
  ctx.DisablePoolRelease()          -> ctx.Clone() for use after the handler returns,
                                       or ctx.OnRelease(fn) for cleanup
  ctx.OnConnectionCloseErr(fn)      -> ctx.OnConnectionClose(handler)
  ctx.ProceedAndReportIfStopped(h)  -> ok := ctx.Proceed(h) (false when h stopped
                                       the chain or did not call ctx.Next())

ctx.SetLogoutFunc now takes a handler rather than a function plus loose
persistence arguments, and ctx.Logout() takes none:
  ctx.SetLogoutFunc(func(ctx iris.Context) error { return mgr.Destroy(ctx) })
  err := ctx.Logout()

ctx.ContentLength() now returns Request.ContentLength, which is -1 when the
length is unknown (a chunked request). Any `ContentLength() == 0` check that
meant "empty body" needs rewriting.

PHASE 5: report, do not change.
These v14 defaults are deliberately stricter than v12. Do NOT revert them.
Produce a list of the ones this repository is affected by, with the file and
line, and stop for human confirmation:
  1. Session cookies default to Secure: true and SameSite=Lax. An app served
     over plain HTTP in development will not receive its session cookie until
     sessions.NewOptions(...).Secure is set to false.
  2. JWT tokens are read from the Authorization header only. v12 also read the
     query string. Apps with token-in-URL download links need
     jwt.NewVerifier(...).WithQueryToken().
  3. CORS no longer sends Access-Control-Allow-Credentials alongside a
     wildcard origin. Browsers reject that pair; configure a specific origin.
  4. ctx.AddCookie defaults to SameSite=Lax.
  5. UploadFormFiles rejects more filenames than v12 did, among them reserved
     Windows device names (NUL, COM1, CON.txt) on every platform, and skips a
     rejected file without returning an error.
Also report, without changing: any use of ctx.Addr() in an authorization, rate
limiting or audit path. It returns the client's own unfiltered claim; the
subnet-filtered value is ctx.RemoteAddr().

PHASE 6: verify.
Run `go build ./...`, then `go vet ./...`, then the test suite. Fix what you
broke. Report anything still failing that you could not map, quoting the
compiler error, rather than inventing an API. If a v12 name has no entry
above, say so instead of guessing at a v14 equivalent.

Finally, summarize: files touched, mappings applied, call sites you could not
migrate, and the Phase 5 list awaiting a decision.
````

## Import path

The module major version moved from `v12` to `v14`, so every import path changes:

```go
// before
import "github.com/kataras/iris/v12"
// after
import "github.com/kataras/iris/v14"
```

Update `go.mod` and every source file. A one-shot rewrite:

```sh
find . -name '*.go' -exec sed -i 's|github.com/kataras/iris/v12|github.com/kataras/iris/v14|g' {} +
sed -i 's|github.com/kataras/iris/v12|github.com/kataras/iris/v14|g' go.mod
```

## Package layout

Sub-packages were reorganized in v14. Adjust the moved import paths (the plain
`sed` above rewrites the module prefix but not these relocations):

| v12 import | v14 import |
| --- | --- |
| `.../v12/sessions` | `.../v14/middleware/sessions` |
| `.../v12/cache` | `.../v14/middleware/cache` |
| `.../v12/websocket` | `.../v14/middleware/websocket` |
| `.../v12/i18n` | `.../v14/middleware/i18n` |
| `.../v12/view` | `.../v14/middleware/view` |
| `.../v12/versioning` | `.../v14/middleware/versioning` |
| `.../v12/middleware/recover` | `.../v14/middleware/recovery` |
| `.../v12/middleware/logger` | removed; use `.../v14/middleware/accesslog` |
| `.../v12/mvc` | `.../v14/controller/mvc` |
| `.../v12/macro` | `.../v14/core/router/macro` |
| `.../v12/hero` | removed; see [Dependency injection](#hero-is-replaced-by-the-built-in-di-container) |
| `.../v12/x/client` | `.../v14/client` |
| `.../v12/x/jsonx` | `.../v14/rest/types` |
| `.../v12/x/errors` | `.../v14/rest/errors` |
| `.../v12/x/mathx` | `.../v14/rest/math` |
| `.../v12/x/pagination` | `.../v14/core/pkg/pagination`; `ListOptions` is `PageOptions`, `List[T]` is `Page[T]` and `NewList` is `NewPage`, aliased as `iris.PageOptions`, `iris.Page[T]` and `iris.NewPage`. `NewPage` takes the same four arguments in another order: `NewList(items, totalCount, filter, opts)` becomes `NewPage(opts, filter, items, totalCount)` |
| `.../v12/x/timex` | `.../v14/core/pkg/timex` |
| `.../v12/x/reflex` | `.../v14/core/pkg/reflex` |
| `.../v12/x/sqlx` | removed; use [github.com/kataras/pg](https://github.com/kataras/pg) |
| `.../v12/core/errgroup` | no longer importable (moved to `internal/errgroup`); use the standard `errors.Join` |

The `context` package keeps its path (`.../v14/context`).

Four of the moved packages also changed their entry point, so the import
rewrite alone is not enough:

```go
// caching: v12 cache.Handler(expiration) / cache.Cache(maxAgeFunc)
app.Use(cache.New(cache.DefaultOptions))

// panic recovery: v12 recover.New()
app.UseRouter(recovery.Handler)

// websockets: v12 websocket.Handler(server, idGenerator...)
app.Get("/ws", websocket.New(server, websocket.Options{}))
```

View engine implementations moved out of the framework to
[iris-contrib/views](https://github.com/iris-contrib/views). The `Blocks`
engine was dropped entirely, because
[kataras/blocks](https://github.com/kataras/blocks) already satisfies the new
`view.Engine` interface. See
[RegisterView and ctx.View](#registerview-and-ctxview-are-replaced-by-the-view-middleware)
for the registration change.

## API replacements in v14

The v12 names below are removed; each has one v14 replacement.

### Configuration is replaced by Options

`iris.Configuration`, `iris.WithConfiguration` and the `With*` configurators
are gone. An application is configured with `iris.Options` or with
`iris.Defaults` for an edit-the-defaults style. `(*Options).Bind` loads YAML
from the `SERVER_CONFIG` environment variable first, then from a file.

```go
app := iris.New(iris.Options{Name: "myapp", LogLevel: "debug"})
// or:
app := iris.New(iris.Defaults(func(o *iris.Options) {
    o.LogLevel = "debug"
}))
```

A partial `Options` literal configures only what it sets and fills its unset
(zero) fields from `iris.DefaultOptions`. A non-nil empty slice or map is an
explicit "none" and is kept as such. To force a meaningful zero value (for
example an empty `URLParamSeparator`), set it per request instead:

```go
app.Use(func(ctx iris.Context) {
    ctx.Options().URLParamSeparator = "" // comma values stay a single value
    ctx.Next()
})
```

### Party is replaced by Router and Child

`app.Party` and `PartyFunc` are removed. Route groups are created with
`Router`, which also accepts controllers and constructor functions resolved
by the dependency injection container. `Child` creates a group that inherits
its parent's options and middlewares.

```go
api := app.Router("/api")         // was: app.Party("/api")
api.Router("/users", NewUsersAPI) // nested group; constructor resolved by DI

// was: app.PartyFunc("/admin", func(p iris.Party) { ... })
app.Router("/admin", func(r iris.Router) {
    r.Use(authMiddleware)
    r.Get("/", dashboard)
})
```

The `iris.Party` type itself is now `iris.Router`.

### PartyConfigure is replaced by Router, and Configure by Routing

`PartyConfigure` and the `PartyConfigurator` interface are removed. A struct
registers its own routes by implementing `Routing(r iris.Router)` and passing
itself to `Router`.

```go
// was: func (api *UsersAPI) Configure(router iris.Party)
func (api *UsersAPI) Routing(r iris.Router) {
    r.Get("/{id:uuid}", api.getUser)
}

// was: app.PartyConfigure("/users", &UsersAPI{...})
app.Router("/users", &UsersAPI{UserRepository: repo})
```

### Done, DoneGlobal and UseGlobal are replaced by the Use family

The post-handler hooks `Done` and `DoneGlobal` and the `UseGlobal` method are
removed. Middleware registration is the `Use` family, and the same `Use` call
behaves differently on the root application than on a child router:

| Method | Runs on |
| --- | --- |
| `UseRouter` | before routing: every request, including 404/405 |
| `Use` on the root app | all requests, including 404/405 |
| `Use` on a child router | matched routes of that router only |
| `UseRelevant` on the root app | matched routes only |
| `UseOnce` / `UseRelevantOnce` | as above, replacing an already-registered instance |

A child router's `Use` delegates to `UseRelevant`, so the two are the same
call on anything below the root.

### OnErrorCode is replaced by HandleNoRoute, HandleNoMethod and the errors options

`OnErrorCode` and `OnAnyErrorCode` are removed. The router-level error cases
have dedicated methods on the root application, and API error rendering goes
through the `rest/errors` options (the `Errors` builder step).

```go
app.HandleNoRoute(notFoundHandler)         // was: app.OnErrorCode(404, ...)
app.HandleNoMethod(methodNotAllowedHandler)
app.EnableStatusCodeResponse() // render responses for handler-set codes
```

### RegisterView and ctx.View are replaced by the view middleware

`app.RegisterView` and `ctx.View` are removed; view engines live in the
`middleware/view` package and the engine implementations in
[iris-contrib/views](https://github.com/iris-contrib/views). Register the
engine as middleware and render through the package function:

```go
// was: templates := iris.HTML("./views", ".html"); app.RegisterView(templates)
templates := views.HTML("./views", ".html")
app.Use(view.Handler(templates))

// was: app.Party("/other").RegisterView(otherTemplates)
app.Router("/other", view.Handler(otherTemplates))

// inside a handler, was: ctx.View("index.html", data)
view.Render(ctx, "index.html", "main-layout", data)
```

### iris.Addr, iris.TLS, iris.AutoTLS and Runner are replaced by app.Run with ServerOptions

The `Runner` type and its constructors are removed. Servers are described by
the `iris.ServerOptions` value passed to `app.Run`; `app.Listen(addr)` remains
the plain-HTTP shortcut.

```go
// was: app.Run(iris.TLS(":443", "cert.pem", "key.pem"))
app.Run(iris.ServerOptions{
    Addr:     ":443",
    CertFile: "cert.pem",
    KeyFile:  "key.pem",
})
```

A failed start is reported differently. v12's `Run` logged a build or listen
error and returned it, so a program that called `app.Listen(":8080")` as a
statement exited with status 0 when, say, the port was already taken. v14 logs
the failure at fatal level and exits with status 1. Set
`iris.Options{DisableFatalRun: true}` to have the error returned as before,
for a process that runs more than one server.

v12's `Configuration.Tunneling` and `iris.WithTunneling`, which drove a local
ngrok agent, are replaced by `ServerOptions.Ngrok` (`iris.NgrokOptions`).
The ngrok SDK is linked only into programs that ask for it, so add a blank
import next to the option:

```go
import _ "github.com/kataras/iris/v14/core/server/ngrok"

app.Run(iris.ServerOptions{
    Addr:  ":8080",
    Ngrok: &iris.NgrokOptions{}, // AuthToken empty: read from NGROK_AUTHTOKEN
})
```

Without the import the server refuses to start and the error names the
import to add.

### SetContextErrorHandler is replaced by the context error-handler option

`Application.SetContextErrorHandler` is removed. The handler for render/write
errors is part of the options; a per-request override exists as a middleware
helper.

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{ErrorHandler: myErrorHandler},
})
// per request:
app.Use(iris.WithErrorHandler(myErrorHandler))
```

### HandleDir and Favicon are replaced by the fileserver controller

`HandleDir` and `Favicon` are removed. Static files are served by the
`controller/fileserver` controller, registered like any other controller.

```go
// was: app.HandleDir("/assets", iris.Dir("./public"))
app.Router("/assets", fileserver.New("./public", fileserver.DefaultOptions))
```

### WWW and SubdomainRedirect are replaced by the rewrite middleware

`Application.WWW` and `Application.SubdomainRedirect` are removed. The
`middleware/rewrite` engine covers both, and a lot more:

```go
engine, err := rewrite.New(rewrite.Options{PrimarySubdomain: "www"})
if err != nil {
    panic(err)
}
app.UseRouter(engine.Handler) // was: app.WWW()
```

### Application.Validator and Validate are replaced by a per-context validator

`Application.Validator` and `app.Validate(v)` are removed. The validator is a
context option now, set globally through `iris.Options` or edited per request:

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{RequestValidator: myValidator},
})

// per request:
app.Use(func(ctx iris.Context) {
    ctx.Options().RequestValidator = myValidator
    ctx.Next()
})
```

The validator runs after the request body is decoded, on `ReadJSON`,
`ReadForm`, `ReadQuery`, `ReadValues`, `ReadHeader`, `ReadParams`, `ReadURL`,
`ReadMsgPack` and `UnmarshalBody`.

### hero is replaced by the built-in DI container

The `hero` sub-package is removed. Dependency injection is built into the
application: register dependencies with `RegisterDependency` and resolve
handler functions with `HandlerFunc`. The container type itself lives in
`core/dep` and is reachable through `Router.DI()`.

```go
// was: hero.Register(dependency); hero.Handler(fn)
app.RegisterDependency(NewUserRepository)
app.Get("/users", app.HandlerFunc(func(repo *UserRepository) iris.Handler {
    return func(ctx iris.Context) { /* ... */ }
}))
```

### mvc.GRPC is replaced by controller/mvc.GRPC

The gRPC controller moved with the rest of MVC and became a builder:

```go
// was:
// mvc.New(app).Register(new(myService)).Handle(ctrl, mvc.GRPC{
//     Server: grpcServer, ServiceName: "helloworld.Greeter", Strict: false,
// })
app.RegisterDependency(new(myService))
app.Router("/", mvc.GRPC(ctrl).
    Server(grpcServer).
    ServiceName("helloworld.Greeter").
    Strict(false))
```

### Route registration moves to the route registry

`Router.Dependencies()` is now `Router.DI()`, and the low-level route controls
moved behind `Router.Routes()`, which returns a `*router.RouteRegistry`:

| v12 | v14 |
| --- | --- |
| `router.Dependencies()` | `router.DI()` |
| `router.SetRegisterRule(rule)` | `router.Routes().SetRegisterRule(rule)` |
| `route.Name()` | `route.GetName()` |
| `route.Method()` | `route.GetMethod()` |
| `route.Subdomain()` | `route.GetSubdomain()` |
| `route.Path()` | `route.GetPath()` |
| `route.MainHandlerName()` | `route.GetMainHandlerName()` |
| `route.Property(key)` | `route.GetProperty(key)` |

Router methods that may register more than one route now return the registry
instead of a `[]*Route`. Call its `First()` for the old single-route behavior.
`HandleMany` is deprecated in favor of `Handle`, which accepts multiple paths.

### NewBuilder and the rest packages are the recommended stack

New v14 code is written against `iris.NewBuilder` (a compiler-ordered chain of
CORS, compression, access log, health, error mapping, dependency services and
API mounting) with handlers built on the `rest/errors` and `rest/types`
packages. The old `iris.New` plus manual wiring keeps working; the builder
removes the boilerplate. [TUTORIAL.md](TUTORIAL.md) walks through the whole
stack.

A builder application answers a failure that has no body with the
`rest/errors` JSON envelope: a bare `ctx.StopWithStatus(403)`, a
`ctx.NotFound()`, the 401 of `jwt` or `basicauth`, an unknown path (404) and
a method a known path does not serve (405, when
`Options.Router.FireMethodNotAllowed` is on). A client or test that expected
an empty body or the text `Not Found` reads JSON such as
`{"http_error_code":{"canonical_name":"NOT_FOUND","status":404},...}`
instead. An `iris.New()` application keeps the bare status unless it
registers `errors.Handler` with `UseRouter`, `HandleNoRoute` and
`HandleNoMethod`.

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

### Typed request binding replaces hand-rolled decode blocks

v12 has one shape for reading a request body: `ctx.ReadJSON(&input)` followed
by your own error response and your own validation call. v14 keeps that method
and adds the typed `Bind` family on the Context — Go 1.27 generic methods that
decode, validate (the input type's `Validate() error`, when implemented) and
write the 400 themselves:

```go
input, ok := ctx.BindJSON[CreateUserInput]()
if !ok {
    return // 400 already written.
}
```

Siblings: `ctx.BindJSONOptional[T]()` (empty body is the zero value),
`ctx.BindJSONOneOrMany[T]()`, `ctx.BindParams[S]()`,
`ctx.BindParam[T](name)`, `ctx.BindPagination[F]()`, `ctx.Paginated(collect)`,
`ctx.Count(n, err)` and the typed getters `ctx.ValueAs[T](key)` /
`ctx.RequestValue[T]()`. Prefer these over porting a v12 handler's manual
error branches; the [TUTORIAL](TUTORIAL.md) is written around them. The v12
`x/pagination` types live on as `context.PageOptions`/`context.Page[T]`
(aliased in the root package as `iris.PageOptions`/`iris.Page[T]`), with
`page.Transform[V](fn)` replacing hand-written
page-mapping loops.

### basicauth is typed by the user type

v12's `basicauth.Options.Allow` returned `any`, so a handler read its own
user with `ctx.User().GetRaw().(User)`. v14 makes the middleware generic:
`Options[U]` names the user type once and `basicauth.GetUser[U](ctx)`
returns it without an assertion. `New`, `Default` and `Load` still return a
`context.Handler`, so the mount sites do not change.

```go
// v12
auth := basicauth.New(basicauth.Options{
    Allow: basicauth.AllowUsers(users), // users is a []User
})
u := ctx.User().GetRaw().(User)

// v14
auth := basicauth.New(basicauth.Options[User]{
    Allow: basicauth.AllowUsers(users),
})
u, ok := basicauth.GetUser[User](ctx)
```

`AllowUsers` takes a `[]U`. The `map[string]string` form moved to
`basicauth.AllowUsersMap`, which is what `Default` uses, and `Default`
accepts only that map now. A file list is decoded into your type with
`basicauth.AllowUsersFile[U]` or kept as a `basicauth.Map` by `Load`.
`ctx.User()` and `ctx.Logout()` keep working for every `U`, and the new
`basicauth.AuthorizedAt(ctx)` reports the first login time when `MaxAge` is
set.

### Helpers that changed shape in auth, jwt, hcaptcha and client

Each of these v12 forms has one v14 replacement:

| v12 | v14 |
| --- | --- |
| `auth.Must(auth.New[T](cfg))` | `auth.Must[T](cfg)`, which takes what `New` takes |
| `claims := jwt.Get(ctx).(*Claims)` | `claims, ok := jwt.Get[*Claims](ctx)`; `ok` is false, without a panic, when there are no claims or they are of another type |
| `ctx.Values().Get(hcaptcha.ResponseContextKey)` | `resp, ok := hcaptcha.Get(ctx)`; the key is unexported |
| `client.BindResponse(resp, &v)` | `v, err := client.Bind[T](resp)` |
| `context.IsErrPathCRSFToken(err)` | `context.IsErrPathCSRFToken(err)` |
| `context.SetCookieKVExpiration` | `context.DefaultCookieExpiration` |
| `iris.Patches().Context().SetCookieKVExpiration(d)` | `iris.Patches().Context().SetDefaultCookieExpiration(d)` |

## Context method changes

This is the largest mechanical part of the upgrade. v14 splits request and
response accessors by name so that no method can be read as the wrong one.

### URL query values are read through ctx.Query()

All eighteen `URLParam*` methods are removed. Query values are read through
`ctx.Query()`, which returns a sorted, read-only entries value:

| v12 | v14 |
| --- | --- |
| `ctx.URLParam(k)` | `ctx.Query().Get(k)` |
| `ctx.URLParamDefault(k, def)` | `ctx.Query().Get(k)` plus your own fallback |
| `ctx.URLParamTrim(k)` / `ctx.URLParamEscape(k)` | `ctx.Query().Get(k)` plus `strings.TrimSpace` / `html.EscapeString` |
| `ctx.URLParamExists(k)` | `ctx.Query().Exists(k)` |
| `ctx.URLParamSlice(k)` | `ctx.Query().GetSlice(k)` |
| `ctx.URLParamInt(k)` / `ctx.URLParamIntDefault(k, def)` | `ctx.Query().GetInt(k)` |
| `ctx.URLParamInt32Default(k, def)` | `ctx.Query().GetInt32(k)` |
| `ctx.URLParamInt64(k)` / `ctx.URLParamInt64Default(k, def)` | `ctx.Query().GetInt64(k)` |
| `ctx.URLParamUint64(k)` | `ctx.Query().GetUint64(k)` |
| `ctx.URLParamFloat64(k)` / `ctx.URLParamFloat64Default(k, def)` | `ctx.Query().GetFloat64(k)` |
| `ctx.URLParamBool(k)` / `ctx.URLParamBoolDefault(k, def)` | `ctx.Query().GetBool(k)` |
| `ctx.URLParams()` | `ctx.Query().ToMap()` |
| `ctx.URLParamsSorted()` | `ctx.Query()` (already sorted) |

The `*Default` forms lose their default argument, so check the result
yourself. The numeric getters return `-1` when the key is missing and `0` when
the value is present but not parseable, which is the distinction the old
`URLParamInt` documented:

```go
// was: page := ctx.URLParamIntDefault("page", 1)
page := ctx.Query().GetInt("page")
if page <= 0 {
    page = 1
}
```

### Path parameters are read by type

`ctx.Params().Get(key)` returns `any` in v14, because path parameters are
stored in their macro type. Read strings with `GetString`:

```go
id := ctx.Params().GetString("id")   // was: ctx.Params().Get("id")
n := ctx.Params().GetInt("n")
```

The `{id:uuid}` macro stores a `uuid.UUID` in v14, where v12 stored the text it
had validated. Read it with `ctx.Params().GetUUID("id")` or
`ctx.BindParam[uuid.UUID]("id")`; `GetString("id")` still works and returns the
canonical lowercase hyphenated form whatever spelling the client sent. The same
`GetUUID`, with `GetDuration`, `GetTime` (RFC 3339) and `GetIP`, exists on
`ctx.Query()` and the form accessors, and each returns the zero value for a
missing or unparsable value:

```go
id := ctx.Params().GetUUID("id")      // was: uuid.Parse(ctx.Params().GetString("id"))
since := ctx.Query().GetTime("since") // zero time.Time when absent or not RFC 3339
```

### Headers are split between request and response

In v12, `ctx.GetHeader(name)` read a request header and `ctx.Header(name, value)`
set a response one. v14 gives the request accessors the plain names and
prefixes the response ones with `Set`:

| v12 | v14 | Acts on |
| --- | --- | --- |
| `ctx.GetHeader(k)` | `ctx.Header(k)` | request |
| `ctx.Header(k, v)` | `ctx.SetHeader(k, v)` | response |
| n/a | `ctx.ResponseHeader(k)` | response (new in v14) |

`ctx.Header` changes meaning between the two versions: it is a two-argument
response setter in v12 and a one-argument request reader in v14. The arity
difference makes every stale call site a compile error rather than a silent
misread, so the compiler will find them all for you.

```go
apiKey := ctx.Header("X-API-Key")       // request (what the client sent)
ctx.SetHeader("X-Cache", "HIT")         // response
cached := ctx.ResponseHeader("X-Cache") // response (what this handler set)
```

Content type follows the same split:

| v12 | v14 | Acts on |
| --- | --- | --- |
| `ctx.GetContentTypeRequested()` | `ctx.ContentType()` | request |
| `ctx.ContentType(v)` | `ctx.SetContentType(v, opts...)` | response |
| n/a | `ctx.ResponseContentType()` / `ctx.ResponseMediaType()` | response (new in v14) |

`ctx.MediaType()`, `ctx.Charset()` and `ctx.IsMediaType(...)` are new request-side
helpers.

### Renamed methods

| v12 | v14 |
| --- | --- |
| `ctx.GetCookie(name)` | `ctx.CookieValue(name)` |
| `ctx.GetRequestCookie(name)` | `ctx.Cookie(name)` |
| `ctx.GetBody()` | `ctx.Body()` |
| `ctx.SetCookieKV(name, value, ...)` | `ctx.AddCookie(name, value, ...)` |
| `ctx.UpsertCookie(cookie, ...)` | `ctx.SetCookie(cookie, ...)`, which now replaces a cookie with the same name, path and domain |
| `ctx.ReadHeaders(ptr)` | `ctx.ReadHeader(ptr)` |
| `ctx.RequestContentLength()` | `ctx.ContentLength()` |
| `ctx.ResetResponseWriter(w)` | `ctx.SetResponseWriter(w)` |
| `ctx.ResetRequest(r)` | `ctx.SetRequest(r)` |
| `ctx.RecordRequestBody(true)` | `_ = recorder.RecordRequestBody(ctx)` |
| `ctx.SetMaxRequestBodySize(n)` | `bodylimit.New(n)` middleware, or `ContextOptions.BodyMaxMemory` |
| `ctx.GetReferrer()` | the `middleware/referrer` middleware |
| `ctx.Problem(v, opts)` | `errors.$ErrorCodeName.Problem(ctx, ...)` |
| `ctx.CompressWriter(true)` / `ctx.CompressReader(true)` | the `middleware/compress` middleware |

`ctx.ReadValues(ptr)` is new: it binds the request context's values to a
pointer of struct value.

### Removed methods

These have no replacement and their call sites need rewriting:
`VisitAllCookies`, `ReflectValue`, `Controller`, `RegisterDependency`,
`UnregisterDependency`, `SetFunc`, `GetFunc` and `CallFunc`.

Three more are gone, and each job they did is done another way:

| v12 | v14 |
| --- | --- |
| `ctx.DisablePoolRelease()` | `ctx.Clone()` for a goroutine that keeps using the Context after the handler returns, or `ctx.OnRelease(fn)` to run cleanup before the Context goes back to the pool. The pool now always takes the Context back. |
| `ctx.OnConnectionCloseErr(fn)` | `ctx.OnConnectionClose(handler)`, or your own goroutine that waits on `ctx.Request().Context().Done()` and handles the error itself. |
| `ctx.ProceedAndReportIfStopped(h)` | `ctx.Proceed(h)`, which returns false when `h` stopped the chain or did not call `ctx.Next()`. `ctx.IsStopped()` cannot tell you afterwards, because `Proceed` restores the handler index. If you need to know which, have `h` record it, for example in a variable it closes over. |

### SetLogoutFunc and Logout take a handler instead of loose arguments

v12 stored an arbitrary function plus persistence arguments and invoked it
through `CallFunc`. v14 stores a typed handler, so both signatures change:

```go
// was: ctx.SetLogoutFunc(fn any, persistenceArgs ...any)
//      err := ctx.Logout(args...)
ctx.SetLogoutFunc(func(ctx iris.Context) error {
    return sessionManager.Destroy(ctx)
})
err := ctx.Logout()
```

## Corrected v12 behavior

Five v12 bugs are fixed in v14. None needs a code change, but each changes
what your program observes at runtime.

### ReadBody fills *string and *[]byte targets and returns immediately

`ctx.ReadBody` with a `*string` or `*[]byte` target read the body and then
fell through to the content-type decoders, re-reading an already-drained
stream. The `*[]byte` case used `copy`, which writes nothing into a
zero-length slice, so those targets stayed empty. Raw targets now receive the
body as-is and return; there is no decoder dispatch for them.

### KitchenTime keeps the minutes when scanning a database TIME value

`x/jsonx.KitchenTime.Scan` (now `rest/types.KitchenTime.Scan`) computed the
minutes into a shadowed variable, so every value read from a `TIME` column
landed on the exact hour: `13:45:00` scanned as `1:00 PM`. It now keeps the
minutes. Values that were already truncated by the old behavior cannot be
recovered from the truncated data.

### A UserMap with capitalized keys now resolves

`UserMap` documents each getter as reading either spelling of its key, "the id
or Id or ID value". The check that was meant to allow that used
`unicode.IsTitle`, which matches only the thirty-odd digraph runes of the
Unicode Lt category and is false for every ASCII letter, so it never ran. A map
written as `iris.Map{"Username": "kataras"}` answered `GetUsername` with
`ErrNotSupported`, and auth code branching on that error saw an empty identity
rather than a failure.

The lookup now tries the exact key first and falls back to a case-insensitive
match. Code that worked around the bug by lowercasing its keys is unaffected.

### ParseContentType keeps the case of parameter values

`ParseContentType` lowercased the whole header, and `ContentType.String`
lowercased it again on the way out. Media types and parameter names are
case-insensitive so that is correct for them, but a multipart `boundary` is a
delimiter matched byte-for-byte against the body (RFC 2046 §5.1.1), and
browsers send mixed case. `ctx.ContentType().Boundary` therefore returned a
value that matched nothing, and a boundary passed through
`ctx.SetContentType` was emitted lowercased while the handler wrote the
original delimiters.

The media type, the parameter names and `charset` are still lowercased. Every
other parameter value is now preserved. The parameter map is also capped at
`context.MaxContentTypeParams` (32), since the header is client-supplied and
was otherwise an allocation amplifier, and a `version` parameter that is not a
positive integer is dropped rather than echoed back.

### UniqueHandlers removes every occurrence and leaves its input alone

`UniqueHandlers` is documented as returning "a new slice of Handlers". It did
neither: it deleted in place inside the caller's backing array, and it removed
only the first match per entry, so `UniqueHandlers([a b a], [a])` returned
`[b a]`. It now returns a fresh slice with every match removed and does not
touch the input. `UpsertHandlers` likewise no longer writes into the slice it
is given.

This matters because a route's registered `Handlers` slice is shared by every
request for that route.

## Breaking default changes

### JSON runs on encoding/json/v2 (Go 1.27)

Every `ctx.JSON` and `ctx.ReadJSON` goes through the standard library's
`encoding/json/v2`. Most output is byte-identical (map keys stay sorted,
`time.Duration` stays int64 nanoseconds, a trailing newline follows every
body), with these visible changes:

- `<`, `>` and `&` are no longer escaped by default. The `JSON.UnescapeHTML`
  option is replaced by the inverted opt-in `JSON.EscapeHTML`; JSONP always
  escapes. JSON parsers decode both forms identically.
- nil slices and maps render as `[]` and `{}` instead of `null`. To restore
  v1 output per call:
  `iris.JSON{MarshalOptions: []json.Options{json.FormatNilSliceAsNull(true), json.FormatNilMapAsNull(true)}}`.
- On your own response structs, `omitempty` on a number, boolean or
  `time.Duration` field no longer omits `0`/`false` — that is v2's documented
  semantics change. Replace those with `omitzero` (Iris's structs already did).
- Request bodies containing duplicate object member names or invalid UTF-8
  are rejected with 400. Member-name matching stays case-insensitive by
  default; pass `iris.JSONReader{CaseSensitive: true}` for v2's strict rule.
- `JSONReader.Optimize` (dead since the jsoniter era) is removed.
- Code checking for an empty JSON body should use `iris.IsErrEmptyJSON` (which
  keeps working) or `errors.Is(err, context.ErrEmptyJSON)`; error TEXT
  comparisons on json errors break under Go 1.27 and were never contractual.

### Session cookies are Secure and SameSite=Lax

v12 set the cookie's `Secure` attribute only when `Config.CookieSecureTLS` was
enabled, and never set `SameSite` at all. v14's `sessions.NewOptions` defaults
to `Secure: true` and `SameSite: http.SameSiteLaxMode`. This mitigates CSRF and
stops session cookies from being sent over plaintext HTTP.

Restore the old behavior (for example when serving over plain HTTP in local
development):

```go
opts := sessions.NewOptions(secret)
opts.Secure = false                   // send the cookie over HTTP too
opts.SameSite = http.SameSiteNoneMode // note: browsers require Secure for None
```

A `Secure` cookie still works behind a TLS-terminating proxy, because the browser
connection is HTTPS even when the app sees HTTP.

The `sessions.Database` interface was redesigned around generics and the
request context. Custom v12 stores (`Acquire`, `OnUpdateExpiration`, `Set`,
`Get`, `Decode`, ...) must be rewritten against the three-method
`Database[T]`: `Get`, `Load` and `Commit`. The built-in stores are
`NewCookieDatabase`, `NewRedisDatabase` and `NewPostgresDatabase`.

### JWT tokens are read from the Authorization header only

v12's `jwt.NewVerifier` defaulted its `Extractors` to `{FromHeader, FromQuery}`.
v14 defaults to the `Authorization` header only, because tokens in URLs leak
into access logs, proxy logs, browser history and Referer headers.

Opt back in when you must accept a query token (for example browser downloads
that cannot set a header):

```go
verifier := jwt.NewVerifier(jwt.HS256, key).WithQueryToken()
```

### CORS no longer advertises credentials with a wildcard origin

v12's CORS handler wrote `Access-Control-Allow-Credentials` on every allowed
request, including the ones it answered with `Access-Control-Allow-Origin: *`.
Browsers reject that combination, and it is always a misconfiguration. v14
sends credentials with a wildcard origin only when you enable them explicitly.
Credentialed CORS continues to work unchanged when the response carries a
specific origin, which is the normal case.

If you relied on wildcard-plus-credentials, configure a specific allowed origin
instead.

### SetCookie replaces a cookie with the same name, path and domain

v12's `ctx.SetCookie` appended a `Set-Cookie` line every time, and
`ctx.UpsertCookie` replaced one by name. v14 has no `UpsertCookie`:
`SetCookie`, `AddCookie` and `RemoveCookie` replace a line that has the same
name, path and domain (the domain compared without case and without a
leading dot), so two calls for one cookie send one line. A cookie with the
same name and another path or domain is still sent as its own line.

### AddCookie defaults to SameSite=Lax

v12 set no `SameSite` attribute. `Context.AddCookie` now sets `SameSite=Lax` by
default. Override per call:

```go
ctx.AddCookie(name, value, iris.CookieSameSite(http.SameSiteStrictMode))
```

## Recommended opt-in hardening

These v14 capabilities are off by default and have no v12 equivalent. Enabling
them is recommended for production.

- **Request body limit.** Set `ContextOptions.BodyMaxMemory` (bytes) to cap
  `Body`, `ReadJSON`, `ReadJSONStream` and `ReadMultipartRelated`; it also
  bounds decompression-bomb reads. `0` (default) means unlimited. For per-route
  limits use the `bodylimit` middleware.
- **Session fixation defense.** Call `session.RegenerateID(ctx)` after a
  privilege change (most importantly right after sign-in).
- **Static file dotfile protection.** Pass `fileserver.BlockHidden` as
  `Options.AssetValidator` to stop serving `.git`, `.env` and other dotfiles.
- **Request ID forgery.** Use `requestid.ServerOnlyGenerator` to ignore a
  client-supplied `X-Request-ID` header.
- **Password hashing cost.** Raise `password.Cost` (`auth/identity`) above
  `bcrypt.DefaultCost` for stronger hashing.
