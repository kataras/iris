# Middleware Reference

All middleware packages are under `github.com/kataras/iris/v14/middleware/`.

## Constructor Convention

One shape across every package:

- With settings: an `Options` struct whose zero value is valid, and `New(opts Options) context.Handler`.
- Without settings: a plain `Handler`, no constructor.
- Can fail: `New(...) (T, error)` plus a `Must(...) T` companion that panics (`rewrite.Must`, `httpcache.Must`).
- Anything else (a builder, a pair of handlers, a `*Limiter`): stated in the first paragraph of the package doc.

There are no functional options. `rate.PurgeEvery(...)`, `methodoverride.HeadersOnly()`, `geolocation.APIKey(...)`, `hcaptcha.WithSiteKey(...)` and their siblings are gone; each is a field on the package's `Options` now.

## Global vs Route Middleware

```go
// Global: runs for ALL requests (including 404s, static files)
app.UseRouter(handler)
// Via builder:
.RouterMiddlewares(handler)

// Route-level: runs only for matched routes
app.Use(handler)
// Via builder:
.Middlewares(handler)
```

## Converting net/http Middleware: iris.FromStd

```go
// Convert native net/http handlers and middleware to iris.Handler.
// Supported forms:
//   iris.FromStd(h http.Handler)
//   iris.FromStd(func(w http.ResponseWriter, r *http.Request))
//   iris.FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
app.Use(iris.FromStd(stdMiddleware))
```

In all three forms the standard handler reads route parameters with `r.PathValue(name)`, and `r.Pattern` (when empty) is set to the matched route's template as registered, e.g. `/users/{id:uuid}`. `r.Pattern` stays empty in the pre-routing `UseRouter` slot (no route matched yet) and in the built-in 404 and 405 routes, so register route-labelling middleware with `Use`. `context.TryFromStd` is the non-panicking form.

## Conditional Middleware: iris.NewConditionalHandler

```go
// Runs the given handlers only when the filter returns true.
// Filter type: func(iris.Context) bool. Handlers must call ctx.Next() to proceed.
app.Use(iris.NewConditionalHandler(func(ctx iris.Context) bool {
    return ctx.Header("X-Debug") == "1"
}, debugHeadersMiddleware, verboseLogMiddleware))
```

---

## accesslog

Structured HTTP request/response logging with configurable formatters.

```go
import "github.com/kataras/iris/v14/middleware/accesslog"

ac := accesslog.File("access_log.json") // also FileUnbuffered(path), New(io.Writer)
ac.AddOutput(os.Stdout)
ac.SetFormatter(&accesslog.JSON{Indent: "  ", HumanTime: true})
// Other formatters: &accesslog.CSV{}, &accesslog.Template{Text: "{{.Code}}"}
ac.IP = true
ac.RequestBody = true
ac.ResponseBody = true
app.UseRouter(ac.Handler)
defer ac.Close() // or CloseContext(ctx) to bound the flush
```

Production tuning:

```go
ac.Async = true            // never block the response on disk I/O
ac.RequestBody = false     // bodies hold credentials and PII
ac.ResponseBody = false
ac.BytesReceivedBody = true // sizes without contents (already the default)
ac.BytesSentBody = true
ac.KeepMultiLineError = false // collapse errors to one line for log shippers
ac.OnlyFailures = true      // an errors-only log
ac.PanicLog = accesslog.LogHandler
```

`accesslog.New()` already defaults `BytesReceivedBody`, `BytesSentBody` and `KeepMultiLineError` to true and `PanicLog` to `LogHandler`, so only `KeepMultiLineError = false` and `Async = true` are real changes above.

`ResponseBody` is left empty for a response that carries a `Content-Encoding` other than `identity`, and `Redact` is not called for that line: registered outside compression (the builder's order, so a panic is still logged), the recorder would hold compressed bytes. The body is not decompressed for the log; status, byte counts and the other fields are logged as usual.

Custom fields, at registration and from a handler:

```go
ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
    f.Set("request_id", requestid.Get(ctx))
})
accesslog.GetFields(ctx).Set("tenant", tenantOf(ctx)) // inside a handler
```

Skipping and live tail:

```go
app.Get("/healthz", accesslog.SkipHandler, healthz) // as route middleware
accesslog.Skip(ctx)                                 // conditionally, in a middleware

broker := ac.Broker() // before the server starts
ln := broker.NewListener()
defer broker.CloseListener(ln)
for log := range ln { /* log.Code, log.Path, ... */ }
```

The JSON record carries `timestamp` (unix ms), `latency`, `code`, `method`, `path`, `ip`, `query`, `params`, `fields`, `request`, `response`, `bytes_received`, `bytes_sent`. `TruncateFile(0)` pairs with an external rotator. The builder registers a configured instance through `LogRequestsWith(ac)`. See [observability.md](./observability.md).

## apikey

Shared-secret authentication for callers that are programs, not people: other services, cron jobs, webhook senders, CLI tools.

```go
import "github.com/kataras/iris/v14/middleware/apikey"

guard := apikey.New(apikey.Options{
    Keys: map[string]string{
        "reporting": os.Getenv("REPORTING_API_KEY"), // principal -> secret
    },
})

api := app.Router("/api", guard)

// In a handler:
principal, ok := apikey.Get(ctx) // "reporting", true
```

| Option | Default | Notes |
|--------|---------|-------|
| `Header` | `"Authorization"` | The `Authorization` header is redacted by default in proxy and log-scrubber configurations; a custom one is not |
| `Scheme` | `"Bearer"` | Matched case-insensitively per RFC 7235. `apikey.NoScheme` (`"-"`) accepts a bare value. An **empty** `Scheme` keeps the Bearer default: an unset field must not turn a check off |
| `Query` | `""` (off) | Reads the key from a query parameter when the header is absent. `middleware/accesslog` records every query argument and its `Redact` hook covers bodies only, so mask it in a custom formatter (`ac.SetFormatter`) first |
| `Keys` | `nil` | Principal name to secret, or to `Hash(secret)`. One entry per consumer: two principals sharing one secret authenticate each other's requests |
| `Hash` | `nil` | Applied to the presented key before comparison, so `Keys` hold digests. Use `apikey.SHA256` (hex): a shared secret is high-entropy random, not a password |
| `Lookup` | `nil` | Resolves a key none of `Keys` matched, for keys in a database. An error is a **500**, not a 401: a store that is down must not read as an invalid key. With `Hash` set it receives `Hash(key)` |
| `AllowAnonymous` | `false` | Development only. Ignored as soon as a key or a `Lookup` exists |
| `ErrorHandler` | `nil` | Default renders the `rest/errors` `Unauthenticated` envelope |

Behaviour worth knowing:

- **Default-deny.** No `Keys` and no `Lookup` rejects every request, with a warning logged on the first one.
- Two `Authorization` headers are a 401: two credentials on one request is not a valid presentation of either.
- The comparison runs over every configured key with no early exit and no branch on the result; both sides are reduced to a 32-byte SHA-256 digest first, so neither which key matched nor how long the keys are leaks through timing.
- On success, when nothing else has set a user, a `context.SimpleUser` carrying the principal is stored, so `ctx.User()` works.
- The rejection reasons are `apikey.ErrMissing` (no key, or a value the scheme does not fit) and `apikey.ErrInvalid` (a key that matched nothing), wrapped for an `ErrorHandler` to tell apart with `errors.Is`.

## basicauth

HTTP Basic Authentication, generic over the user type: `Options[U]` names it, `Allow` returns it and `basicauth.GetUser[U](ctx)` hands it back to the handlers typed. `ctx.User()` sees the same value as a `context.User`.

```go
import "github.com/kataras/iris/v14/middleware/basicauth"

// Quick setup with a map; the user is a *iris.SimpleUser.
auth := basicauth.Default(map[string]string{"admin": "password"})
app.Use(auth)

// Typed setup: operators is a []Operator with Username and Password fields
// (json tags count, embedded structs too), or a type implementing
// basicauth.User, or anything with a basicauth.Credentials option.
auth := basicauth.New(basicauth.Options[Operator]{
    Realm: "My API",
    Allow: basicauth.AllowUsers(operators),
    // From file, with bcrypt-hashed passwords, decoded into Operator:
    // Allow: basicauth.AllowUsersFile[Operator]("users.yml", basicauth.BCRYPT),
    // Your own lookup, e.g. a database:
    // Allow: func(ctx iris.Context, username, password string) (Operator, bool) { ... },
    ErrorHandler: basicauth.DefaultErrorHandler,
    MaxAge:    5 * time.Minute,
    HTTPSOnly: true, // refuse to accept credentials over plaintext
    MaxTries:  3,    // 403 after three failed attempts, counted server-side per client and username
})

app.Get("/me", auth, func(ctx iris.Context) {
    op, ok := basicauth.GetUser[Operator](ctx) // typed, no assertion
    since, ok := basicauth.AuthorizedAt(ctx)   // first login time; false without MaxAge
    _ = ctx.Logout()                           // forget the login, 401 for the client
})

// Shortcut: load users from a JSON or YAML file as basicauth.Map, returns the handler directly.
auth := basicauth.Load("users.yml")
```

Behaviour worth knowing:

- `basicauth.AllowUsersMap(map[string]string{...})` is the `Allow` behind `Default`; `Default` accepts only that map form.
- Passwords are compared in constant time and an unknown username is compared against a decoy (a real bcrypt hash under `BCRYPT`), so the response time does not enumerate users.
- Accepted credentials are kept in memory as a SHA-256 digest of `username:password`, never in cleartext; `GC{Every: ...}` prunes expired entries.
- A custom `ErrorHandler` replaces `DefaultErrorHandler`, which is what sets `WWW-Authenticate`; set the header yourself or browsers stop prompting. The error values name the user, never the password.

## bodylimit

Request body size limits to prevent abuse.

```go
import "github.com/kataras/iris/v14/middleware/bodylimit"

app.Use(bodylimit.New(2 * iris.MB))   // 2 MB limit
app.Use(bodylimit.New(512 * iris.KB)) // 512 KB limit
// Returns 413 on violation: a declared Content-Length above the limit before
// the handler runs, a streamed body as soon as the read passes the limit.
// Under errors.Handler both answer TOO_LARGE, "request body too large".
```

## cache

HTTP caching headers and conditional request handling.

```go
import "github.com/kataras/iris/v14/middleware/cache"

app.Use(cache.New(cache.DefaultOptions)) // public, max-age=3600, s-maxage=7200, ETag, 24h Expires

app.Use(cache.New(cache.Options{
    ETag:              true,
    CacheControl:      "public, max-age=3600",
    LastModified:      true,
    Expires:           24 * time.Hour,
    Pragma:            cache.NoPragmaCache, // "no-cache"
    Vary:              cache.NewVaryBuilder().AcceptEncoding().Build(),
    ProxyAuthenticate: cache.NewProxyAuthenticateBuilder().Basic("restricted").Build(),
}))

// Builders return plain strings; hoist them to startup.
cc := cache.NewCacheControlBuilder(). // Public, Private, NoCache, NoStore, MaxAge,
    Public().MaxAge(time.Hour).       // time.Duration, written in whole seconds; also SMaxAge, MustRevalidate, ProxyRevalidate
    MustRevalidate().
    Build()
```

Caveat: this middleware derives validators from the clock, not your payload (the `ETag` is the current Unix second, `Last-Modified` is hardcoded to 24h ago). It is a header-policy and revalidation helper, not content validation. For data-derived 304s and for server-side response storage (`middleware/cache/httpcache`), see [caching.md](./caching.md).

## compress

Response compression: gzip, deflate, brotli, snappy, s2 and zstd.

```go
import "github.com/kataras/iris/v14/middleware/compress"

app.UseRouter(compress.Handler)
// Automatically detects client support via Accept-Encoding
// The builder registers this automatically when Compression(true)
```

It also **transparently decompresses request bodies** carrying `Content-Encoding`, with no ceiling of its own. Bound it, or a one-megabyte upload becomes a gigabyte in memory:

```go
app.Use(compress.Safe)                 // 32MB, compress.DefaultMaxDecompressedSize
app.Use(compress.Limited(8 * iris.MB)) // or your own
```

Reading past the cap yields `compress.ErrDecompressedBodyTooLarge`. `ContextOptions.BodyMaxMemory` and a `bodylimit` registered *after* this middleware bound the same stream, because the body being read by then is the decompressed one. `compress.Handler` logs a warning at first use when nothing bounds it.

Compression trades CPU for bytes. In front of already-compressed payloads (images, video) it is a loss; register it on the groups that serve text rather than at the root.

## cors

Cross-Origin Resource Sharing.

```go
import "github.com/kataras/iris/v14/middleware/cors"

app.UseRouter(cors.New().
    // One comma-separated line. For a list use AllowOrigins("https://a.com", "https://*.b.com").
    AllowOrigin("https://example.com,https://app.example.com").
    AllowHeaders("Content-Type", "Authorization").
    ExposeHeaders("X-Total-Count").
    MaxAge(12 * time.Hour).
    Handler())

// Allow all origins
cors.New().AllowOrigin("*").Handler()

// Custom validation (origin only)
cors.New().AllowOriginMatcherFunc(func(origin string) bool {
    return strings.HasSuffix(origin, ".example.com")
}).Handler()
// Context-aware variant: AllowOriginFunc(func(ctx iris.Context, origin string) bool { ... })
// Regex variant: AllowOriginRegex(`^https://.*\.example\.com$`)

// With error handler (used by the builder, rest/errors integration)
cors.New().HandleErrorFunc(errors.FailedPrecondition.Err).AllowOrigin("*").Handler()
```

There is no `AllowMethods` method: a preflight's `Access-Control-Allow-Methods` is a fixed list of every standard method (`GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD, CONNECT, TRACE, QUERY`), and route registration controls which verbs exist. Other chain methods: `AllowCredentials()`, `DisallowCredentials()`, `ReferrerPolicy(cors.NoReferrerWhenDowngrade)`, `ExtractOriginFunc(cors.StrictOriginExtractor)`, `SkipLocalRequests(true)`.

Under the builder, `AllowOrigin(line)` is the shorthand for `Cors(cors.New().HandleErrorFunc(errors.FailedPrecondition.Err).AllowOrigin(line))`. Use the `Cors` step at the same position when you need anything the origin line cannot express; the value is taken as given, so set `HandleErrorFunc` yourself or rejections render as a bare 403. `Cors(nil)` installs nothing, the same as `AllowOrigin("none")`.

```go
iris.NewBuilder().
    Prefix("/api").
    Cors(cors.New().
        HandleErrorFunc(errors.FailedPrecondition.Err).
        AllowOrigin("*").
        AllowCredentials()).
    Compression(true)
```

On credentials: browsers reject `Access-Control-Allow-Credentials` alongside `Access-Control-Allow-Origin: *`, so the middleware suppresses that header for a wildcard origin unless `AllowCredentials()` was called. A request carrying an `Origin` gets that origin echoed back rather than `*`, which is every browser request, so credentials are advertised normally there; the wildcard case is a request with no `Origin` header at all.

## counter

Request counter metrics.

```go
import "github.com/kataras/iris/v14/middleware/counter"
```

## geolocation

IP-based geolocation.

```go
import "github.com/kataras/iris/v14/middleware/geolocation"
```

## grpc

gRPC integration.

```go
import "github.com/kataras/iris/v14/middleware/grpc"
```

## hcaptcha

hCaptcha verification.

```go
import "github.com/kataras/iris/v14/middleware/hcaptcha"
```

## httpcost

Per-request resource usage (time, memory, CPU) reduced to one cost number, for performance analysis or usage billing.

```go
import "github.com/kataras/iris/v14/middleware/httpcost"

app.Use(httpcost.New(httpcost.DefaultOptions)) // 0.01/sec, 0.001/MB, 0.02/CPU
app.Use(httpcost.New(httpcost.Options{CostPerSecond: 0.02, CostPerMB: 0.002, CostPerCPU: 0.04}))
```

`Metric` carries `StartTime` (a `types.ISO8601`, not a `time.Time`), `Duration`, `MemoryUsed`, `NetworkUsed`, `CpuTime` and `Cost`.

The metric is written **after** the chain returns, so `Get` inside the handler finds nothing. Read it from an outer middleware that calls `ctx.Next()` first:

```go
app.Use(func(ctx iris.Context) {
    ctx.Next()
    if m, ok := httpcost.Get(ctx); ok {
        ctx.Application().Logger().Infof("%s cost=%.4f", ctx.Path(), m.Cost)
    }
})
```

`httpcost.Compute(func())` applies the same formula to an arbitrary block. See [observability.md](./observability.md).

## i18n

Internationalization and localization with multiple detection strategies.

```go
import "github.com/kataras/iris/v14/middleware/i18n"

// Load translations
locales := i18n.New()
locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
locales.SetDefault("en-US")

// Middleware with language detection
app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
    URLParameter: "lang",   // ?lang=el-GR
    Cookie:       "lang",   // Cookie-based
    Subdomain:    true,     // en.example.com
}))

// Path-based routing (/en-US/, /el-GR/)
app.UseRouter(i18n.NewRouter(locales, i18n.RouterOptions{
    Cookie: "lang",
}))

// In handler
locale := i18n.GetLocale(ctx)
msg := locale.GetMessage("welcome", "Iris")
```

Translation file (YAML):
```yaml
welcome: "Welcome, %s!"
FreeDay:
  one: "You have a day off"
  other: "You have %[1]d free days"
```

## ipaccess

IP allowlist and blocklist. Patterns are exact IPs, CIDR ranges or regular expressions.

```go
import "github.com/kataras/iris/v14/middleware/ipaccess"

// New returns (context.Handler, error); Must takes the same patterns and panics on a bad one instead.
handler := ipaccess.Must(
    []string{"10.0.0.0/8", "203.0.113.7"}, // allowed
    []string{`^198\.51\.100\.`},           // blocked
)
app.Router("/admin", handler, adminAPI)
```

Blocked patterns are checked first and win. A request matching neither list gets 403.

It matches `ctx.Addr()`. When `ContextOptions.RemoteAddrHeaders` is unset - the default - that is the connection address and a forged `X-Forwarded-For` cannot defeat it. **If you do configure `RemoteAddrHeaders`, this middleware reads them**, so the proxy in front must overwrite those headers on every request; otherwise a client sets its own address and walks through the allowlist. Behind a reverse proxy with no forwarding configured, every request appears to come from the proxy: allowlist the proxy, or filter there.

Regexp patterns are anchored at the start, so `10\.0\.0\.\d{1,3}` cannot be satisfied by `110.0.0.99`. The end is left open on purpose, so a prefix pattern like `^198\.51\.100\.` still covers the whole /24; add `$` yourself for an exact match.

An empty allowed list means "no allowlist": `ipaccess.New(nil, blocked)` is a blocklist-only policy rather than a deny-all.

## jwt

JSON Web Token authentication with signing, verification, and blocklisting.

```go
import "github.com/kataras/iris/v14/middleware/jwt"

// Signer
signer := jwt.NewSigner(jwt.HS256, []byte("secret"), 15*time.Minute)

// Verifier. As of v14 the default extractor list reads the Authorization
// header ONLY: tokens in a URL leak into access logs, proxy logs, browser
// history and Referer headers.
verifier := jwt.NewVerifier(jwt.HS256, []byte("secret"))
verifier.WithDefaultBlocklist() // Enable token revocation
// verifier.WithQueryToken()    // opt back into ?token= (chainable)

// Validators passed here run on every Verify, blocklist or not. Pin the
// issuer when several services share a signing key, or a token minted by
// one of them is accepted by all of them.
verifier = jwt.NewVerifier(jwt.HS256, []byte("secret"),
    jwt.Expected{Issuer: "my-app"})

// Custom claims
type UserClaims struct {
    jwt.Claims
    Username string `json:"username"`
    Role     string `json:"role"`
}

// Sign token
token, err := signer.Sign(&UserClaims{
    Claims:   jwt.Claims{Subject: "user-123"},
    Username: "john",
    Role:     "admin",
})

// Verify middleware
app.Use(verifier.Verify(func() any {
    return new(UserClaims)
}))

// In handler: T is what the constructor given to Verify returns.
claims, ok := jwt.Get[*UserClaims](ctx) // false, no panic, when absent or another type
```

A rejected token answers 401 with the reason stored as a private error, so under `errors.Handler` the body reads `"message":"Unauthorized"` whatever the error options map: the client cannot read from it why the token was refused.

Algorithms: `HS256`, `HS384`, `HS512`, `RS256`, `ES256`, and more.

Prefer a short-lived access token with a refresh companion over one long-lived token, so revocation has a bounded window:

```go
pair, err := signer.NewTokenPair(accessClaims, refreshClaims, 24*time.Hour)
```

## methodoverride

HTTP method override via header or form field.

```go
import "github.com/kataras/iris/v14/middleware/methodoverride"

app.UseRouter(methodoverride.New(methodoverride.Options{}))
```

By default the override is read from the `X-HTTP-Method`, `X-HTTP-Method-Override` and `X-Method-Override` headers **and** from `_method` in the query string and form body. Only `POST` is overridable, and the resulting method must be a standard HTTP method (`QUERY` counts as one).

The query and form getters are the ones to think about: a cross-site HTML form can set them and needs no preflight, so `POST /account?_method=DELETE` reaches a `DELETE`-only route. `HeadersOnly` drops them, leaving only the headers, which a cross-site form cannot set:

```go
app.UseRouter(methodoverride.New(methodoverride.Options{HeadersOnly: true}))
```

`Options` fields: `Methods`, `Headers`, `HeadersOnly`, `FormField`, `Query`, `Getters`, `SaveOriginalMethodContextKey`. The zero value reproduces the defaults, and `Methods` replaces the overridable-method list rather than appending to it.

## modrevision

Build info and health check endpoint.

```go
import "github.com/kataras/iris/v14/middleware/modrevision"

app.Get("/health", modrevision.New(modrevision.Options{
    ServerName:     "My API",
    Env:            "production",
    Developer:      "team@company.com",
    CloudDetection: true,
}))
```

## monitor

A live process and host dashboard plus a JSON stats endpoint.

```go
import "github.com/kataras/iris/v14/middleware/monitor"

m := monitor.New(monitor.Options{
    PID:                   0,               // 0 means the current process
    RefreshInterval:       2 * time.Second, // the default; see below
    ViewRefreshInterval:   2 * time.Second,
    ViewAnimationInterval: 500 * time.Millisecond,
    ViewTitle:             "Service Monitor",
})
iris.RegisterOnInterrupt(m.Stop)

mon := app.Router("/monitor", basicauth.New(adminOpts))
mon.Get("/", m.View)   // HTML dashboard
mon.Post("/", m.Stats) // the JSON it polls
```

Two requirements that are easy to miss:

- `View` and `Stats` must be on the **same path**, differing only by method: the dashboard polls its own URL with POST.
- `RefreshInterval` defaults to `monitor.DefaultRefreshInterval` (2 seconds) when unset. Pass your own value to change the collector's tick.

Read the numbers directly with `m.Holder.GetStats()`. `monitor.Stats` has `PIDCPU`, `PIDRAM`, `PIDConns`, `OSCPU`, `OSRAM`, `OSTotalRAM`, `OSLoadAvg`, `OSConns`, also published as `expvar` gauges (`pid_cpu`, `os_ram`, …). Gate it behind auth: it discloses host capacity and load.

## pprof

Go profiling endpoints. `New()` returns a routing controller, so mount it as one.

```go
import "github.com/kataras/iris/v14/middleware/pprof"

app.Router("/debug/pprof", basicauth.New(adminOpts), pprof.New())
```

Serves heap, goroutine, allocs, block, mutex, threadcreate, cmdline, profile and trace (`?seconds=N` on trace and profile). **Never expose publicly**: profiles reveal source paths and memory contents, and a long CPU profile is itself a load generator.

## rate

Token bucket rate limiting.

```go
import "github.com/kataras/iris/v14/middleware/rate"

// 1 request per second, burst of 5
app.Use(rate.Limit(1, 5, rate.Options{}))

// 1 request per minute, burst of 3, via the Every helper
app.Use(rate.Limit(rate.Every(time.Minute), 3, rate.Options{}))

// With periodic cleanup of old clients
app.Use(rate.Limit(1, 5, rate.Options{
    PurgeEvery:       5 * time.Second,
    PurgeMaxLifetime: 5 * time.Minute,
}))

// Custom exceeded handler
app.Use(rate.Limit(1, 5, rate.Options{
    ExceedHandler: func(ctx iris.Context) {
        errors.ResourceExhausted.Message(ctx, "rate limit exceeded")
    },
}))

// Custom client identifier (default is the remote IP):
// call SetIdentifier from a middleware registered BEFORE the limiter.
app.Use(func(ctx iris.Context) {
    rate.SetIdentifier(ctx, ctx.Header("X-API-Key"))
    ctx.Next()
})
app.Use(rate.Limit(1, 5, rate.Options{}))
```

`rate.Options` fields: `ExceedHandler`, `ClientData`, `PurgeEvery`, `PurgeMaxLifetime`. The zero value is valid, and `PurgeEvery` and `PurgeMaxLifetime` are independent: setting one alone leaves the other at its default.

Clients are keyed by remote IP unless `SetIdentifier` says otherwise. Builder apps already render a JSON 429 under `RESOURCE_EXHAUSTED` with a retry hint, so `ExceedHandler` is only needed to change that shape. `rate.Get(ctx).RetryDelay()` reports the retry delay. Read it through that accessor rather than the `Delay` field: a `Client` is shared by every concurrent request from the same identifier, and the field is written under its mutex.

Layer a tighter budget on expensive routes rather than lowering the global limit:

```go
app.Use(rate.Limit(rate.Every(time.Second), 20, rate.Options{})) // global
app.Post("/signin", rate.Limit(rate.Every(10*time.Second), 3, rate.Options{}), signin)
```

## recaptcha

Google reCAPTCHA verification.

```go
import "github.com/kataras/iris/v14/middleware/recaptcha"
```

## recorder

Makes the request body re-readable by replacing it with a rewinding buffer. Needed whenever two consumers must both read the body (an access log plus your decoder, for example).

```go
import "github.com/kataras/iris/v14/middleware/recorder"

app.Use(recorder.Request) // middleware form: record every request body

// Or on demand, inside a handler or another middleware:
_ = recorder.RecordRequestBody(ctx)  // returns *BufferCloser, idempotent
raw := recorder.ReadRequestBody(ctx) // []byte, body stays readable
if recorder.IsRequestBodyRecorded(ctx) { /* ... */ }
```

`Context.RecordRequestBody` no longer exists; `recorder.RecordRequestBody(ctx)` replaces it.

## recovery

Panic recovery with stack traces and error logging.

```go
import "github.com/kataras/iris/v14/middleware/recovery"

// Basic usage
app.UseRouter(recovery.Handler)

// With custom error handler
app.UseRouter(
    recovery.WithErrorHandler(func(ctx iris.Context, err error) {
        errors.Internal.LogErr(ctx, err) // logs the dump, sends INTERNAL
    }),
    recovery.Handler,
)

// Builder automatically registers: recovery.WithErrorHandler(errors.Internal.LogErr)
```

`ErrPanicRecovery` struct contains: `Cause`, `Callers` (file:line callers), `Stack` (debug stack), `RegisteredHandlers`, `CurrentHandlerName`, `CurrentHandlerFileLine`, `Request`.

`Request` is a `httputil.DumpRequest` of the failed request with the credential-bearing headers scrubbed to `[redacted]`: `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`, `X-Csrf-Token`, matched case-insensitively. Change the list through `recovery.RedactedRequestHeaders`. `Error()` returns the cause and the callers only; `LogMessage()` returns the full form including the dump. Prefer `Error()` in anything that reaches a client, since `ErrPanicRecovery` is otherwise easy to format into a response wholesale.

## referrer

Referrer tracking and parsing.

```go
import "github.com/kataras/iris/v14/middleware/referrer"
```

## requestid

Unique request ID generation.

```go
import "github.com/kataras/iris/v14/middleware/requestid"

app.Use(requestid.New())
// Generates X-Request-Id header for each request

id := requestid.Get(ctx)
```

The default generator **honors a client-supplied `X-Request-ID`**, which is handy for tracing across services you control and a log-forgery vector from the open internet. On a public edge, generate server-side only:

```go
app.UseRouter(requestid.New(requestid.ServerOnlyGenerator))
```

Register it before the access log so a field setter can read it.

## rewrite

URL rewriting rules.

```go
import "github.com/kataras/iris/v14/middleware/rewrite"

engine, err := rewrite.New(rewrite.Options{
    RedirectMatch: []string{ // "CODE PATTERN TARGET"
        "301 /old-path/(.*) /new-path/$1",
        "301 /seo/(.*) /$1",
    },
    PrimarySubdomain: "www", // redirect root domain to www.
})
// handle err, then:
app.UseRouter(engine.Handler)

// Or rewrite.Must for the same options, panicking on a parser error:
app.UseRouter(rewrite.Must(rewrite.Options{
    RedirectMatch: []string{"301 /seo/(.*) /$1"},
}).Handler)

// Or load rules from a YAML/JSON file (panics on error):
app.UseRouter(rewrite.Load("redirects.yml"))
```

## servertiming

Exposes phase timings to browser devtools through the `Server-Timing` header.

```go
import "github.com/kataras/iris/v14/middleware/servertiming"

app.Use(servertiming.Handler) // records the total request as the "app" metric

func handler(ctx iris.Context) {
    servertiming.Start(ctx, "db", "query rows")
    rows, err := repo.Rows(ctx)
    servertiming.End(ctx, "db") // returns false if "db" was never started
    // ...
}
```

Every `Start` needs its `End` or the metric never lands. The header is written before the body, so timings recorded after the first write are lost. Metric names reach the browser: keep internal hostnames and query text out of them. `WriteMetrics(ctx, metrics)` writes a pre-built set.

## sessions

Server-side session management with generic, typed session data and multiple backends (cookie, PostgreSQL, Redis).

```go
import "github.com/kataras/iris/v14/middleware/sessions"

// Typed session data: any struct works.
type UserSession struct {
    ID       string
    Username string
}

// Keys encrypt and sign the cookie payload
// (64-byte hash key, 32-byte block key recommended).
sessionOptions := sessions.NewOptions(hashKey, blockKey) // sessions.Options (a value)
sessionOptions.MaxAge = 1 * time.Hour                    // default is 24 hours
// Other Options fields: Path, Domain, SameSite, Partitioned, Secure, DataCodecs.

// Cookie-based backend: data lives inside the encrypted cookie.
sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)

// Middleware: loads the session per request and auto-commits at the end.
app.Use(sessions.Handler(sessionDB, "my_session"))
// Read-only variant (no auto-commit):
// app.Use(sessions.ReadOnlyHandler(sessionDB, "my_session"))

app.Get("/set", func(ctx iris.Context) {
    session := sessions.Get[UserSession](ctx)
    session.Set(UserSession{ID: "1", Username: "kataras"}) // or session.Data = UserSession{...}
    // Auto-committed by the Handler middleware.
})

app.Get("/get", func(ctx iris.Context) {
    session := sessions.Get[UserSession](ctx)
    ctx.JSON(session.Data)
})
```

Without the middleware, query the database directly. `sessions.Single` fixes the cookie name so `Get` takes only the context:

```go
sessionDB := sessions.Single(
    sessions.NewCookieDatabase[UserSession](sessionOptions),
    "my_session",
)

app.Get("/profile", func(ctx iris.Context) {
    session, err := sessionDB.Get(ctx)
    if errors.HandleError(ctx, err) {
        return
    }

    session.Set(UserSession{ID: "1", Username: "kataras"})
    // Commit persists the session and writes the cookie.
    if err := session.Commit(ctx); errors.HandleError(ctx, err) {
        return
    }
})
```

Flash messages (one-time reads, removed after `GetFlash`):

```go
session.SetFlash("notice", "Login successful")
message, ok := session.GetFlash("notice")
// Commit after SetFlash when not using the Handler middleware.
```

Other session methods: `session.Delete(ctx)` expires it, `session.RegenerateID(ctx)` rotates the server-side ID after sign-in (no-op for cookie databases), `session.IsNew`.

Database backends:

```go
// PostgreSQL (github.com/jackc/pgx/v5)
pgDB := sessions.NewPostgresDatabase[UserSession](sessions.DefaultTableName, conn, sessionOptions)
pgDB.ShiftExpiration(true) // optional: slide expiration to NOW+MaxAge on each commit
errCh := pgDB.StartCleanupScheduler(context.Background(), 4*time.Hour)

// Redis (github.com/redis/go-redis/v9)
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
redisDB := sessions.NewRedisDatabase[UserSession](redisClient, sessionOptions)
```

## versioning

API versioning via header, query parameter, or subdomain.

```go
import "github.com/kataras/iris/v14/middleware/versioning"

app.UseRouter(versioning.New(versioning.Options{
    Default:    "1.0.0",
    Latest:     "2.0.0",
    Strict:     false, // if true, requests must carry an explicit version
    Header:     "Accept-Version",
    QueryParam: "v",
    Subdomain:  true,
    Versions: versioning.Versions{
        {Range: ">=1.0.0 <2.0.0", Path: "/v1", Deprecated: true},
        {Range: ">=2.0.0", Path: "/v2"},
    },
    Deprecation: versioning.DeprecationOptions{
        WarnMessage:     "WARNING! You are using a deprecated version of this API.",
        DeprecationDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        DeprecationInfo: "Use version 2.0.0 instead.",
    },
}))

// Detection: Header, Query (?v=2.0.0), Subdomain (v2.api.example.com).
// "latest" resolves to Options.Latest. Deprecated matches receive the
// X-Api-Warn, X-Api-Deprecation-Date and X-Api-Deprecation-Info headers.
```

## view

Template rendering. In v14 the package moved to `middleware/view` and holds only the contract: **engine implementations live in `github.com/iris-contrib/views`**. The bundled `Blocks` engine was removed because `github.com/kataras/blocks` already satisfies `view.Engine` directly.

```go
import (
    "github.com/iris-contrib/views"
    "github.com/kataras/iris/v14/middleware/view"
)

templates := views.HTML("./views", ".html")
app.Use(view.Handler(templates))                          // app-wide
app.Router("/admin", view.Handler(adminTemplates), admin)  // or scoped per router

func page(ctx iris.Context) {
    view.Render(ctx, "index.html", "main-layout", data) // "" for no layout
}
```

Any type satisfying the interface works as an engine:

```go
type Engine interface {
    ExecuteTemplate(w io.Writer, templateName, layoutName string, data any) error
}
type EngineFuncer interface{ AddFunc(funcName string, funcBody any) }
type EngineExtenser interface{ Ext() string }
```

Per-request helpers: `view.WithLayout`, `view.Layout`, `view.WithData`, `view.Data`, `view.KeyValue`, `view.GetData`. `view.WithData` stores the same value on every request, so `view.KeyValue` copies a map before its first write and never changes the one you passed: read the result with `view.GetData(ctx)`. `view.AddFunc` and `view.WithFuncMap` are not per-request. They call `AddFunc` on the engine, which every request shares, and they run when a request arrives, which is too late for an engine that was loaded at startup. Register template functions on the engine before `Load` (blocks: `templates.Funcs(funcMap)`), never pass one that closes over `ctx`, and put a function that depends on the request into the view data, where a template calls it with `{{ call .Greet .Name }}`. `view.ErrNoEngine` is returned when no engine is registered (mapped to `Internal` by `rest/errors`). `ErrTemplateNotFound` carries the name and whether it was a layout.

`app.RegisterView` and `ctx.View` are gone; see [migration-v12-to-v14.md](./migration-v12-to-v14.md).

## websocket

WebSocket support via the neffos library.

```go
import (
    "github.com/kataras/iris/v14/middleware/websocket"
    "github.com/kataras/neffos"
    "github.com/kataras/neffos/gorilla"
)

serverEvents := neffos.Namespaces{
    "default": neffos.Events{
        neffos.OnNamespaceConnected: func(nsConn *neffos.NSConn, msg neffos.Message) error {
            ctx := websocket.GetContext(nsConn.Conn)
            log.Printf("connected from %s", ctx.RemoteAddr())
            return nil
        },
        "chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
            nsConn.BroadcastOthers(msg)
            return nil
        },
    },
}

wsServer := neffos.New(gorilla.DefaultUpgrader, serverEvents)
app.Any("/ws", websocket.New(wsServer, websocket.Options{}))
```

## Community Middleware (iris-contrib/middleware)

Thirteen middleware live outside the framework at `github.com/iris-contrib/middleware`, one Go module each, so only the one you import pulls its dependencies: `casbin` (ACL/RBAC/ABAC authorization), `cloudwatch` (AWS metrics), `cors`, `csrf`, `expmetric` (expvar counters), `jwt`, `newrelic`, `pg` (PostgreSQL), `prometheus`, `secure` (security headers), `throttler` and `tollboothic` (rate limiting), `zap` (logging). Sentry ships its own Iris handler in `getsentry/sentry-go/iris`.

Install one with `go get github.com/iris-contrib/middleware/<name>@main`.

Prefer the built-in `cors` and `jwt` over the contrib versions of the same name: the built-ins are maintained with the framework and the builder wires both.

The `main` branch targets Iris v14; the v12 line is preserved at the `v12` tag. These modules are maintained by contributors on their own schedule, not with the framework.

## Source Files

Each middleware is in its own package under `middleware/`:
`accesslog`, `apikey`, `basicauth`, `bodylimit`, `cache`, `compress`, `cors`, `counter`, `geolocation`, `grpc`, `hcaptcha`, `httpcost`, `i18n`, `ipaccess`, `jwt`, `methodoverride`, `modrevision`, `monitor`, `pprof`, `rate`, `recaptcha`, `recorder`, `recovery`, `referrer`, `requestid`, `rewrite`, `servertiming`, `sessions`, `versioning`, `view`, `websocket`
