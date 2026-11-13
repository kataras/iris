# Configuration and Deployment

## Installation

```sh
go get github.com/kataras/iris/v14@latest
```

Module path is `github.com/kataras/iris/v14`. Requires Go 1.27+.

## Options Struct

The `iris.Options` struct configures the application. All fields are optional.

```go
type Options struct {
    // Application name and log prefix
    Name string `yaml:"Name"`

    // Logging level: "disable", "fatal", "error", "warn", "info" (default), "debug"
    LogLevel string `yaml:"LogLevel"`

    // Trusted domain (e.g., "example.com")
    Host string `yaml:"Host"`

    // Keep Run and Listen from exiting the process on failure: log at error
    // level and return the error instead. Default false: log at fatal level
    // and exit with status 1.
    DisableFatalRun bool `yaml:"DisableFatalRun"`

    // Router configuration
    Router RouterOptions `yaml:"Router"`

    // Request/response configuration
    Context ContextOptions `yaml:"Context"`
}
```

### Usage

```go
// Inline options
app := iris.New(iris.Options{
    Name:     "My API",
    LogLevel: "debug",
    Host:     "api.example.com",
})

// Defaults with edits
app := iris.New(iris.Defaults(func(opts *iris.Options) {
    opts.LogLevel = "debug"
    opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
}))

// Edit options after creation
app.EditOptions(func(opts *iris.Options) {
    opts.LogLevel = "error"
})

// Load YAML from the SERVER_CONFIG environment variable first, then a file.
var opts iris.Options
err := opts.Bind("config.yml")
```

### Options merge with the defaults

A partial `Options` literal configures only what it sets and inherits the rest from `iris.DefaultOptions`. `Configure` calls `applyDefaults`, which fills every unset field, delegating to `ContextOptions.FillZero` for the nested struct.

In v12 a partial literal **replaced** the whole configuration, so every unset field became its zero value and silently dropped `Charset`, `TimeFormat`, `PostMaxMemory`, the private-subnet list and the write error handler. That no longer happens.

Two rules govern what counts as "unset":

- A **non-nil empty** slice or map is an explicit "none" and is kept as-is. `RemoteAddrHeaders: []string{}` means trust nothing; omitting the field inherits the default.
- **Booleans are never filled**, since `false` is indistinguishable from unset.

A scalar whose zero value is meaningful has the same ambiguity, so force it per request instead:

```go
app.Use(func(ctx iris.Context) {
    ctx.Options().URLParamSeparator = "" // comma values stay a single value
    ctx.Next()
})
```

## RouterOptions

```go
type RouterOptions struct {
    // Server host with port (set automatically from ServerOptions.Addr)
    Host string

    // Disable automatic /path/ -> /path redirection
    DisablePathCorrection bool

    // Execute handler directly instead of redirect on path correction
    DisablePathCorrectionRedirection bool

    // Auto-correct typos in paths (e.g., /cont -> /contact)
    EnablePathIntelligence bool

    // Preserve encoded slashes in parameters
    EnablePathEscape bool

    // Convert all route paths to lowercase
    ForceLowercaseRouting bool

    // Support semicolons as query parameter separators
    AllowQuerySemicolons bool

    // Allow registering routes at runtime
    EnableDynamicHandler bool

    // Return 405 Method Not Allowed for unsupported methods
    FireMethodNotAllowed bool
}
```

## ContextOptions

Configures request and response handling for all routes. Also editable per request via `ctx.Options()`.

```go
type ContextOptions struct {
    // Request options.

    // Separator for multi-value query reads (ctx.Query().GetSlice). Defaults to ",".
    URLParamSeparator string

    // When true, ReadForm/ReadQuery/ReadBody return iris.ErrEmptyForm on empty form data.
    FireEmptyFormError bool

    // Time format for datetime parsing (e.g., CheckIfModifiedSince).
    // Defaults to "Mon, 02 Jan 2006 15:04:05 GMT".
    TimeFormat string

    // Maximum size of multipart form data. Defaults to 32MB (32 << 20).
    PostMaxMemory int64

    // Maximum bytes read from a request body. Defaults to 0, meaning NO LIMIT.
    BodyMaxMemory int64

    // Request header names used to determine the client's IP (behind proxies).
    // ctx.RemoteAddr() reads them in this order, each from right to left:
    // the first public address is the client's IP, without its port. A private
    // address is skipped. An entry that is not an address ("unknown", "unix:")
    // ends that header with no result, so the next header is read and at last
    // the address of the connection is used.
    // Defaults to empty (no "X-" headers are trusted).
    RemoteAddrHeaders []string

    // When the headers give no public address, use the first address from the
    // left of a header whose entries are all addresses, even if it is within
    // RemoteAddrPrivateSubnets. A header with an entry that is not an address
    // is read as with this option off: it gives nothing. Defaults to false.
    RemoteAddrHeadersForce bool

    // IP ranges considered private subnets. Defaults to common private ranges.
    RemoteAddrPrivateSubnets []netutil.IPRange

    // Headers that indicate a valid HTTPS request (see Context.IsSSL()).
    SSLProxyHeaders map[string]string

    // Headers that may contain a proxied hostname value (see Context.Host()).
    HostProxyHeaders map[string]bool

    // Validator called after request body decoding on ReadJSON, ReadForm,
    // ReadQuery and friends. Defaults to nil.
    // Interface: Validator { Struct(any) error }
    RequestValidator Validator

    // Response options.

    // Character encoding for rendering. Defaults to "utf-8".
    Charset string

    // Handler for errors that occur during writes (JSON, XML, Render, etc.).
    ErrorHandler ErrorHandler

    // Disables the error handler completely.
    DisableErrorHandler bool

    // Arbitrary developer-provided objects, accessed via opts.Get/Set.
    Objects map[string]any
}
```

There is no view configuration here. View engines are registered as middleware; see the `view` section of [middleware-reference.md](./middleware-reference.md).

### BodyMaxMemory

Unset (`0`) means unlimited, so this is the single most valuable option to turn on for a public service:

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{BodyMaxMemory: 10 * iris.MB},
})
```

It bounds `Body`, `ReadJSON`, `ReadJSONStream` and `ReadMultipartRelated`, and therefore everything built on them (`ReadXML`, `ReadYAML`, `ReadBody`, `ctx.BindJSON`). It also bounds **decompression** reads, which is what stops a compression bomb from a client sending `Content-Encoding: gzip`. Exceeding it fails with `context.ErrRequestEntityTooLarge`, which `rest/errors` maps to 413.

It is separate from `PostMaxMemory`, which only bounds multipart form parsing. For per-route limits use the `bodylimit` middleware.

### RequestValidator

Runs after the body has been decoded into a struct, on `ReadJSON`, `ReadMsgPack`, `ReadForm`, `ReadQuery`, `ReadParams`, `ReadHeader`, `ReadURL`, `ReadValues` and `UnmarshalBody`. The typed handlers of `controller/mvc/handler` run it too, on a request or param struct bound from a body, a form, the URL query, path parameters or headers. Any validator with a `Struct(any) error` method fits, so a third-party library plugs straight in:

```go
app := iris.New(iris.Options{
    Context: iris.ContextOptions{RequestValidator: validator.New()},
})
```

This is application-wide and reflection/tag driven. It is distinct from the per-type `Validate() error` hook that `ctx.BindJSON` calls, which is plain Go and lives on the request struct: see [rest-helpers.md](./rest-helpers.md). Use either or both; the per-type hook is preferred for new code because it needs no tags and produces canonical field errors.

## ServerOptions

Passed to `app.Run()` or used internally by `app.Listen()`.

```go
type ServerOptions struct {
    // Listen address ("host:port")
    Addr string

    // Network type: "tcp" (default), "tcp4", "tcp6", "unix", "unixpacket"
    Network string

    // Connection keep-alive period
    KeepAlive time.Duration

    // SO_REUSEPORT (SO_REUSEADDR on Windows) for multi-CPU scaling
    SocketSharding bool

    // Hosts the certificate is valid for. First entry is the root domain
    // (used by Let's Encrypt). Falls back to the app Host, then to Addr.
    Hosts []string

    // TLS configuration (mutually exclusive options below)
    TLSConfig   *tls.Config // Custom TLS config
    CertFile    string      // PEM certificate: file path or raw PEM content
    KeyFile     string      // PEM key: file path or raw PEM content
    Letsencrypt bool        // Automatic HTTPS via Let's Encrypt

    // Ngrok tunnel (cannot combine with TLS or Let's Encrypt)
    Ngrok *NgrokOptions

    // SSH reverse tunnel (cannot combine with TLS, Let's Encrypt or Ngrok)
    SSHTunnel *SSHTunnelOptions

    // Timeout configuration (server and handler)
    Timeout *TimeoutOptions

    // Suppress startup log
    DisableStartupLog bool

    // Disable CTRL+C interrupt handler
    DisableInterruptHandler bool

    // Custom log writer
    LogWriter io.Writer
}
```

Mutual exclusivity is enforced at startup: combining `TLSConfig` with `Letsencrypt`, `Ngrok`, `CertFile` or `KeyFile` panics, as does combining `Ngrok` or `SSHTunnel` with any other TLS option.

## Starting the Server

### Listen (Simple)

```go
app.Listen(":8080")
```

Blocks until the server stops and returns nil after a graceful shutdown, once that shutdown has finished: the requests in flight are answered, or its deadline has passed, and `app.Shutdown` has run its whole list of closers (the server's shutdown first, then `AddCloser`, `Deferrables`, the `Close` of a service or a worker). A `main` that ends with `Listen` therefore cuts no closer. The other way round is a deadlock: a closer or an interrupt function that waits for `Listen` or `Run` to return blocks for good, because they wait for it. A program that calls `Listen` or `Run` cannot reach the `server.Server` they start. It can reach the `http.Server` that `ServerOptions.OnServerCreated` hands out, and a `Shutdown` or `Close` called on that one is not the list: it runs no closer and releases `Run` the moment it is called, not when the requests in flight have been drained. A failed start (build error, port taken) is logged at fatal level and the process exits with status 1, so the call is a statement, not an `if err :=`. With `iris.Options{DisableFatalRun: true}` the failure is logged at error level and returned instead; use that when one process runs several servers or a test opens a real listener. The same rule applies to `Run`.

Equivalent to:

```go
app.Run(iris.ServerOptions{
    Addr:           ":8080",
    Network:        "tcp",
    KeepAlive:      3 * time.Minute,
    SocketSharding: true,
    // LogWriter defaults to the application's logger output.
})
```

### Run (Advanced)

```go
app.Run(iris.ServerOptions{
    Addr:           ":443",
    Network:        "tcp",
    KeepAlive:      3 * time.Minute,
    SocketSharding: true,
    CertFile:       "cert.pem",
    KeyFile:        "key.pem",
})
```

## TLS / HTTPS

### Certificate Files

```go
app.Run(iris.ServerOptions{
    Addr:     ":443",
    CertFile: "/path/to/cert.pem",
    KeyFile:  "/path/to/key.pem",
})
```

`CertFile` and `KeyFile` also accept raw PEM contents instead of file paths.

### Custom TLS Config

Prefer `CertFile`/`KeyFile` or `Letsencrypt`: the framework builds a TLS 1.3, HTTP/2-ready config for you. Bring your own `tls.Config` only when you need something those options cannot express (client certificates, cipher suites for legacy TLS 1.2 clients):

```go
app.Run(iris.ServerOptions{
    Addr: ":443",
    TLSConfig: &tls.Config{
        Certificates: []tls.Certificate{cert},
        // ... custom TLS settings
    },
})
```

A custom config is normalized, not served verbatim: `NextProtos` becomes `["h2", "http/1.1"]` when nil (a non-nil list is kept, with `"http/1.1"` ensured, so HTTP/2 is no longer lost silently) and `ServerName` is filled when empty. `MinVersion` is respected as-is: the framework-built configs use `tls.VersionTLS13`, so `MinVersion: tls.VersionTLS12` is a downgrade; set it only for clients that cannot speak TLS 1.3.

### Server Limits (Go 1.27)

`ServerOptions.Limits` tunes the standard server's request limits. Two Go 1.27 defaults make it worth knowing: requests repeating one header name beyond `http.DefaultMaxHeaderValueCount` are rejected with 400, and (since every TLS Iris server offers "h2") HTTP/2 responses follow RFC 9218 client-driven prioritization. The `LimitsOptions` type lives in the `core/server` package (no root-package alias):

```go
import "github.com/kataras/iris/v14/core/server"

app.Run(iris.ServerOptions{
    Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem",
    Limits: &server.LimitsOptions{
        MaxHeaderBytes:        512 * 1024, // 0 keeps Go's DefaultMaxHeaderBytes (1MB).
        MaxHeaderValueCount:   32,         // 0 keeps Go's DefaultMaxHeaderValueCount.
        DisableClientPriority: true,       // opt out of RFC 9218 h2 response scheduling.
    },
})
```

Set fields only to change a default; zero values keep the standard library's behavior.

### OnServerCreated Hook

`ServerOptions.OnServerCreated` runs with the standard `*http.Server` after the `Timeout` and `Limits` options are applied and before serving starts; returning an error aborts the startup. It is the escape hatch for any `http.Server` field Iris does not mirror as an option (including fields future Go releases add) and for TLS knobs on its `TLSConfig`, such as post-quantum key exchange (`tls.MLKEM1024`) or ML-DSA signature schemes. Go's default `CurvePreferences` already include the hybrid post-quantum key exchanges, which is why Iris deliberately leaves that field nil.

```go
app.Run(iris.ServerOptions{
    Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem",
    OnServerCreated: func(srv *http.Server) error {
        srv.TLSConfig.CurvePreferences = []tls.CurveID{tls.MLKEM1024, tls.X25519MLKEM768}
        return nil
    },
})
```

### Let's Encrypt (Automatic HTTPS)

`Letsencrypt` is a plain bool, not a struct:

```go
app.Run(iris.ServerOptions{
    Addr:        ":443",
    Hosts:       []string{"example.com"}, // optional, falls back to app Host or Addr
    Letsencrypt: true,
})
```

Behavior:
- Automatic certificate acquisition and renewal (14 days before expiration)
- Certificates cached in the `./letsencrypt` directory
- The domain is validated with the ACME tls-alpn-01 challenge, which the TLS listener itself answers: it offers `acme-tls/1` after `h2` and `http/1.1`. Let's Encrypt connects to port 443 for it, so port 443 of every host must reach this server. No HTTP-01 handler is served, and port 80 plays no part
- If `CertFile` and `KeyFile` are also set, the server presents the issued certificate and falls back to the pair when Let's Encrypt fails (useful for local testing). No log line says which of the two is being served

## Ngrok Tunnel

Expose local server to the internet via ngrok. ngrok is **opt-in**: the tunnel code lives in `core/server/ngrok`, which registers itself from `init()`, so add the blank import. Without it, a server with `Ngrok` set fails to start with `server: ngrok is configured but its listener is not linked in: add the import _ "github.com/kataras/iris/v14/core/server/ngrok"`. `ServerOptions.Ngrok` and `iris.NgrokOptions` are unchanged, so YAML configuration keeps working.

```go
import _ "github.com/kataras/iris/v14/core/server/ngrok"

app.Run(iris.ServerOptions{
    Addr: ":8080",
    Ngrok: &iris.NgrokOptions{
        AuthToken:   "your-ngrok-token",    // Or NGROK_AUTHTOKEN env var
        Region:      "us",
        Compression: true,                  // Skip if compression is already enabled in the app
        // Oauth: map[string]string{"google": "example.com"}, // optional OAuth gate
    },
})
```

## SSH Tunnel

Reverse SSH tunnel via serveo.net, localhost.run, or custom servers. The `SSHTunnelOptions` type lives in the `core/server` package (no root-package alias):

```go
import "github.com/kataras/iris/v14/core/server"

app.Run(iris.ServerOptions{
    Addr: ":8080",
    SSHTunnel: &server.SSHTunnelOptions{
        Domain:     "serveo.net",
        RemotePort: 80,
        LocalHost:  "localhost",
        LocalPort:  8080, // int, not string
        SSHKeyPath: "~/.ssh/id_rsa", // optional
        // User: "", ExtraArgs: []string{}, // optional
    },
})
```

Resulting command: `ssh -T [-i <SSHKeyPath>] -R <RemotePort>:<LocalHost>:<LocalPort> [extra args] [user@]<domain>`.

## Configuration Sources

### YAML File

```go
// Load options from YAML file
app := iris.New(iris.YAML("config.yml"))
```

config.yml:
```yaml
Name: "My API"
LogLevel: "info"
Host: "api.example.com"
Router:
  EnablePathIntelligence: true
  FireMethodNotAllowed: true
Context:
  RemoteAddrHeaders:
    - "X-Forwarded-For"
    - "X-Real-Ip"
```

### Environment Variables

```go
// Bind from SERVER_CONFIG env var (YAML format) or file
opts := &iris.Options{}
if err := opts.Bind("config.yml"); err != nil {
    log.Fatal(err)
}
app := iris.New(*opts)
```

The `Bind` method checks `SERVER_CONFIG` environment variable first (expects YAML), then falls back to the given filename.

### LoadConfiguration (the whole service, not just the framework)

`Options.Bind` and `iris.YAML` read the framework's own options. A service's configuration nests them inside its own struct, and `iris.LoadConfiguration` is the loader for that shape:

```go
type Configuration struct {
    iris.BaseConfiguration `yaml:",inline"`

    DatabaseURL string `yaml:"DatabaseURL"`
}

c, err := iris.LoadConfiguration[Configuration]("MY_SERVICE_CONFIG", "configuration.yml")
if err != nil {
    log.Fatal(err)
}

app := iris.New(c.Iris)
app.Listen(c.Addr())
```

```go
func LoadConfiguration[T any](envKey, filename string) (T, error)
```

- The environment variable wins over the file, as in `Options.Bind`.
- The variable may hold **either the YAML document or a path to it**: a value naming an existing file is read as a file, anything else is decoded as the document, so one binary works whether the configuration is mounted or passed inline. `Options.Bind` never does this; it is hardcoded to `SERVER_CONFIG` and always reads a document.
- If `T` (or `*T`) has a `BindEnv()` method it runs **after** decoding, so environment overrides win over the document.

`iris.BaseConfiguration` is the part every service shares, ready to embed inline:

| Field | YAML | Meaning |
|-------|------|---------|
| `Host` | `Host` | The interface to listen on; empty means every interface. **Not** `Iris.Host` (the trusted domain for subdomain routing); `Addr` never reads that one |
| `Port` | `Port` | The TCP port; zero means 8080 |
| `AllowOrigin` | `AllowOrigin` | As `Builder.AllowOrigin` accepts it |
| `EnableCompression` | `EnableCompression` | As `Builder.Compression` accepts it |
| `RequestLog` | `RequestLog` | As `Builder.LogRequests` accepts it |
| `Iris` | `Iris` | The framework `Options`, as `iris.New` accepts them |

Its `BindEnv` reads `HOST`, `PORT`, `HTTP_PLATFORM_PORT` and `ALLOW_ORIGIN`. `HTTP_PLATFORM_PORT` wins over `PORT`, because a hosting platform that picks the port passes it there while `PORT` may still carry the value baked into the image; a non-numeric port is ignored rather than turned into zero. `Addr()` returns `host:port` through `net.JoinHostPort`, so an IPv6 host is bracketed.

There is deliberately no `app.Serve(host, port)`: `Listen` and `Run` stay the only entry points.

### Configurator Pattern

```go
// Custom configurator
type MyConfig struct{}

func (c MyConfig) Configure(app *iris.Application) {
    app.EditOptions(func(opts *iris.Options) {
        opts.LogLevel = "debug"
    })
}

app := iris.New(MyConfig{})

// Function-based configurator
app := iris.New(iris.ConfiguratorFunc(func(app *iris.Application) {
    app.Logger().SetLevel("debug")
}))
```

## Graceful Shutdown

Unless `ServerOptions.DisableInterruptHandler` is set, the server handles SIGINT and SIGTERM itself. A signal runs the interrupt list in the order its entries were registered and then ends the process: `os.Exit(0)`, or status 1 after a `Received an error during shutdown` line when an entry returned an error. The server's own entry on that list stops accepting connections and drains the requests in flight for up to `ShutdownOnInterruptTimeout`; the other entries are the functions given to `iris.RegisterOnInterrupt`, and one registered before `Listen` runs before the server's entry, while the server still serves. `Listen` and `Run` return at the moment of that exit, when every entry has returned, so statements placed after `Listen` race the exit and are no place for cleanup. The server's entry shuts down the server and nothing else: the closers (`Deferrables`, `AddCloser`, the `Close` methods `Services` found) run only when something calls `app.Shutdown(ctx)`, and with nothing registered a signal drains the server and exits without them.

To handle the signal yourself, turn the built-in handler off. Left on, its `os.Exit(0)` ends the process as soon as the server has drained and cuts the closers `app.Shutdown` has not run yet:

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
)

app := iris.New()
// ... register routes ...

go func() {
    app.Run(iris.ServerOptions{Addr: ":8080", DisableInterruptHandler: true})
}()

// Wait for interrupt signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Graceful shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
app.Shutdown(ctx)
```

Or use the built-in interrupt handler:

```go
iris.RegisterOnInterrupt(func() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    app.Shutdown(ctx)
})

app.Listen(":8080")
```

Six things to know about the second form:

- The function is registered before the server's own interrupt entry, so it runs first: its context is the deadline of the drain and is passed to every closer that takes one. `ShutdownOnInterruptTimeout` decides nothing then.
- `iris.RegisterOnInterrupt(app.Shutdown)` is the short form, and it is bounded. The signal handler gives a function that takes a context one whose deadline is the `Timeout.ShutdownOnInterruptTimeout` of the running server: 15 seconds from `iris.Timeout()` or a nil `Timeout`, no deadline for a zero or negative value. With more than one server in the process the one that asked for the most wins: the largest value of all of them, and no deadline once one of them asked for none. A value is never taken back, so a server that has stopped still counts. A server started with `DisableInterruptHandler` hands its option to nobody: the functions get no deadline from it. A request still in flight at the deadline is cut, a stream that watches only its client included.
- That deadline is one for the whole interrupt list and it is counted from the signal. When the drain uses all of it, the closers after the server get a context that has expired already: the `Stop` of a `Worker` is called with no time left and its `Run did not return` error comes back from `app.Shutdown`. The hand-written form above has the same shape, one context for the drain and the closers. Size the deadline for both.
- `Listen` and `Run` return once `app.Shutdown` has run its whole list, so `Listen` is the last statement of `main` and no closer is cut. Put no `select {}` after it: after a shutdown that is not a signal (`app.Shutdown` called from your own code) `main` would hang there. After a signal the process exits when the functions on the interrupt list are done, so statements after `Listen` may not run.
- A closer or an interrupt function must not wait for `Listen` or `Run` to return (on a channel that `main` closes after `Listen`, for example). They wait for it, so it blocks for good: the Go runtime stops the program with `all goroutines are asleep - deadlock!` when nothing else is running, and it hangs otherwise.
- An interrupt function that returns an error ends the process with status 1, after a `Received an error during shutdown` line. `iris.RegisterOnInterrupt(app.Shutdown)` returns the joined error of the closers, so a closer that fails on a signal shows in the exit status. The `func()` form above drops the error of `app.Shutdown` and exits with 0.

From inside a handler, stop the server with `go app.Shutdown(context.Background())` or pass a context with a deadline. Called in place with a context that has no deadline, the handler's own `ctx` included, `Shutdown` waits for its own request, and `Run` waits with it. `go app.Shutdown(ctx)` with the handler's `ctx` is not a way out: it closes the other connections at once.

### Closers and shutdown errors

`app.Shutdown(ctx)` runs all registered closers (the HTTP server's shutdown is prepended automatically on `Run`), and `Run` and `Listen` return once it has run the last of them. Errors from individual closers are aggregated with `errors.Join`, so callers can match each underlying error via `errors.Is` / `errors.As`:

```go
if err := app.Shutdown(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // at least one closer timed out
    }
}
```

`app.AddCloser(v)` accepts these types (any other type panics at registration time):

| Type | Notes |
|------|-------|
| `io.Closer` | `Close() error` |
| `closer.Closer` | `CloseWithContext(context.Context) error` |
| `func()` | Simple cleanup |
| `func() error` | Cleanup with error |
| `func(context.Context)` | Context-aware cleanup |
| `func(context.Context) error` | Context-aware cleanup with error |

## Deferrables (Builder)

Register cleanup functions for graceful shutdown:

```go
iris.NewBuilder().
    // ...
    Deferrables(
        db,                            // Objects with Close() method
        redisClient,                   // Auto-detected io.Closer
        func() { worker.Stop() },      // Simple function
        func() error { return cache.Flush() }, // Function with error
    ).
    // ...
```

The builder also auto-detects `Close()` methods on objects passed to `Services()` and registers them as closers.

## Workers (Builder)

A value passed to `Services()` that implements `iris.Worker` is registered as a dependency **and** started inside the builder's lifecycle: a scheduler, a queue consumer, a cache refresher.

```go
type Worker interface {
    Run(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

- `Run` starts in its own goroutine **after** every `iris.Initializer` has returned, with a context cancelled at shutdown. An error returned before shutdown is logged with the worker's type name and nothing else: a failed worker does not take down a server that can still answer every request which does not need it.
- `Stop` runs once, in registration order with the closers (a `Worker` that is also a closer is stopped and waited for before its own `Close`), and `Shutdown` then waits for `Run` to return. A `Run` that does not return in time is logged and reported as a shutdown error.
- A value that is both a `Worker` and a closer is stopped, waited for, then closed, in that order: closing the connection a running `Run` is still reading from is the crash that ordering avoids.
- The budget is the `Application.Shutdown` context when that one carries a deadline, otherwise ten seconds. Ten seconds is **per worker**, not per shutdown, because the closers run one after another; pass a deadline to `Shutdown` to bound the whole chain.
- A shutdown that arrives before the application is built cancels the worker and keeps a later `Build` from starting it.
- Only a `Worker` passed to `Services` as a value is started; one returned by a constructor is not.

## Development

There is no built-in CLI and no live reload inside the framework. Run the application with `go run .` and restart it on change, under whatever file watcher you already use. Nothing is read from the working directory at startup to enable a development mode.

## Build Info

```go
// Print version and build info
iris.PrintBuildInfo(os.Stdout)
// Output:
// Iris Version: 14.0.0
// Build Time:     2024-01-15T10:30:00Z
// Build Revision: abc123def

// Access build info
revision := iris.BuildRevision
buildTime := iris.BuildTime
version := iris.Version // "14.0.0"
```

## Logger Configuration

```go
app.Logger().SetLevel("debug")    // Set log level
app.Logger().SetOutput(file)      // Set output writer
app.Logger().AddOutput(os.Stdout) // Add additional output

// Log levels: "disable", "fatal", "error", "warn", "info", "debug"

// Check debug mode
if app.IsDebug() {
    // debug-only logic
}
```

## Timeout Configuration

`iris.Timeout()` returns a `*TimeoutOptions` builder (defaults: 15s shutdown-on-interrupt timeout, default handler timeout message):

```go
app.Run(iris.ServerOptions{
    Addr: ":8080",
    Timeout: iris.Timeout().
        WithReadTimeout(30 * time.Second).
        WithHandlerTimeout(60 * time.Second).
        WithShutdownOnInterruptTimeout(10 * time.Second),
})
```

Builder methods on `*TimeoutOptions`:

| Method | Sets |
|--------|------|
| `WithServerTimeout(d)` | Total lifecycle timeout; auto-calculates read-header, write, and idle timeouts |
| `WithReadTimeout(d)` | `ReadTimeout` (headers + body) |
| `WithReadHeaderTimeout(d)` | `ReadHeaderTimeout` |
| `WithWriteTimeout(d)` | `WriteTimeout` |
| `WithIdleTimeout(d)` | `IdleTimeout` |
| `WithHandlerTimeout(d, htmlBody ...string)` | Wraps the router in `http.TimeoutHandler` with optional custom timeout message |
| `WithShutdownOnInterruptTimeout(d)` | Max wait for graceful shutdown on CTRL+C, and the deadline of the context `iris.RegisterOnInterrupt` functions receive. Zero or negative sets no limit |

The 15 seconds come from `iris.Timeout()` and from a nil `ServerOptions.Timeout`. A `&iris.TimeoutOptions{...}` literal, or a configuration file whose `Timeout` section leaves the field out, carries zero: no deadline on interrupt, so the server waits for the last request however long it takes. Set the field when the drain must be bounded. The same value bounds `iris.RegisterOnInterrupt(app.Shutdown)`, as the Graceful Shutdown section above describes.

## Source Files

- `options.go` - Options struct, Configurator, YAML loading, Bind
- `iris_configuration.go` - `LoadConfiguration`, `BaseConfiguration`, `BindEnv`, `Addr`
- `iris_builder.go` - the builder chain, `Services`, `Worker` and its start/stop wiring
- `iris.go` - Application, Listen, Run, Build, Shutdown, AddCloser, PrintBuildInfo
- `core/server/server.go` - ServerOptions, TLS, Let's Encrypt, Ngrok, SSH tunnels
- `core/server/ngrok/ngrok.go` - the ngrok listener, linked in by the blank import
- `core/server/ngrok_registry.go` - `RegisterNgrokListener`, `RegisteredNgrokListener`, the hook the blank import uses
- `core/server/timeout.go` - TimeoutOptions builder
- `core/server/closer/closer.go` - Closers, errors.Join aggregation
- `core/router/router_options.go` - RouterOptions
- `context/context_options.go` - ContextOptions
