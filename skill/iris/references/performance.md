# Performance

Measure before optimizing, and know what the framework already does so you do not rebuild it. Most Iris performance work is about not materializing whole datasets in memory, not about making the router faster.

## What Iris Already Does

Understanding this list prevents the most common wasted effort.

**Contexts and writers are pooled.** `*Context` and the response writer come from a `sync.Pool` and are recycled after every request. The parameter and value stores (`ctx.Params()`, `ctx.Values()`) are **slices, not maps**, reset with their capacity retained. A request that reads three path parameters allocates nothing for them after warmup.

The hard rule that follows: **never retain `iris.Context` past the handler's return.** A goroutine holding it reads a recycled context belonging to a different request. When background work needs request values, take a standard context:

```go
go audit.Record(ctx.WithoutCancel(), order) // correct
go audit.Record(ctx, order)                 // wrong: ctx is recycled underneath
```

Goroutines are not the only consumers that keep a context. An outbound HTTP client does too: the transport holds the request context for cancel propagation after the response arrives. Hand it `ctx.StdContext()`, the request's own standard context, rather than `ctx` itself.

**Routing is a prefix tree, resolved at build time.** `Build()` compiles registered routes into prefix trees, one per HTTP method and subdomain. Typed parameters are validated by plain Go evaluator functions selected at registration. There is no regex engine and no route-list scan at request time, so route count has almost no effect on per-request latency.

**Typed accessors are `strconv`, not reflection.** `ctx.Params().GetInt("id")` and `ctx.Query().GetString("q")` parse directly. Missing or unparseable values return documented zeros rather than errors: `""` for strings, `-1` for a missing int and `0` for an unparseable one.

**`ctx.JSON` streams.** It encodes into the response writer through a `json.Encoder` rather than building a byte slice first, so a large response does not have a whole-body allocation.

## Measuring

### Benchmarking handlers

`*iris.Application` is an `http.Handler` once built, so a handler benchmark needs no network:

```go
func BenchmarkGetUser(b *testing.B) {
    app := newApp()
    if err := app.Build(); err != nil { // once, outside the loop
        b.Fatal(err)
    }
    req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
    b.ReportAllocs()
    for b.Loop() {
        w := httptest.NewRecorder()
        app.ServeHTTP(w, req)
    }
}
```

For POST benchmarks, rebuild the body each iteration: a drained `io.Reader` makes every iteration after the first measure the empty-body path.

### Per-request cost

`middleware/httpcost` measures time, memory and CPU per request and turns them into one number, useful for finding the expensive endpoints and for usage billing.

```go
app.Use(httpcost.New(httpcost.DefaultOptions))
```

The metric lands on the context **after** the chain returns, so read it from an outer middleware that calls `ctx.Next()` first (or from an access-log field setter):

```go
app.Use(func(ctx iris.Context) {
    ctx.Next() // let the httpcost middleware and the handler run
    if m, ok := httpcost.Get(ctx); ok {
        logger.Infof("%s cost=%.4f cpu=%s", ctx.Path(), m.Cost, m.CpuTime)
    }
})
```

`httpcost.Compute(func())` measures an arbitrary block with the same formula. Full options in [observability.md](./observability.md).

## Response Efficiency

### Compression

```go
app.UseRouter(compress.Handler) // or the builder's Compression(true)
```

Negotiates gzip, deflate, brotli, snappy, s2 and zstd from `Accept-Encoding`, and transparently **decompresses request bodies** carrying `Content-Encoding`. That inbound path is why `BodyMaxMemory` matters: it bounds the decompressed read, which is what stops a compression bomb.

Compression trades CPU for bytes. On a fast internal network in front of already-compressed payloads (images, video) it is a loss; register it per group rather than at the root when only some responses are text.

### JSON encoding options

```go
type JSON struct {
    Indent         string
    Prefix         string
    EscapeHTML     bool // restores the old encoding/json <, >, & escaping.
    ASCII          bool
    Secure         bool
    MarshalOptions []json.Options // extra encoding/json/v2 options, applied last.
}
```

Every response is encoded through the standard library's `encoding/json/v2` into a pooled buffer (framework defaults: sorted map keys, duplicate names and invalid UTF-8 tolerated on output). Types implementing `json.MarshalerTo` stream directly. `Indent` is for humans; never ship it in a hot API.

```go
ctx.JSON(v)                              // fastest
ctx.JSON(v, iris.JSON{Indent: "  "})     // debug/browser friendly
```

`ctx.JSONOrError` shares the same encoder.

### Streaming large responses

`ctx.StreamWriter` sends a response in chunks without holding it in memory. The contract: return `nil` to flush what you wrote and be called again; return a non-nil error (conventionally `io.EOF`) to finish. On client disconnect it returns the request context's cancellation error.

```go
func exportOrders(ctx iris.Context) {
    ctx.SetContentType("text/csv")
    rows := svc.OrderCursor(ctx)
    err := ctx.StreamWriter(func(w io.Writer) error {
        batch, more := rows.Next(500) // hundreds of rows per call, not one
        for _, r := range batch {
            fmt.Fprintf(w, "%d,%s,%.2f\n", r.ID, r.Customer, r.Total)
        }
        if !more {
            return io.EOF
        }
        return nil
    })
    if err != nil && !errors.Is(err, io.EOF) {
        ctx.Application().Logger().Warnf("export aborted: %v", err)
    }
}
```

Batch size is the whole game: one row per callback means one flush per row. The manual equivalent is `ctx.Write` + `ctx.ResponseWriter().Flush()` with `ctx.IsCanceled()` checks.

`StreamWriter` is a poll loop, which suits a writer that blocks until it has something (a cursor, a ticker, a queue read). For an **event stream**, use the `sse` package instead: it is a push API, and it handles the response-writer buffering, the absolute `WriteTimeout`, heartbeats and framing that `StreamWriter` leaves to you. See [sse.md](./sse.md).

Two of the bundled middleware buffer a streamed body and must step aside for it: `middleware/compress` writes into a compressor, and the `middleware/recorder` that `accesslog` installs whenever it reads the response body (which `LogRequests(true)` turns on) writes into a `bytes.Buffer`. `ctx.DisableResponseBuffering()` removes both for the current response, before the first write; `sse.Open` calls it on the first `Send`. The recorder also opts out of streaming media types on its own now, so a hand-rolled `text/event-stream` handler is no longer de-streamed by an access log.

## Request Efficiency

### Cap request bodies

```go
app := iris.New(iris.Options{Context: iris.ContextOptions{BodyMaxMemory: 10 * iris.MB}})
```

Overflow fails with `context.ErrRequestEntityTooLarge`, mapped to 413. `bodylimit.New(2 * iris.MB)` scopes a tighter limit to one group.

### Stream large arrays in

`ctx.ReadJSONStream` decodes a JSON array element by element instead of unmarshalling the whole thing:

```go
func importOrders(ctx iris.Context) {
    batch := make([]Order, 0, 500)
    err := ctx.ReadJSONStream(func(decode iris.DecodeFunc) error {
        var o Order
        if err := decode(&o); err != nil {
            return err
        }
        batch = append(batch, o)
        if len(batch) == cap(batch) {
            if err := svc.InsertBatch(ctx, batch); err != nil {
                return err
            }
            batch = batch[:0] // reuse the backing array
        }
        return nil
    })
    if err != nil {
        errors.HandleRequestError(ctx, err)
        return
    }
    if len(batch) > 0 {
        if err := svc.InsertBatch(ctx, batch); err != nil {
            errors.Internal.LogErr(ctx, err)
            return
        }
    }
    ctx.StatusCode(iris.StatusAccepted)
}
```

### Paginate instead of loading everything

`ctx.Paginated` reads `?page=` and `?size=` (plus an optional JSON filter body) and calls a collector that returns exactly one page. The database returns one page and the handler allocates one page.

```go
func (api *OrdersAPI) list(ctx iris.Context) {
    ctx.Paginated(func(reqCtx context.Context, opts iris.PageOptions, f OrderFilter) ([]Order, int64, error) {
        return api.orders.Page(reqCtx, opts, f)
    })
}
```

It returns a `bool` like the other response helpers: `false` means a response was already written.

## Concurrency in Handlers

### Background work

```go
func placeOrder(ctx iris.Context) {
    in, ok := ctx.BindJSON[NewOrder]()
    if !ok {
        return
    }
    order, err := svc.Place(ctx, in)
    if err != nil {
        errors.Internal.LogErr(ctx, err)
        return
    }
    // Must finish even if the client hangs up. Values kept, cancellation dropped.
    go notify.OrderPlaced(ctx.WithoutCancel(), order)
    ctx.Created(order, nil)
}
```

### Bounded worker pools

An unbounded `go` per request turns a traffic spike into an out-of-memory kill. Bound the pool and shed load explicitly, so the client gets a fast 503 instead of an unbounded wait:

```go
type Enricher struct{ jobs chan Job }

func (e *Enricher) Submit(j Job) bool {
    select {
    case e.jobs <- j:
        return true
    default:
        return false // queue full
    }
}

func handler(ctx iris.Context) {
    if !enricher.Submit(job) {
        errors.Unavailable.Message(ctx, "the enrichment queue is full")
        return
    }
    ctx.StatusCode(iris.StatusAccepted)
}
```

Register the pool through `Services(...)` so its `Close()` drains on shutdown, and warm it in `Init(ctx) error` if it needs priming. See [architecture-and-patterns.md](./architecture-and-patterns.md).

## Caching

Response and application caching is the largest lever for read-heavy APIs and has its own reference: [caching.md](./caching.md).

## Source Files

- `context/context.go`, `context/context_request.go` - Pooling, typed accessors, `ReadJSONStream`, `WithoutCancel`
- `context/context_response.go` - `StreamWriter`, `JSON` options, `DisableResponseBuffering`
- `sse/`, `ai/` - Event streams and model streams (see [sse.md](./sse.md), [ai.md](./ai.md))
- `core/router/` - Route tree construction at build time
- `middleware/compress/` - Codec negotiation and request-body decompression
- `middleware/httpcost/` - Per-request cost metric
- `context/page.go`, `core/pkg/pagination/pagination.go` - `Paginated`, `PageOptions`, `Page[T]`
- `httptest/` - Test helpers (see [testing-guide.md](./testing-guide.md))
