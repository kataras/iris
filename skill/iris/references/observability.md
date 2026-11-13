# Observability

Access logs, correlation IDs, application logging, live process stats, profiling, per-request cost and build information. Compact per-package API listings are in [middleware-reference.md](./middleware-reference.md); this file is how to compose them into a production stack and which behaviors surprise people.

## Structured Access Logging

`middleware/accesslog` writes one record per request. Build it once at startup and register it on `UseRouter` so 404s, 405s and CORS preflights are logged too (plain `Use` on a child router only sees matched routes).

```go
func newAccessLog() *accesslog.AccessLog {
    ac := accesslog.File("./access_log.json")
    ac.Async = true               // do not block the response on disk I/O
    ac.RequestBody = false        // bodies hold credentials and PII
    ac.ResponseBody = false
    ac.BytesReceivedBody = true   // sizes without contents: cheap and safe
    ac.BytesSentBody = true
    ac.SetFormatter(&accesslog.JSON{HumanTime: true})
    ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
        f.Set("request_id", requestid.Get(ctx))
    })
    return ac
}

func main() {
    ac := newAccessLog()
    defer ac.Close() // flushes the async buffer

    app := iris.New()
    app.UseRouter(requestid.New(requestid.ServerOnlyGenerator))
    app.UseRouter(ac.Handler)
    // ...
}
```

With the builder, pass it to `LogRequestsWith` at the `LogRequests` position and the closer is registered for you:

```go
builder.LogRequestsWith(newAccessLog())
```

`LogRequests(true)` installs the framework's own tuned access log instead. Use one or the other.

**It no longer de-streams a streaming response.** `accesslog` installs a `middleware/recorder` whenever it reads the response body, and the recorder used to forward its buffer only when the handler had set `Transfer-Encoding: chunked`, which no Server-Sent Events handler does. So `LogRequests(true)` silently turned every event stream into one lump on return. The recorder now steps aside for a streaming media type and honours `Disable()`, and `sse.Open` removes it explicitly on the first `Send`. An access log line for such a response carries no body size, which is the correct trade.

A handler that flushes behind that recorder without streaming, such as an `httputil.ReverseProxy` mounted with `iris.FromStd` whose upstream sends no `Content-Length`, keeps its status code and headers: the recorder hands them to the writer underneath before the flush, and the body stays recorded for the log line.

**Watch the defaults.** `accesslog.New()` already sets `BytesReceivedBody`, `BytesSentBody` and `KeepMultiLineError` to true, and `PanicLog` to `accesslog.LogHandler`. Setting those to their existing values is a no-op; `KeepMultiLineError = false` is the actual change (it collapses a multi-line error into one line, which is what you want for line-oriented log shipping).

Constructors: `accesslog.File(path)` (buffered), `accesslog.FileUnbuffered(path)`, `accesslog.New(w io.Writer)`.

### The record

The JSON formatter emits `timestamp` (unix ms), `latency`, `code`, `method`, `path`, `ip`, `query`, `params`, `fields`, `request`, `response`, `bytes_received`, `bytes_sent`. The default text format is pipe-separated:

```
Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
```

Formatters: `&accesslog.JSON{Indent, HumanTime}`, `&accesslog.CSV{Header}`, `&accesslog.Template{Text}` (fields `.Now`, `.TimeFormat`, `.Code`, `.Method`, `.Path`, `.Latency`). Custom ones implement `Formatter`:

```go
type Formatter interface {
    SetOutput(dest io.Writer)
    Format(log *accesslog.Log) (bool, error)
}
```

### Custom fields

`AddFields` runs per request at startup registration; `GetFields` sets a field from inside a handler. Both land in the record's `fields` object.

```go
func (api *TenantsAPI) get(ctx iris.Context) {
    accesslog.GetFields(ctx).Set("tenant", tenantOf(ctx))
    // ...
}
```

### Skipping noise, or logging only failures

```go
app.Get("/healthz", accesslog.SkipHandler, healthz) // route middleware
app.Use(func(ctx iris.Context) {                    // conditional
    if strings.HasPrefix(ctx.Path(), "/assets/") {
        accesslog.Skip(ctx)
    }
    ctx.Next()
})

ac.OnlyFailures = true // errors-only log
```

### Shutdown and rotation

`Close()` flushes and stops the async worker; `CloseContext(ctx)` bounds the wait. Rotation is writer substitution, since the middleware only needs an `io.Writer`:

```go
ac := accesslog.New(&lumberjack.Logger{Filename: "access.log", MaxSize: 100, MaxBackups: 7})
```

With an external rotator that truncates in place, call `ac.TruncateFile(0)` after it runs.

### Live tail

`Broker` fans records out to in-process listeners. Take the broker **before** the server starts. Pushing those records to a browser is `sse` on the other end of the listener (see [sse.md](./sse.md)).

```go
broker := ac.Broker()
go func() {
    ln := broker.NewListener()
    defer broker.CloseListener(ln)
    for log := range ln {
        if log.Code >= 500 {
            alert(log)
        }
    }
}()
```

In a builder application put that loop in an `iris.Worker` and pass it to `Services` instead: it starts after the initializers, `Run`'s context is cancelled at shutdown and `Shutdown` waits for it. See [configuration-and-deployment.md](./configuration-and-deployment.md#workers-builder).

## Request Correlation

```go
app.UseRouter(requestid.New(requestid.ServerOnlyGenerator))
id := requestid.Get(ctx) // also on the response as X-Request-Id
```

The default generator honors a client-supplied `X-Request-ID`, which is convenient for tracing across services you control and a log-forgery vector from the open internet. Use `ServerOnlyGenerator` on a public edge. Register it before the access log so the field setter can read it.

## Application Logging

`app.Logger()` is a `*golog.Logger`, also reachable as `ctx.Application().Logger()`.

```go
app.Logger().SetLevel("info")       // fatal | error | warn | info | debug
app.Logger().SetFormat("json")
app.Logger().SetTimeFormat(time.RFC3339)
app.Logger().AddOutput(auditFile)   // in addition to stderr

app.Logger().Errorf("charge %s failed: %v", id, err)
app.Logger().Info("worker pool warmed", golog.Fields{"size": 8})
```

Handler-level error logging should go through `rest/errors` instead, so the client gets a canonical body and the server keeps the detail: `errors.Internal.LogErr(ctx, err)`. See [error-handling.md](./error-handling.md).

## The Monitor Dashboard

`middleware/monitor` serves a live process/host dashboard and a JSON stats endpoint.

```go
m := monitor.New(monitor.Options{
    RefreshInterval:     2 * time.Second, // the default; see below
    ViewRefreshInterval: 2 * time.Second,
    ViewTitle:           "Service Monitor",
})
iris.RegisterOnInterrupt(m.Stop)

mon := app.Router("/monitor", basicauth.New(adminOpts))
mon.Get("/", m.View)   // the HTML dashboard
mon.Post("/", m.Stats) // the JSON the dashboard polls
```

Two things to get right:

1. **`View` and `Stats` must sit on the same path**, differing only by method. The dashboard polls its own URL with POST.
2. **`RefreshInterval` defaults to `monitor.DefaultRefreshInterval` (2 seconds) when unset.** Pass your own value to change the collector's tick.

Read the numbers directly with `m.Holder.GetStats()`. `monitor.Stats` carries `PIDCPU`, `PIDRAM`, `PIDConns`, `OSCPU`, `OSRAM`, `OSTotalRAM`, `OSLoadAvg`, `OSConns`, also published as the `expvar` gauges `pid_cpu`, `pid_ram`, `pid_conns`, `os_cpu`, `os_ram`, `os_total_ram`, `os_load_avg`, `os_conns`.

## Profiling with pprof

`pprof.New()` returns a routing controller, so mount it as one and put auth in front:

```go
app.Router("/debug/pprof", basicauth.New(adminOpts), pprof.New())
```

Serves heap, goroutine, allocs, block, mutex, threadcreate, cmdline, profile and trace (`?seconds=N` on trace and profile). **Never expose this publicly**: the profiles reveal source paths, memory contents and enough timing to help an attacker, and a long CPU profile is itself a load generator.

## Per-Request Cost

```go
app.Use(httpcost.New(httpcost.DefaultOptions)) // 0.01/s, 0.001/MB, 0.02/CPU
```

`Options` is `{CostPerSecond, CostPerMB, CostPerCPU float64}`. `Metric` carries `StartTime` (an `types.ISO8601`, not a `time.Time`), `Duration`, `MemoryUsed`, `NetworkUsed`, `CpuTime` and `Cost`.

The metric is written **after** the chain returns, so `httpcost.Get(ctx)` inside your handler finds nothing. Read it from an outer middleware that calls `ctx.Next()` first, or from an access-log field setter:

```go
ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
    if m, ok := httpcost.Get(ctx); ok {
        f.Set("cost", m.Cost)
        f.Set("cpu_time", m.CpuTime.String())
    }
})
```

`httpcost.Compute(func())` applies the same formula to an arbitrary block. Treat the memory and CPU numbers as indicative: they are sampled around the handler, not exact per-request accounting.

## Server-Timing

Exposes phase timings to browser devtools through the `Server-Timing` header.

```go
app.Use(servertiming.Handler) // records the total as the "app" metric

func (api *ReportsAPI) build(ctx iris.Context) {
    servertiming.Start(ctx, "db", "query rows")
    rows, err := api.reports.Rows(ctx)
    servertiming.End(ctx, "db")
    // ...
}
```

Every `Start` needs its `End`, or the metric never lands. The header is written before the body, so timings recorded after the first write are lost. Metric names reach the browser: do not encode internal hostnames or query text in them.

## Health Checks and Build Information

The builder's `Health(true, env, developer)` step publishes a health endpoint carrying the environment and a developer contact. `NoHealth()` omits it. Standalone, `middleware/modrevision` does the same job:

```go
app.Get("/health", modrevision.New(modrevision.Options{
    ServerName:     "my-service",
    Env:            "production",
    Developer:      "kataras",
    CloudDetection: true,
}))
```

It reads `core/pkg/buildinfo`, which surfaces the `vcs.revision` and `vcs.time` the Go toolchain stamps into the binary, so the endpoint reports the exact commit without a build-time `-ldflags` dance.

Liveness and readiness are different questions: liveness should stay dependency-free (if it fails, the orchestrator restarts you), while readiness may check the database. Do not wire a dependency check into the endpoint an orchestrator uses to decide whether to kill the process.

Health endpoints disclose environment and contact detail; gate or drop them on a public edge. See [security-hardening.md](./security-hardening.md).

## Third-Party Metrics and Tracing

Prometheus, OpenTelemetry and friends need no Iris-specific integration. Their handlers are `http.Handler` values and their instrumentation is ordinary middleware:

```go
app.Get("/metrics", iris.FromStd(promhttp.Handler()))
```

## Source Files

- `middleware/accesslog/` - `AccessLog`, formatters, `Fields`, `Broker`
- `middleware/requestid/requestid.go` - `New`, `Get`, `DefaultGenerator`, `ServerOnlyGenerator`
- `middleware/monitor/monitor.go` - `Options`, `Stats`, `View`, `Holder`
- `middleware/pprof/`, `middleware/servertiming/`, `middleware/httpcost/` - Profiling, phase timings, cost
- `middleware/modrevision/`, `core/pkg/buildinfo/` - Build and revision reporting
- `iris_builder.go` - `LogRequests`, `LogRequestsWith`, `Health`, `NoHealth`
- `middleware/recorder/response.go` - `Disable`, the streaming media-type opt-out (see [sse.md](./sse.md))
