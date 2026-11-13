# Security and Hardening

What Iris secures for you, what it deliberately leaves to the application, and every knob worth turning on before production. The v12 → v14 default changes and their escape hatches are in [migration-v12-to-v14.md](./migration-v12-to-v14.md).

## Secure by Default

v14 ships these on. Each row's escape hatch exists, but reach for it only with a reason.

| Default | Escape hatch |
|---------|--------------|
| Session cookies are `Secure` and `SameSite=Lax` | `opts.Secure = false` (plain-HTTP local dev only) |
| `ctx.AddCookie` sets `HttpOnly` and `SameSite=Lax` | `iris.CookieSameSite(...)`, and there is no way to unset `HttpOnly` per call |
| JWT reads the `Authorization` header only | `jwt.NewVerifier(...).WithQueryToken()` |
| CORS never sends `Allow-Credentials: true` with `Origin: *` | `cors.New().AllowCredentials()`. Note this covers the wildcard only: `cors.New()` reflects a *specific* origin back with credentials, see CORS below |
| Auth sign-in redirects only to same-site relative paths | Redirect from your own handler after checking an allowlist |
| OAuth2 authorization and device codes are high-entropy opaque tokens | None; codes are opaque to clients |
| Builder apps recover panics and render generic error bodies | None wanted. **Do not add a second recovery middleware** |
| Typed path-macro failures render 400 `INVALID_ARGUMENT` | Per-macro `SetErrorHandler` |
| Rate-limit rejections render JSON 429 `RESOURCE_EXHAUSTED` | `rate.Options{ExceedHandler: ...}` |

Two things Iris does **not** do, by design: it ships no CSRF middleware and injects no security headers. Both are below.

Defaults that are permissive on purpose, each with an opt-in:

| Permissive default | Opt-in |
|---|---|
| `ContextOptions.BodyMaxMemory` is `0` (unlimited) | Set it, or `bodylimit.New(n)` per route |
| `compress.Handler` decompresses request bodies with no ceiling | `compress.Safe`, or `compress.Limited(n)` |
| `cors.New()` reflects any origin **with** credentials | `cors.Strict()` |
| `cache.New` applies `public` to credentialed requests too | `cache.Strict(opts)`, or `cache.PrivateOptions` |
| `httpcache` keys omit `Authorization` and cookies | Set `Options.Headers` / `Options.Cookies` |
| `accesslog` records request bodies | `AccessLog.Redact`, or `RequestBody = false` |
| `methodoverride` accepts `_method` from query and form | `methodoverride.Options{HeadersOnly: true}` |
| The static file server serves dotfiles | `AssetValidator: fileserver.BlockHidden` |
| `pprof` and `monitor` have no auth | Wrap in `basicauth` or `ipaccess` |

## Transport Security

`app.Run` takes an `iris.ServerOptions`; `app.Listen(addr)` is the plain-HTTP shortcut. One call starts one listener.

Prefer `CertFile`/`KeyFile` or `Letsencrypt`: the framework-built TLS configs use TLS 1.3 and negotiate HTTP/2.

```go
// Explicit certificate files.
app.Run(iris.ServerOptions{Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem"})

// Let's Encrypt: the first Hosts entry is the root domain. CertFile/KeyFile
// act as the fallback when ACME cannot complete.
app.Run(iris.ServerOptions{
    Addr:        ":443",
    Letsencrypt: true,
    Hosts:       []string{"example.com", "www.example.com"},
})

// Bring your own tls.Config only when you actually need one
// (client certificates, cipher suites for legacy TLS 1.2 clients).
app.Run(iris.ServerOptions{Addr: ":443", TLSConfig: &tls.Config{ /* ... */ }})
```

A custom `TLSConfig` is normalized, not served verbatim: `NextProtos` becomes `["h2", "http/1.1"]` when nil (a non-nil list is kept, with `"http/1.1"` ensured) and `ServerName` is filled when empty. `MinVersion` is always respected as-is. Since the framework-built configs use `tls.VersionTLS13`, setting `MinVersion: tls.VersionTLS12` is a *downgrade*; do that only for clients that cannot speak TLS 1.3.

### Header limits and HTTP/2 priority

Every TLS Iris server offers "h2", so two Go 1.27 defaults apply: a request repeating one header name beyond `http.DefaultMaxHeaderValueCount` is rejected with 400, and HTTP/2 responses follow RFC 9218 client-driven prioritization. `ServerOptions.Limits` tunes both (the `LimitsOptions` type lives in the `core/server` package, no root-package alias):

```go
import "github.com/kataras/iris/v14/core/server"

app.Run(iris.ServerOptions{
    Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem",
    Limits: &server.LimitsOptions{
        MaxHeaderBytes:        512 * 1024, // request line + headers; the body is BodyMaxMemory's job.
        MaxHeaderValueCount:   32,         // tighten the repeated-header guard; 0 keeps Go's default.
        DisableClientPriority: true,       // opt out of RFC 9218 response scheduling on h2.
    },
})
```

### OnServerCreated

`ServerOptions.OnServerCreated` receives the standard `*http.Server` after Iris applies `Timeout` and `Limits`, before serving starts; a non-nil error aborts the startup. It is the escape hatch for any `http.Server` field Iris does not mirror (including fields added by future Go releases) and for TLS knobs on its `TLSConfig`, e.g. post-quantum key exchange (`tls.MLKEM1024`) or ML-DSA signature schemes. Go's default `CurvePreferences` already include the hybrid post-quantum key exchanges, which is why Iris deliberately leaves that field nil.

```go
app.Run(iris.ServerOptions{
    Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem",
    OnServerCreated: func(srv *http.Server) error {
        // Pure post-quantum key exchange only (hybrids are already on by default).
        srv.TLSConfig.CurvePreferences = []tls.CurveID{tls.MLKEM1024, tls.X25519MLKEM768}
        return nil
    },
})
```

## Bounding the Request

### Body size

Two tiers: an application-wide cap and a tighter per-route one.

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{BodyMaxMemory: 10 * iris.MB},
})
```

`BodyMaxMemory` bounds `Body`, `ReadJSON`, `ReadJSONStream` and `ReadMultipartRelated` (so also `ReadXML`/`ReadYAML`/`ReadBody` and `ctx.BindJSON`), and it bounds decompression reads, which is what stops a compression bomb. Overflow fails with `context.ErrRequestEntityTooLarge`, which `rest/errors` maps to 413. It is distinct from `PostMaxMemory`, which only bounds multipart form parsing.

```go
uploads := app.Router("/uploads", bodylimit.New(2*iris.MB)) // middleware/bodylimit
```

### Rate limiting

```go
import "github.com/kataras/iris/v14/middleware/rate"

// 10 requests/second sustained, burst 20, forget idle clients.
app.Use(rate.Limit(rate.Every(time.Second), 20, rate.Options{
    PurgeEvery:       time.Minute,
    PurgeMaxLifetime: 5 * time.Minute,
}))

// A stricter budget on the expensive route.
app.Post("/signin", rate.Limit(rate.Every(10*time.Second), 3, rate.Options{}), signin)
```

Clients are keyed by remote IP by default. To key by something else, call `SetIdentifier` from a middleware registered **before** the limiter:

```go
app.Use(func(ctx iris.Context) {
    rate.SetIdentifier(ctx, ctx.Header("X-API-Key"))
    ctx.Next()
})
app.Use(rate.Limit(1, 5, rate.Options{}))
```

Builder apps already render a JSON 429 under `RESOURCE_EXHAUSTED` with a retry hint; `rate.Options{ExceedHandler: h}` replaces it.

### Request IDs

The default generator honors a client-supplied `X-Request-ID`, which lets a client forge log entries. For anything security-adjacent, generate server-side only:

```go
app.UseRouter(requestid.New(requestid.ServerOnlyGenerator))
```

### IP allowlists

```go
handler := ipaccess.Must(
    []string{"10.0.0.0/8", "203.0.113.7"}, // allowed: exact IPs, CIDR or regex
    []string{`^198\.51\.100\.`},           // blocked, checked first
)
app.Router("/admin", handler, adminAPI)
```

Blocked patterns win over allowed. A request matching neither gets 403. It matches the **connection** address, not `RemoteAddr`, so it is not fooled by a forged `X-Forwarded-For`, and it does not work behind a proxy unless the proxy itself is allowlisted.

## Input Validation

Validate at the edge. Give the request type a `Validate() error` and `ctx.BindJSON` runs it after decoding; the handler never calls it.

```go
type CreateUser struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func (in CreateUser) Validate() error {
    return validation.Join(
        validation.String("name", in.Name).NotEmpty().Length(2, 64),
        validation.String("email", in.Email).Email(),
        validation.Number("age", in.Age).InRange(18, 130),
    )
}

func (api *UsersAPI) create(ctx iris.Context) {
    in, ok := ctx.BindJSON[CreateUser]() // 400 INVALID_ARGUMENT already written
    if !ok {
        return
    }
    id, err := api.users.Create(ctx, in)
    ctx.Created(id, err)
}
```

Typed path macros reject malformed input before your handler runs, which is cheaper and harder to forget than in-handler parsing:

```go
r.Get("/users/{id:uuid}", getUser)           // non-uuid never reaches the handler
r.Get("/plans/{page:int min(1) max(500)}", listPlans)
```

Builder apps render those failures as 400 `INVALID_ARGUMENT` with "invalid path parameter".

## Keeping Internals Private

Never let a driver or filesystem error reach the client. `rest/errors` has three tools:

```go
// At the response site: log the real error, render a generic 500.
if err != nil {
    errors.Internal.LogErr(ctx, err)
    return
}

// Same, when you also want the error attached to the (server-side) log entry.
errors.Internal.ErrPrivate(ctx, err)

// At the source layer: mark an error private so no mapper leaks its text.
return errors.Private(fmt.Errorf("dial %s: %w", dsn, err))
```

`errors.NewOptions()` already maps `database/sql.ErrNoRows` to 404, so a missing row does not surface as a 500 with a driver message.

## Sessions and Cookies

```go
opts := sessions.NewOptions(hashKey, blockKey) // 64-byte hash, 32-byte block
opts.MaxAge = 2 * time.Hour                    // default is 24h
opts.SameSite = http.SameSiteStrictMode        // tighten from the Lax default
db := sessions.NewCookieDatabase[UserSession](opts)
app.Use(sessions.Handler(db, "app_session"))
```

Rotate the session ID whenever privilege changes, most importantly right after sign-in. This is the session-fixation defense:

```go
func signin(ctx iris.Context) {
    user, err := authenticate(ctx)
    if err != nil {
        errors.Unauthenticated.Message(ctx, "invalid credentials")
        return
    }
    session := sessions.Get[UserSession](ctx)
    session.Set(UserSession{UserID: user.ID})
    if err := session.RegenerateID(ctx); err != nil { // before writing the body
        errors.Internal.LogErr(ctx, err)
        return
    }
    ctx.OK(user, nil)
}
```

`RegenerateID` is a no-op for cookie databases: their data lives in the signed and encrypted cookie, so there is no server-side ID to rotate. Use `session.Delete(ctx)` on sign-out.

Changing `hashKey` or `blockKey` signs every visitor out and nothing more: a cookie signed with the old keys, like one older than `MaxAge`, is treated as no cookie and replaced. It is not a 500.

For cookies you set yourself:

```go
ctx.AddCookie("theme", "dark")                                   // HttpOnly + SameSite=Lax
ctx.AddCookie("t", v, iris.CookieSecure)                         // Secure only when the connection is TLS
ctx.AddCookie("t", v, iris.CookieSecureForce)                    // Secure unconditionally (behind a TLS proxy)
ctx.AddCookie("t", v, iris.CookieSameSite(http.SameSiteStrictMode))
ctx.AddCookieOptions(iris.CookieSecureForce)                     // apply to every cookie this request sets
```

## Authentication Hardening

### JWT

Short-lived access tokens with a refresh pair, and a blocklist so sign-out is real:

```go
signer := jwt.NewSigner(jwt.HS256, key, 15*time.Minute)
pair, err := signer.NewTokenPair(accessClaims, refreshClaims, 24*time.Hour)

verifier := jwt.NewVerifier(jwt.HS256, key).WithDefaultBlocklist()
app.Use(verifier.Verify(func() any { return new(UserClaims) }))
```

The verifier reads the `Authorization` header only. `WithQueryToken()` adds `?token=`; take it only for clients that cannot set a header (browser downloads), and understand that the token then lands in access logs, proxy logs, browser history and `Referer` headers.

For tokens issued by an identity provider (a `kid` in the header, RS256), set `Verifier.HeaderValidator` to the `ValidateHeader` method of a `github.com/kataras/jwt` key set, see [middleware-reference.md](./middleware-reference.md#jwt). A validator of your own must always compare the header's `alg` with the algorithm it expects.

### Basic authentication

```go
app.Router("/admin", basicauth.New(basicauth.Options[basicauth.Map]{
    Realm:     "Authorization Required",
    Allow:     basicauth.AllowUsersFile[basicauth.Map]("users.yml", basicauth.BCRYPT),
    HTTPSOnly: true, // refuse to accept credentials over plaintext
    MaxTries:  3,    // 403 after three failures
    MaxAge:    time.Hour,
}), adminAPI)
```

`basicauth.Default(map[string]string{"admin": "pass"})` is for local development only: plaintext credentials in source.

### API keys

A caller that is a program, not a person, presents a shared secret. Use `middleware/apikey`; do not read `X-API-Key` in a hand-written middleware and compare it with `==`, which leaks the key's length and a prefix through timing and gives every consumer the same credential.

```go
guard := apikey.New(apikey.Options{
    Hash: apikey.SHA256, // Keys hold digests, so the config holds no usable credential
    Keys: map[string]string{
        "reporting": os.Getenv("REPORTING_API_KEY_SHA256"),
        "ops":       os.Getenv("OPS_API_KEY_SHA256"),
    },
})

app.Router("/api", guard, reportsAPI)
```

- One entry per consumer. Two principals sharing a secret authenticate each other's requests, and which one gets reported depends on map iteration order.
- The comparison reduces both sides to a 32-byte SHA-256 digest and runs `subtle.ConstantTimeCompare` over every configured key with no early exit, so neither which key matched nor how long it is can be read out of the response time.
- The default is deny: no `Keys` and no `Lookup` rejects every request and logs a warning. `AllowAnonymous: true` is for local development and is never true in production.
- `Query` puts the key in the URL, where it reaches browser history, proxy logs, `Referer` headers and the access log, whose `Redact` hook covers bodies only. Enable it only for a caller that cannot set a header, and mask the parameter in a custom `accesslog` formatter first.
- `Scheme: apikey.NoScheme` accepts a bare value for a custom header such as `X-API-Key`. Prefer the `Authorization: Bearer` default: proxies, CDNs and log scrubbers redact that header by name and do not redact a custom one.
- `Lookup` resolves keys held in a database. Its error is a 500, not a 401, so a store that is down never reads as an invalid key.

### Password hashing cost

```go
password.Cost = 12 // auth/identity/password; default is bcrypt.DefaultCost (10)
```

The cost is embedded in each hash, so raising it keeps existing hashes verifiable.

## CORS

```go
app.UseRouter(cors.New().
    ExtractOriginFunc(cors.StrictOriginExtractor). // do not trust a Referer fallback
    AllowOrigins("https://app.example.com").
    AllowHeaders("Content-Type", "Authorization").
    AllowCredentials().
    MaxAge(12 * time.Hour).
    Handler())
```

`AllowCredentials()` is safe here because a specific origin is configured.

Be precise about what the middleware guarantees: it refuses to pair credentials with the wildcard `*`, because browsers reject that combination anyway. It does **not** refuse to pair them with a *reflected* origin, and reflecting every origin is what `cors.New()` does by default. So `app.UseRouter(cors.New().Handler())` on an API that reads cookies lets any site read authenticated responses.

`cors.Strict()` is the same constructor with credentials off until `AllowCredentials()` is called. Prefer it whenever the API serves authenticated responses.

Origin pattern matching: a `*` never crosses a `/`, and a `*` in a label position (directly after a `.`) matches exactly one label. `https://www.example.*` therefore matches `https://www.example.com` but not `https://www.example.evil.com`. Written without the dot, `https://www.example*` is a prefix match and does accept `https://www.exampleshop.com`.

`MaxAge` emits whole seconds, as `Access-Control-Max-Age` requires.

The builder's `AllowOrigin` step takes a comma-separated line for the simple case. Full options in [middleware-reference.md](./middleware-reference.md).

## CSRF Protection

Iris ships no CSRF middleware. With `SameSite=Lax` cookies the remaining gap is same-site sub-resource requests, and for a JSON API a required custom header closes it (a cross-origin form cannot set one):

```go
func requireCustomHeader(ctx iris.Context) {
    switch ctx.Method() {
    case iris.MethodGet, iris.MethodHead, iris.MethodOptions:
        ctx.Next()
        return
    }
    if ctx.Header("X-Requested-With") == "" {
        errors.PermissionDenied.Message(ctx, "custom request header required")
        return
    }
    ctx.Next()
}
```

For cookie-authenticated HTML forms, use a synchronizer token stored in the session instead.

## Static File Safety

The file server serves dotfiles by default. It logs a warning at startup when the served directory holds a well-known sensitive dotfile and no `AssetValidator` is set, but the files are still served. Block them:

```go
app.Router("/static", fileserver.New("./public", fileserver.Options{
    IndexName:      "index.html",
    AssetValidator: fileserver.BlockHidden,
}))
```

`BlockHidden` denies any path containing a dot-prefixed segment: `.git`, `.env`, `.htpasswd`. It also blocks `.well-known`, so compose your own validator when you need ACME challenges served from the same tree:

```go
AssetValidator: func(ctx iris.Context, name string) bool {
    if strings.HasPrefix(name, "/.well-known/") {
        return true
    }
    return fileserver.BlockHidden(ctx, name)
},
```

**Symbolic links.** A string directory is served through `http.Dir`, which checks the request path as text but follows a symbolic link inside the directory that points outside it. To confine the server to the directory, pass the `fs.FS` of an `os.Root`:

```go
root, err := os.OpenRoot("./public")
if err != nil {
    log.Fatal(err)
}
app.Router("/static", fileserver.New(root.FS(), fileserver.Options{AssetValidator: fileserver.BlockHidden}))
```

A link that leaves the root then answers 404. Two costs: the root holds the directory open for the life of the process, so a deploy that replaces the directory keeps serving the old one until restart; and a link that leaves the directory on purpose (say `./public/shared` pointing elsewhere) is no longer served at all.

## Security Headers

Iris injects none. Register your own on `UseRouter` so they also cover 404s:

```go
app.UseRouter(func(ctx iris.Context) {
    ctx.SetHeader("X-Content-Type-Options", "nosniff")
    ctx.SetHeader("X-Frame-Options", "DENY")
    ctx.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
    ctx.SetHeader("Content-Security-Policy", "default-src 'self'")
    if ctx.IsSSL() { // never send HSTS over plaintext
        ctx.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    }
    ctx.Next()
})
```

## Health Endpoint Exposure

The builder's `Health(true, env, developer)` step publishes the environment name and a developer contact. That is fine internally and a disclosure on a public edge. Drop it or gate it:

```go
builder.NoHealth()                              // no endpoint at all
app.Router("/health", basicauth.New(opts), h)   // or keep it behind auth
```

## Production Checklist

| Area | Action |
|------|--------|
| TLS | `CertFile`/`KeyFile` or `Letsencrypt` (TLS 1.3, h2); a custom `TLSConfig` is normalized, its `MinVersion` respected |
| Server limits | `Limits: &server.LimitsOptions{...}` for header size/count and RFC 9218 h2 priority; `OnServerCreated` for unmirrored `http.Server`/TLS knobs |
| Body | `BodyMaxMemory` set app-wide; `bodylimit` on upload routes |
| Rate | Global limiter plus a tighter budget on auth routes |
| Request IDs | `requestid.ServerOnlyGenerator` |
| Validation | `Validate()` on every request type; typed path macros |
| Errors | `errors.Private` at the source, `Internal.LogErr` at the response |
| Sessions | 64/32-byte keys, short `MaxAge`, `RegenerateID` after sign-in |
| Cookies | `CookieSecureForce` behind a TLS proxy; `SameSiteStrict` where usable |
| JWT | Short access max-age, refresh pair, blocklist, no query token |
| API keys | `apikey.New` with `Hash: apikey.SHA256`, one key per consumer, header not query |
| CORS | Specific origins, `StrictOriginExtractor` |
| CSRF | Custom-header guard (JSON API) or session token (forms) |
| Static | `AssetValidator: fileserver.BlockHidden` |
| Headers | Own `UseRouter` middleware; HSTS gated on `ctx.IsSSL()` |
| Secrets | Environment only, fail fast at startup, never in the repo |
| Health | `NoHealth()` or auth-gated on a public edge |
| Logs | Access logs hold IPs, paths and query strings. Treat them as sensitive; do not log bodies in production |
| Dependencies | `govulncheck ./...` in CI |

## Source Files

- `middleware/apikey/apikey.go` - `New`, `Options`, `Get`, `SHA256`, `NoScheme`, `ErrMissing`, `ErrInvalid`
- `options.go`, `context/context_options.go` - `ServerOptions`, `BodyMaxMemory`, `RequestValidator`
- `core/server/server.go`, `core/server/limits.go` - TLS config normalization, `LimitsOptions`, `OnServerCreated`
- `middleware/rate/`, `middleware/bodylimit/`, `middleware/ipaccess/`, `middleware/requestid/` - Request bounding
- `rest/errors/`, `rest/errors/validation/` - Canonical errors, privacy, validation rules
- `middleware/sessions/options.go`, `middleware/sessions/session.go` - Session defaults, `RegenerateID`
- `middleware/jwt/verifier.go`, `middleware/basicauth/`, `auth/identity/password/` - Authentication
- `middleware/cors/` - Origin handling
- `controller/fileserver/fileserver.go` - `Options`, `BlockHidden`
- `MIGRATION.md` - "Breaking default changes" and "Recommended opt-in hardening"
