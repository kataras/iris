# Context API

The `iris.Context` (alias for `*context.Context`) is the core request/response object. A new context is acquired from a sync.Pool for each connection and released automatically.

For the typed binding accessors (generic `ctx.BindParam`, `ctx.BindJSON`, `auth.GetUser`), see ./rest-helpers.md.

## Request Reading

### JSON

`ctx.BindJSON[T]()` is the reader handlers should use: it decodes, runs the payload's `Validate() error` and writes the 4xx itself, so the handler carries no error branch. See ./rest-helpers.md.

```go
type CreateUserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func createUser(ctx iris.Context) {
    input, ok := ctx.BindJSON[CreateUserInput]()
    if !ok {
        return // the 400 is already written.
    }
    // use input...
}
```

`ctx.ReadJSON(&v)` is the lower-level call `BindJSON` is built on. Reach for it when the target is not a fresh value (an existing struct to merge into) or when you need the reader options below.

```go
var input CreateUserInput
if err := ctx.ReadJSON(&input); err != nil {
    errors.InvalidArgument.Detail(ctx, "invalid json", err.Error())
    return
}
```

**JSON Reader options:**

```go
ctx.ReadJSON(&v, iris.JSONReader{
    DisallowUnknownFields: true,  // Reject unknown fields
    CaseSensitive:         false, // true restores json/v2's strict member matching
    ArrayStream:           false, // Stream array elements (ReadJSONStream only)
})
```

### JSON Stream

```go
func handler(ctx iris.Context) {
    var users []User
    err := ctx.ReadJSONStream(func(decode iris.DecodeFunc) error {
        var u User
        if err := decode(&u); err != nil {
            return err
        }
        users = append(users, u)
        return nil
    }, iris.JSONReader{})
}
```

### XML

```go
var input XMLInput
if err := ctx.ReadXML(&input); err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

### YAML

```go
var input YAMLInput
if err := ctx.ReadYAML(&input); err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

### Form Data

```go
type UserForm struct {
    Name  string `form:"name"`
    Email string `form:"email"`
    Age   int    `form:"age"`
}

var form UserForm
if err := ctx.ReadForm(&form); err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

`ReadForm` reports a client field the struct does not declare as an error; `iris.IsErrPath(err)` recognises that class. A form carrying the hidden `csrf.token` field (`context.CSRFTokenFormKey`) that the struct does not declare is the one exception: `ReadForm` drops only that unknown-key entry and returns every other decode error (a bad value in another field, a second unknown key) as a `schema.MultiError` without it, so `iris.IsErrPath(err)` still reports true when only unknown keys are left. The decoder splits keys on dots, so `form:"csrf.token"` never receives the token; to bind it, tag a field `form:"csrf"` whose type has a field tagged `form:"token"`. `context.IsErrPathCSRFToken(err)` is for code that decodes the form itself and needs to recognise the CSRF case, also when the `schema.MultiError` is wrapped.

### Query Parameters

```go
type SearchQuery struct {
    Keyword  string `url:"keyword"`
    Page     int    `url:"page"`
    PerPage  int    `url:"per_page"`
}

var q SearchQuery
if err := ctx.ReadQuery(&q); err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

`ReadQuery` binds a field that has no `url` tag by its `form` tag, so a struct tagged `form` only serves a GET query and a POST form. `ctx.ReadBody` reads the query on a GET (and on another method when the request has a query and no content type it knows), so the same holds there, and in a typed handler (`handler.Handler`) whose request type has no `url`, `param` or `header` field. When the request type does have one of those, the typed handler does not bind its `form` fields from the query. For a field with a `form` tag and no `url` tag: the Go field name is not a query key, `form:"-"` keeps it out of the query, and `form:"name,required"` is enforced on a non-empty query. A field with a `url` tag keeps it, `url:"-"` stays excluded, and a field with neither tag binds by its field name.

A `types.SimpleDate` field (`rest/types`) binds from `YYYY-MM-DD` in `ReadQuery`, `ReadParams`, `ReadURL`, `ReadForm` and `ReadHeader`, because the type implements `encoding.TextUnmarshaler`. A `time.Time` field takes RFC 3339 there.

Or individually through `ctx.Query()`, which returns `encoding.StringSliceEntries` (a read-only, sorted view of the URL query):

```go
keyword := ctx.Query().Get("keyword")          // string, "" if missing
page := ctx.Query().GetInt("page")             // int, -1 if missing or invalid
active := ctx.Query().GetBool("active")        // bool, false if missing
ttl := ctx.Query().GetDuration("ttl")          // time.Duration ("30s", "1h30m"), 0 if missing or invalid
since := ctx.Query().GetTime("since")          // time.Time (RFC 3339), zero time.Time if missing or invalid
addr := ctx.Query().GetIP("addr")              // netip.Addr, zero (IsValid() false) if missing or invalid
ref := ctx.Query().GetUUID("ref")              // uuid.UUID, zero (uuid.Nil()) if missing or invalid
exists := ctx.Query().Exists("keyword")        // bool

// Multi-value keys:
cols := ctx.Query().GetStringSlice("columns")  // splits the joined value by the configured separator
raw := ctx.Query().GetSlice("columns")         // the original []string from the query map
```

There is no `URLParam` or `URLParamIntDefault` method in v14; use `ctx.Query()` accessors or `ReadQuery`.

### Strict Query Readers

The `Query()` accessors are lenient: `GetInt` returns `-1` both for a missing key and for `?page=banana`. Three readers make a present but unparsable value a 400 naming the key, never a silent zero. The second result is the `ok` of the `Bind*` family: false means the response was already written.

```go
from, ok := ctx.QueryDate("from", iris.QueryRequired())  // types.SimpleDate
if !ok {
    return
}

limit, ok := ctx.QueryInt("limit", iris.QueryDefault("25"))
if !ok {
    return
}

detailed, ok := ctx.QueryBool("detailed")  // ParseBool plus yes/no/on/off, any case
if !ok {
    return
}
```

Options (`iris.QueryOption`, aliases of `context.*`):

| Option | Effect |
|--------|--------|
| `iris.QueryRequired()` | An absent or blank value is a 400 naming the key |
| `iris.QueryDefault(raw)` | The raw string parsed when the key is absent, in the client's own wire form. Wins over `QueryRequired`. An unparsable default panics on the first request |
| `iris.QueryZeroValues(v...)` | Raw values that mean "unset" (`"0"`, `"zero"`, `"0000-00-00"`, `"null"`), each read exactly like an absent value |

`?from=null` is a 400 unless `QueryZeroValues("null")` declared it: what a raw value means is the endpoint's decision, not the parser's.

A key may name several alternatives separated by a space, tried in the order stated; the first that carries a value wins. A URL query key cannot contain a space, so a key without one is a single key.

```go
date, ok := ctx.QueryDate("date day from")  // ?day=2026-01-02 answers here
if !ok {
    return
}
```

The 400 for a missing value names every accepted key ("missing date, day or from query parameter"); the 400 for an unparsable one names only the key that carried it, since the client did not send the others.

### Binding an Update and Its Columns

`ctx.BindUpdate[T]()` decodes the body exactly as `BindJSON` does (request validator and `Validate` included) and returns the `columns` query list next to it.

```go
input, columns, ok := ctx.BindUpdate[UpdateUserInput]()
if !ok {
    return // 400 already written: the body, or an unknown column.
}

updated, err := svc.UpdateUser(ctx, id, input, columns...)
ctx.NoContent(updated, err)
```

- Both spellings are accepted: `?columns=a&columns=b` and `?columns=a,b`. Values are trimmed, empties dropped, duplicates removed, the client's order kept. No columns is a nil slice and means a full update.
- Every column is checked against the JSON member names of `T` (the `json` tag name, or the field name where there is none; embedded structs flattened, `json:"-"` skipped). An unknown one answers 400 naming the column and the allowed names.
- The comparison is exact where the body decoder is case-insensitive, because the name reaches an `UPDATE`. A non-struct `T` (a map) has no member names and accepts every column.
- The names returned are the client's: map them to real columns, never interpolate them into SQL.

### Auto-Detect Content Type

```go
// ReadBody auto-detects content type and reads accordingly
var input Input
if err := ctx.ReadBody(&input); err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
```

### Binary Formats

```go
// Protobuf
var req HelloRequest
ctx.ReadProtobuf(&req)

// Protobuf JSON
ctx.ReadProtobufJSON(&req)

// MsgPack
var msg Message
ctx.ReadMsgPack(&msg)
```

### Path Parameters

`ctx.Params()` returns `*encoding.PathParams` (an `Entries[any]`). Accessors return the value directly, with a type-specific default on missing key or failed conversion (NOT a `(value, error)` tuple):

```go
id := ctx.Params().Get("id")              // any (raw value)
name := ctx.Params().GetString("name")    // string, "" on failure
page := ctx.Params().GetInt("page")       // int, -1 on failure
id64 := ctx.Params().GetInt64("id")       // int64, -1 on failure
uid := ctx.Params().GetUint64("uid")      // uint64, 0 on failure
active := ctx.Params().GetBool("active")  // bool, false on failure
sid := ctx.Params().GetUUID("sid")       // uuid.UUID, zero (uuid.Nil()) on failure
```

| Accessor | Failure default |
|----------|-----------------|
| `GetString` | `""` |
| `GetInt`, `GetInt64` | `-1` |
| `GetUint`, `GetUint64` | `0` |
| `GetFloat64` | `-1` |
| `GetBool` | `false` |
| `GetUUID` | zero `uuid.UUID` (equal to `uuid.Nil()`) |
| `GetDuration` | `0` |
| `GetTime` | zero `time.Time` |
| `GetIP` | zero `netip.Addr` (`IsValid()` false) |

A string value is parsed: `GetUUID` with `uuid.Parse`, `GetDuration` with `time.ParseDuration`, `GetTime` as RFC 3339 (through `encoding.ParseTime`; calendar days stay with `GetDate`), `GetIP` with `netip.ParseAddr`. The same getters exist on `ctx.Query()`, `ctx.Form()`, `ctx.PostForm()` and `ctx.Values()`. On a `{id:uuid}` route the macro already stored a `uuid.UUID`, so `GetUUID("id")` returns it as is and `GetString("id")` returns its canonical lowercase text.

On a single `encoding.Entry` (from `GetEntry(key)`), each of the four types has three forms: `UUID()` parses a string and returns the zero value on failure, `GetUUID() (uuid.UUID, bool)` reports false unless the stored value already is a `uuid.UUID`, and `MustUUID()` panics unless it is. `Duration`, `Time` and `IP` follow the same pattern. `encoding.ParseTime` is a variable (RFC 3339, with or without fractional seconds); assign your own function to accept other layouts.

### Headers (request)

Request headers are read with `ctx.Header(name)` (joins multiple values with ", "):

```go
contentType := ctx.Header("Content-Type")
auth := ctx.Header("Authorization")
```

`ctx.ContentType()` is a request method: it parses the request's Content-Type header and returns a `ContentType` struct (not a setter). `ctx.MediaType()` returns just the media type string. Struct binding: `ctx.ReadHeader(&ptr)` with `header` field tags.

### Raw Body

```go
body, err := ctx.Body()             // []byte
n, err := ctx.BodyLength()          // int64, size without decoding
```

`BodyLength` normally reads `Content-Length` without touching the body. Two edge cases: a missing content length makes net/http substitute `http.NoBody`, so it returns `0`; and an *unknown* length (`-1`, chunked encoding) makes it drain the body to count the bytes, so a later `ctx.Body()` reads an empty stream. Use `recorder.RecordRequestBody(ctx)` first if you need both.

When `ContextOptions.BodyMaxMemory` is set, reading a body larger than the limit fails with `context.ErrRequestEntityTooLarge`, which `rest/errors` maps to 413.

### File Uploads

```go
file, info, err := ctx.FormFile("upload")
if err != nil {
    errors.InvalidArgument.Err(ctx, err)
    return
}
defer file.Close()
// info.Filename, info.Size, info.Header
```

### Client Information

```go
ip := ctx.RemoteAddr()   // resolved public client IP (skips private subnets)
addr := ctx.Addr()       // first-hop address, private addresses allowed
method := ctx.Method()
path := ctx.Path()             // decoded: "/a b"
raw := ctx.EscapedPath()       // as the client sent it: "/a%20b"
host := ctx.Host()
subdomain := ctx.Subdomain()
isAjax := ctx.IsAjax()
isTLS := ctx.IsSSL()
```

`ctx.CurrentRoute()` returns the read-only route this request matched (`GetName`, `GetMethod`, `GetPath`, `StaticPath`, `ResolvePath` and friends), and `ctx.LastPathParameter()` the last dynamic parameter of that route. Both dropped the `Get` prefix in v14: `GetCurrentRoute`, `GetLastPathParameter` and `RequestPath` no longer exist.

`RemoteAddr` and `Addr` differ in what they trust and what they filter:

- **`RemoteAddr()`** walks the headers named in `ContextOptions.RemoteAddrHeaders` (empty by default), skipping addresses inside `RemoteAddrPrivateSubnets`, and falls back to the connection address. Private or first-entry addresses are only honored when `RemoteAddrHeadersForce` is true.
- **`Addr()`** returns the first valid address it finds, **including private ones**, so it still yields something useful for a local client.

Security note: `RemoteAddr` trusts whatever headers you configured, and a client can forge those. Only enable `RemoteAddrHeaders` behind a proxy you control, never make an authorization decision from either value, and be careful about IP-based rate limiting on a forgeable header. For connection-address filtering that headers cannot influence, use the `ipaccess` middleware.

## Response Writing

The response helpers are what a handler calls: `ctx.OK`, `ctx.Created`, `ctx.NoContent`, `ctx.Count`, `ctx.List` and `ctx.Paginated` each take the service's `(value, err)` pair, set the status and route the error through `rest/errors`. See ./rest-helpers.md. The writers below are the layer beneath them, for a response that is none of those shapes.

### JSON

```go
ctx.JSON(iris.Map{"key": "value"})

// With options
ctx.JSON(data, iris.JSON{
    Indent: "  ",
    Prefix: "",
})

// Struct
ctx.JSON(user)

// Slice
ctx.JSON(users)
```

### JSON Arrays: ctx.List

`ctx.List(items, err, tolerate ...error) bool` renders a list as a 200 JSON array and folds the error mapping in, like the rest of the response helper family.

```go
users, err := svc.ListUsers(ctx)
ctx.List(users, err, sql.ErrNoRows)
```

- A nil or empty slice (or array) renders `[]`, never `null`.
- An error matching one of `tolerate` (`errors.Is`, so a wrapped one matches) also renders `[]` with 200: a repository's not-found for an empty set is not a failure of the list endpoint.
- Any other error goes through the error renderer and `List` returns false.
- A non-slice, non-array `items` panics naming the type. Use `ctx.Paginated` when the list can grow: `List` sends the whole set.

### XML

```go
ctx.XML(data)

// With options
ctx.XML(data, iris.XML{Indent: "  "})

// Map to XML
ctx.XML(iris.XMLMap("Root", map[string]any{"key": "value"}))
```

### YAML

```go
ctx.YAML(data)
```

### HTML

```go
ctx.HTML("<h1>Hello</h1>")           // HTML(body string): no format args
ctx.Writef("<h1>Hello, %s!</h1>", name) // use Writef for formatted output
```

### Markdown

```go
ctx.Markdown([]byte("# Hello World"))
// Renders as HTML: <h1>Hello World</h1>
```

### Plain Text

```go
ctx.Text("Hello, World!")
ctx.Writef("Hello, %s!", name)
ctx.WriteString("raw string")
```

### Binary Formats

```go
ctx.Binary([]byte("binary data"))
ctx.MsgPack(data)
ctx.Protobuf(protoMessage)
ctx.ProtobufJSON(protoMessage)
```

### Fast-path JSON (encoding/json/v2 MarshalerTo)

```go
// A type implementing json.MarshalerTo (encoding/json/v2) streams itself
// straight into the response encoder; plain ctx.JSON picks it up.
ctx.JSON(data)
```

### JSONP

```go
ctx.JSONP(data, iris.JSONP{Callback: "myCallback"})
// Output: myCallback({"key":"value"})
```

The callback is written into a `text/javascript` body, so it is validated as a JavaScript identifier reference (`cb`, `window.cb`, `cbs[0]`). Anything else fails with `context.ErrInvalidJSONPCallback` and nothing is written. This matters because the usual source is the request itself:

```go
ctx.JSONP(data, iris.JSONP{Callback: ctx.Query().Get("callback")})
```

and an unvalidated callback is attacker-authored script served from your own origin. `context.IsValidJSONPCallback` exposes the same check for your own validation.

### Status Code

```go
ctx.StatusCode(iris.StatusCreated)        // 201
ctx.StatusCode(iris.StatusNoContent)      // 204
ctx.StatusCode(iris.StatusNotFound)       // 404
```

A handler does not normally reach for these: `ctx.Created(id, err)` carries the 201, `ctx.NoContent(modified, err)` the 204 (or a 304), and `errors.NotFound` the 404.

### Headers (response)

```go
ctx.SetHeader("X-Custom", "value")  // replace a response header; empty value deletes it
ctx.AddHeader("Vary", "Accept-Encoding") // append, for list-valued headers
ctx.ResponseHeader("X-Custom")      // read a RESPONSE header (request read is ctx.Header)
ctx.SetContentType("application/json") // set the Content-Type, returns error on invalid value
ctx.ResponseContentType()           // parsed response Content-Type (request read is ctx.ContentType)
ctx.ResponseMediaType()             // just the media type string, faster than the above
```

`GetHeader`, `GetContentType` and `GetMediaType` do not exist in v14. The `Get` prefix read as a response accessor while the method actually read the request, so the names were dropped and each side got an unambiguous one. Porting v12 code, `ctx.GetHeader(k)` becomes `ctx.Header(k)`, never `ctx.ResponseHeader(k)`:

```go
apiKey := ctx.Header("X-API-Key")       // request  (what the client sent)
cached := ctx.ResponseHeader("X-Cache") // response (what this handler set)
```

Reading the key is not the same as guarding a route with it. `apikey.New` from `middleware/apikey` does the comparison in constant time, names the caller and renders the 401; see [middleware-reference.md](./middleware-reference.md#apikey).

Multiple values are joined with `", "` in both directions, unlike `http.Header.Get`, which returns only the first.

`SetHeader` replaces and `AddHeader` appends, the way `http.Header.Set` and `http.Header.Add` do. There is no `SetHeaderOverride`:

```go
ctx.SetHeader("X-Cache", "HIT")                    // replace; correct for single-valued ones
ctx.AddHeader("Vary", "Accept-Encoding")           // append; correct for list-valued headers
ctx.ResponseWriter().Header().Add("X-Multi", "v2") // the underlying map, if you prefer
```

`SetHeader` with an empty value removes the header; `AddHeader` ignores one. Reach for `AddHeader` only where a list is meant, such as `Vary` or `Link`. Two `Access-Control-Allow-Origin` headers, which is what registering a middleware twice used to produce, make browsers reject the response outright, and a replacing `SetHeader` cannot produce them.

### Response Writer Access

```go
w := ctx.ResponseWriter()       // iris ResponseWriter (StatusCode(), Written(), Flush(), Hijack()...)
std := ctx.ResponseWriter().Unwrap() // the underlying http.ResponseWriter (renamed from Std)
// Unwrap works with http.NewResponseController (Go 1.20+).
```

### Cookies

```go
// Set a cookie from a struct (iris.Cookie is an alias for http.Cookie):
ctx.SetCookie(&iris.Cookie{
    Name:     "session",
    Value:    "abc123",
    Path:     "/",
    HttpOnly: true,
    Secure:   true,
    SameSite: iris.SameSiteLaxMode,
})

// Or set a cookie by name and value with option helpers.
// Defaults: Path "/", HttpOnly true, expires after 365 days.
ctx.AddCookie("session", "abc123",
    iris.CookieHTTPOnly(true),
    iris.CookieSecure, // passed WITHOUT arguments: sets Secure when the request connection uses TLS
    iris.CookieExpires(24*time.Hour),
    iris.CookiePath("/"),
    iris.CookieSameSite(iris.SameSiteLaxMode),
)

// Set a cookie as given (no defaults applied). It replaces, like SetHeader:
ctx.SetCookie(&iris.Cookie{Name: "session", Value: "xyz", Path: "/"})

// Read a cookie (returns *http.Cookie, error):
cookie, err := ctx.Cookie("session")

// Read just the value (returns "" when missing):
value := ctx.CookieValue("session")

// Remove a cookie:
ctx.RemoveCookie("session")
```

There is no `GetCookie` method: read with `Cookie` or `CookieValue`.

`SetCookie`, `AddCookie` and `RemoveCookie` replace, like `SetHeader`: when the response already carries a `Set-Cookie` line with the same name, path and domain (domain compared case-insensitively, without a leading dot), that line is replaced in place and further duplicates are dropped, so two calls emit one line (RFC 6265 section 4.1.1). A cookie with the same name but another path or domain is added as its own line, and a cookie set then removed in one response leaves one line, the removal. There is no `UpsertCookie`; `SetCookie` is the replace-or-add call. `AddCookie` gives a cookie with no `CookieExpires` option the lifetime in `context.DefaultCookieExpiration` (365 days), changed process-wide with `iris.Patches().Context().SetDefaultCookieExpiration(d)`.

**CookieOption helpers.** A `CookieOption` is `func(ctx *Context, c *http.Cookie, op uint8)`. Some helpers are option VALUES used bare (`iris.CookieSecure`, `iris.CookieSecureForce`, `iris.CookieCleanPath`, `iris.CookiePartitioned`); others are constructors that return an option (`iris.CookieHTTPOnly(bool)`, `iris.CookieExpires(time.Duration)`, `iris.CookiePath(string)`, `iris.CookieSameSite(http.SameSite)`, `iris.CookieDomain(string)`, `iris.CookieEncoding(secureCookie)`).

| Helper | Effect |
|--------|--------|
| `iris.CookieSecure` | Secure=true only when the request came over TLS |
| `iris.CookieSecureForce` | Secure=true always |

**Request-scoped option defaults.** Register options once (for example in a middleware) and they apply to all `SetCookie`/`AddCookie`/`RemoveCookie`/`Cookie` calls of the request:

```go
ctx.AddCookieOptions(iris.CookieSecure, iris.CookiePath("/api"))
// ...
ctx.ClearCookieOptions() // remove previously registered options
```

### Redirect

```go
ctx.Redirect("/new-path", iris.StatusFound)           // 302
ctx.Redirect("/new-path", iris.StatusMovedPermanently) // 301
```

### File Operations

```go
// Send file as download
ctx.SendFile("path/to/file.pdf", "download.pdf")

// Serve file inline
ctx.ServeFile("path/to/file.html")

// Rate-limited variants: limit is bytes per second, burst is the bucket size in bytes.
ctx.ServeFileWithRate("path/to/video.mp4", 1024*1024, 64*1024) // ~1 MB/s
ctx.SendFileWithRate("path/to/file.pdf", "download.pdf", 256*1024, 32*1024)
```

### Component Rendering

```go
// Render a templ component or any type implementing Component interface
ctx.RenderComponent(component)
```

The `Component` interface:

```go
type Component interface {
    Render(ctx context.Context, w io.Writer) error
}
```

Shorthand for route registration:

```go
app.Get("/", iris.Component(myComponent))
```

### RenderResponse

```go
// For custom renderers implementing the ResponseRenderer interface
ctx.RenderResponse(iris.StatusOK, customRenderer)
```

## Content Negotiation

```go
ctx.Negotiate(iris.N{
    Text: "plain text response",
    JSON: iris.Map{"format": "json"},
    XML:  iris.Map{"format": "xml"},
    HTML: "<h1>HTML response</h1>",
})
```

## Request-Scoped Values

```go
// Store values for the duration of the request
ctx.Values().Set("key", value)
ctx.Values().Get("key")
ctx.Values().GetString("key")
ctx.Values().GetInt("key")

// User management
ctx.SetUser(user)     // Set authenticated user
user := ctx.User()    // Get authenticated user
```

Never write the `User` itself to the response. `SimpleUser` has `Password` and `Token` fields and serializes them like any other, and a value passed to `SetUser` is wrapped in a `UserPartial` whose `Raw` field is serialized whole, so `ctx.JSON(ctx.User())` returns whatever secret the value carries. The basicauth middleware puts the plaintext password on the `*SimpleUser` it builds, and `basicauth.GetUser[U](ctx)` returns your own type as it was stored. Read the fields you mean to expose:

```go
u := ctx.User()
id, _ := u.GetID()
username, _ := u.GetUsername()
ctx.JSON(iris.Map{"id": id, "username": username})
```

## Flow Control

```go
ctx.Next()                              // Call next handler in chain
ctx.Skip()                              // Skip the next handler only
ctx.StopExecution()                     // Stop handler chain (following Next calls are ignored)
stopped := ctx.IsStopped()              // Reports whether StopExecution was called
resumed := ctx.ResumeExecution()        // Undo StopExecution; reports whether it restored the chain
ctx.StopWithStatus(iris.StatusForbidden) // Stop with status code
ctx.StopWithError(iris.StatusBadRequest, err) // Stop with error: writes err's text, or only the status when err is private anywhere in its chain
ctx.StopWithJSON(iris.StatusOK, data)   // Stop with JSON response
ctx.StopWithText(iris.StatusOK, "done") // Stop with text response
```

## Application Access

```go
app := ctx.Application()
logger := ctx.Application().Logger()
isDebug := ctx.IsDebug()
```

## Context as context.Context

`iris.Context` implements the standard `context.Context` interface. You can pass it directly to service methods that complete before the handler returns:

```go
func (api *UserAPI) list(ctx iris.Context) {
    // ctx can be passed to any function expecting context.Context
    users, err := api.service.ListUsers(ctx)
    // ...
}
```

For reading Iris data (path params, authenticated user) back out of a plain `context.Context` inside services, type-assert it back to `iris.Context`: see ./rest-helpers.md.

### Consumers that keep the context

`iris.Context` is pooled and recycled after the handler returns, so it must never reach code that holds a context past the call itself. The subtle case is an outbound HTTP request: the transport keeps the request context for cancel propagation and can still walk it after the response is delivered, when the handler may already have returned, which races with the pool. Database drivers and anything that spawns its own goroutine have the same shape. Pass `ctx.StdContext()` there: it returns the request's standard library context (`Request().Context()`), still canceled on client disconnect, with no pooled value in the chain.

```go
req, _ := http.NewRequestWithContext(ctx.StdContext(), http.MethodGet, url, nil) // safe to hand to a client
users, err := api.service.ListUsers(ctx) // fine: returns before the handler does
```

### Escaping request cancellation

`iris.Context` is canceled when the client disconnects, which is correct for work the client is waiting on and wrong for work that must finish. `WithoutCancel()` returns a standard context that keeps the request's values but is never canceled:

```go
go audit.Record(ctx.WithoutCancel(), order) // survives the client hanging up
```

This matters because **`iris.Context` is pooled and recycled after the handler returns**. A goroutine holding `ctx` itself reads a context that now belongs to a different request. The context returned by `WithoutCancel` is a plain standard one and is safe to keep.

```go
go audit.Record(ctx, order)                 // WRONG: ctx is recycled underneath
go audit.Record(context.Background(), order) // works, but loses request values
```

### Cleaning up after the handler returns

`ctx.OnRelease(fn)` registers a callback the router runs after the handler chain has returned and **before** the response writer and the Context go back to their pools, in reverse registration order. It runs on the request's own goroutine, so it may touch the Context, and it is the last point at which the response writer is still this request's.

It exists for goroutines that write after the handler has moved on: a heartbeat, a ticker draining a batch. The Context is released inside `ServeHTTP`, before net/http cancels the request context, so a goroutine waiting on `Request().Context().Done()` can otherwise write through a nil writer or into the next request's Context. `sse.Stream` and `ai.Response` end themselves through this hook when a handler returns without `Close`.

```go
ticker := time.NewTicker(time.Second)
ctx.OnRelease(ticker.Stop)
```

`ctx.RunReleaseHooks()` is the host-side half: anything that acquires a Context from the pool itself must call it before releasing. It makes a bounded number of passes (8) and drops a hook that keeps re-arming itself, with a debug log line. `httptest.Do` runs the hooks before releasing the writer; `Pool.ForceRelease` drops them, so a handler that force-releases mid-chain cannot leave hooks for the next request.

## Source Files

- `context/context.go` - Main Context implementation, flow control, `OnRelease`/`RunReleaseHooks`
- `context/context_request.go` - Request reading (Header, Query, Params, Cookie)
- `context/context_response.go` - Response writing (SetHeader, SetContentType, cookies, files)
- `context/encoding/entries.go` - Generic Entries accessors used by Params/Query/Values
- `context/response_writer.go` - ResponseWriter interface (Unwrap)
