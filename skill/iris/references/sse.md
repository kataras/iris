# Server-Sent Events

Package `github.com/kataras/iris/v14/sse`. Transport only, both directions: a push-based response stream, and a tolerant reader for consuming somebody else's. It ships no application vocabulary, no replay store and no pub/sub fan-out. For AI model streams built on top of it, see [ai.md](./ai.md).

Use it instead of hand-rolling `text/event-stream` with `ctx.StreamWriter` or `fmt.Fprintf` + `Flush`. `ctx.StreamWriter` is still correct for a chunked export or a ticker-driven poll loop; its godoc now points here for event streams.

## Writer

```go
import (
    "github.com/kataras/iris/v14/rest/errors"
    "github.com/kataras/iris/v14/sse"
)

func chat(ctx iris.Context) {
    stream, err := sse.Open(ctx)
    if errors.HandleError(ctx, err) {
        return // ErrNotStreamable renders as a 500: nothing written yet.
    }
    defer stream.Close(nil) // writes the terminal frame on success too.

    for token := range tokens {
        if err := stream.Send(sse.Event{Name: "token", Data: []byte(token)}); err != nil {
            return // the client is gone.
        }
    }
}
```

| Symbol | Signature | Notes |
|--------|-----------|-------|
| `sse.Open` | `Open(ctx *context.Context, opts ...Options) (*Stream, error)` | Writes nothing. Fails with `ErrNotStreamable` when the response cannot flush |
| `Stream.Send` | `Send(ev Event) error` | Writes one frame and flushes. The first `Send` commits `200 text/event-stream` |
| `Stream.SendJSON` | `SendJSON(name string, v any) error` | Marshals `v` into the data field |
| `Stream.Comment` | `Comment(text string) error` | A comment frame, which dispatches no event |
| `Stream.Close` | `Close(err error) error` | Idempotent. Writes the terminator, or nothing when `err != nil` and uncommitted |
| `Stream.Committed` | `Committed() bool` | Has any byte reached the client. The line between an error response and a truncated answer |
| `Stream.Context` | `Context() context.Context` | Done on client disconnect, on a failed write, or on close |
| `Stream.Err` | `Err() error` | Why the stream ended, or nil |
| `Stream.LastEventID` | `LastEventID() string` | The resume cursor, from the header or the query parameter |
| `Stream.Coalesce` | `Coalesce(interval time.Duration, names ...string) *Coalescer` | Batches the named events |

`Stream` is safe for concurrent use: `Send`, `Comment` and `Close` take the same lock the heartbeat does.

### What the first Send does

`Open` writes nothing on purpose, so an authorization check, a validation failure or a policy veto after it still produces an ordinary status code with an ordinary body, and a handler that emits nothing and returns leaves the response untouched. Everything below happens on the first `Send`, for that response only:

1. Removes every buffering response-writer wrapper (`ctx.DisableResponseBuffering()`), so `middleware/compress` and the `middleware/recorder` that `accesslog` installs stop holding the events.
2. Clears the server's absolute `WriteTimeout`, which is a total deadline rather than an idle one and would otherwise end a long generation mid-answer.
3. Enables full duplex on HTTP/1, so a request body can still be read after the response has started.
4. Sets `Cache-Control: no-cache, no-transform` and `X-Accel-Buffering: no`, and deletes any `Content-Length`.
5. Sets `Connection: keep-alive` on HTTP/1 only. It is a protocol error on HTTP/2.

`Open` does verify up front that the response can flush, because the answer is otherwise only available at flush time, which is too late.

### Event

```go
type Event struct {
    ID      string        // "id": becomes the client's Last-Event-ID.
    Name    string        // "event": empty means "message".
    Data    []byte        // "data": multi-line values split across fields, rejoined with "\n".
    Retry   time.Duration // "retry": the client's reconnection delay.
    Comment string        // written as an SSE comment before the fields.
}
```

A zero field is absent from the wire. A nil `Data` with a `Comment` writes a comment-only frame, which dispatches no event; a non-nil but empty `Data` dispatches an event with an empty payload. `ID`, `Name` and `Comment` are rejected with `ErrInvalidField` when they contain CR, LF or NUL, because those values are newline-terminated on the wire and would otherwise forge frames.

### Options and their defaults

Start from `sse.DefaultOptions()`. A zero field means **off**, not default, so disabling the heartbeat or the terminator stays expressible.

| Field | Default | Why that value |
|-------|---------|----------------|
| `Heartbeat` | `15s` | The largest interval that survives nginx `proxy_read_timeout` (60s), an AWS ALB idle timeout (60s), Cloudflare's 524 (100s) and Heroku's rolling 55s at once. A Google Cloud load balancer applies its backend timeout to the whole request and response, so no heartbeat rescues it: raise `timeoutSec` there |
| `HeartbeatFrame` | `": ping\n\n"` | An SSE comment, which conforming clients ignore. Configurable because `sashabaranov/go-openai` counts comment frames toward `emptyMessagesLimit` and fails the stream |
| `Retry` | `3s` | Written once with the first event. Zero omits the field and leaves the client's own default |
| `AutoID` | `true` | A monotonic id on every frame that does not carry one, so a client that sees id 7 after id 5 knows it missed one. It is **not** resumption: the counter restarts at 1 on every stream, so a `Last-Event-ID` from a previous connection means nothing to it. A resumable stream needs ids of the application's own, tied to its store |
| `Terminator` | `"done"` | A reserved event name, never an in-band data sentinel: a sentinel such as `[DONE]` travels on the same channel as the payload, so a model emitting the literal string would end the client's stream |
| `ErrorFrame` | `ReportErrorFrame` | Renders the error event `Close` writes when a committed stream failed. Nil selects the default |
| `SendTimeout` | `30s` | A write deadline, not a queue: `Send` is synchronous, so there is no buffer to budget. Exceeding it fails with `ErrSlowClient` and never skips an event, because a dropped delta corrupts the assembled message undetectably |

The heartbeat's larger job is liveness rather than keep-alive. Only a real FIN or RST, or a failing write, cancels a request context; a client that disappears without one (a closed lid, a dropped link, an evicted NAT entry) stays invisible until TCP gives up, roughly 924 seconds on Linux, all of it leaked goroutine and leaked upstream work.

### The terminal frame

`Close` writes it on success too, and that is the point: if the terminator is always present, its absence is how a client detects truncation. Chunked framing catches a torn connection, but a proxy or load-balancer idle timeout closes the stream cleanly and the client cannot tell a finished answer from a severed one.

When `Close` is given an error and nothing has been committed, it writes nothing and returns the error, so the caller renders an ordinary HTTP response. After the first `Send` it writes an error event and then the terminator, because the status code is already spent.

Browser clients must close on the terminator explicitly, since `EventSource` would otherwise reconnect:

```js
es.addEventListener("done", () => es.close());
```

### The error frame

`Options.ErrorFrame` renders the event `Close` writes for a stream that failed after committing. The default, `ReportErrorFrame`, runs the error through `rest/errors.Report`, which is what makes a mid-stream failure behave like every other failure in the application:

```go
type ErrorFrameFunc func(ctx *context.Context, err error) (Event, bool)
```

| Renderer | Behaviour |
|----------|-----------|
| `ReportErrorFrame` (default) | The route's error mappings apply, the logger and the collector fire so a cloud sink sees it, a correlation id is generated, and an internal or private error is redacted to `"server error"`. The payload is the `rest/errors` `Error` envelope, so a client parses one shape whether the failure happened before the first byte or after it |
| `RawErrorFrame` | The error's own text, with no mapping, no redaction and no reporting. Safe only when that text did not come from an upstream |

The redaction is the point: an unmapped error becomes a generic message, so an upstream message naming a vendor, a model or a host cannot leak into a stream by default. Map the errors you do want described, through `errors.Options.MapErrors` and its relatives, exactly as for an ordinary response.

Returning `false` writes no frame at all. `ReportErrorFrame` does that when `Report` has nothing to report, which is the case for a client that disconnected while `errors.SkipCanceled` is on: there is nobody left to tell, and nobody should be paged. `ai.Open` installs an AI-aware version of the same renderer, so a classified upstream failure keeps its canonical code (see [ai.md](./ai.md)).

### Coalescing deltas

```go
c := stream.Coalesce(100*time.Millisecond, "token")
defer c.Close()

for token := range tokens {
    c.Send(sse.Event{Name: "token", Data: []byte(token)})
}
```

One frame per token is one write, one flush, one TCP segment and one client-side render per token. A background ticker drains the buffers, which makes the interval a ceiling rather than a floor: without one, a buffered delta is only emitted when the next arrives, so text already on the server sits unsent through exactly the pause a reader notices.

Payloads are concatenated, so this is for text-like deltas and **not** for events whose data is a self-contained JSON document (`SendJSON` therefore always flushes first and sends immediately). Anything not named passes straight through after the pending buffers flush, so ordering holds. The coalescer owns the stream while open, and `Stream.Send` flushes it first, so the two cannot interleave. `Coalescer` has `Send`, `SendJSON`, `Flush` and an idempotent `Close`.

### Shutdown and disconnect

`http.Server.Shutdown` does not cancel a request context, so a handler in a stream loop is never told the process is stopping. Select on both:

```go
for {
    select {
    case <-ctx.ShuttingDown():   // graceful shutdown: end cleanly.
        return
    case <-stream.Context().Done(): // client gone, write failed, or closed.
        return
    case token := <-tokens:
        stream.Send(sse.Event{Name: "token", Data: []byte(token)})
    }
}
```

`ctx.ShuttingDown()` returns a nil channel (which blocks forever) when the request did not come from a server Iris started, which is the correct answer under `httptest`.

### Resuming

`LastEventID` reads the `Last-Event-ID` header **and** the `lastEventId` query parameter. Both are load-bearing: a browser `EventSource` sends the header and cannot be made to send anything else, while a `fetch`-based client (every LLM SDK, because `EventSource` cannot POST a body) usually has to put it in the URL. The value is sanitised, because it is echoed back on every frame. Where the replayed events come from is the application's business; this package stores nothing.

## Reader

`sse.Read(r io.Reader, opts ...ReadOptions) iter.Seq2[Event, error]` is the framework's first `iter.Seq2` API.

```go
for ev, err := range sse.Read(resp.Body, sse.ReadOptions{Terminators: []string{"[DONE]"}}) {
    if err != nil {
        return err
    }
    // ev.Name, ev.Data
}
```

It implements the WHATWG parsing algorithm rather than matching a prefix. `data:` with and without a space are both legal, and a reader that matches `data: ` sees an empty stream from a producer that omits it. Multi-line data joined with `\n`, comments, unknown fields, a leading byte order mark and CR/LF/CRLF line endings are all handled. It never yields a partially framed event, because an event is dispatched by the blank line that ends it, so a stream cut mid-event ends the iteration with `io.ErrUnexpectedEOF` rather than quietly, as if the answer were complete. A clean end of stream simply ends the iteration; `io.EOF` is not reported as an error. The terminal error, if any, comes last.

| `ReadOptions` field | Default | Notes |
|---------------------|---------|-------|
| `Terminators` | empty | Ends the iteration when an event's data equals one exactly. `[]string{"[DONE]"}` for an OpenAI-dialect upstream. A compatibility concession on the reading side only |
| `RequireTerminator` | `false` | Ends the iteration with `ErrNoTerminator` when the stream closes cleanly without one of the `Terminators`. No effect when `Terminators` is empty. An upstream that promised a sentinel and closed without it was cut, however clean the close looked to the transport; `ai.Relay` turns this on whenever terminators are configured |
| `MaxEventBytes` | `DefaultMaxEventBytes` (1 MB) | Caps **one** event; exceeding it ends the iteration with `ErrTooLarge` |
| `KeepComments` | `false` | Yields comment-only frames, such as an upstream's own heartbeat, instead of skipping them |

Lenient on parse, strict on emit: a closed parser rejects a valid-but-newer producer, and an event stream is exactly the kind of format producers extend.

## Errors

Match with `errors.Is`.

| Error | Means |
|-------|-------|
| `ErrNotStreamable` | The response cannot flush. Reported by `Open` with zero bytes written |
| `ErrInvalidField` | CR, LF or NUL in `ID`, `Name` or `Comment` |
| `ErrSlowClient` | One `Send` exceeded `SendTimeout` |
| `ErrClosed` | `Send` on a closed stream |
| `ErrTooLarge` | A read event exceeded `MaxEventBytes` |
| `ErrNoTerminator` | A read stream ended cleanly without one of the `Terminators`, under `RequireTerminator`. It matches `io.ErrUnexpectedEOF` too |

## Traps

**`Timeout.HandlerTimeout` makes streaming impossible.** It wraps the router in `http.TimeoutHandler`, whose response writer implements neither `http.Flusher` nor `http.Hijacker`; every write buffers into a 503 page. This is the usual cause of `ErrNotStreamable`. Setting it logs a warning at startup. Bound slow handlers individually instead.

**`Server.WriteTimeout` is absolute, not idle.** It is a deadline on the whole response, so a long generation dies at the limit however busy the connection was. The first `Send` clears it for that response.

**`LogRequests(true)` used to de-stream every SSE response, and no longer does.** `accesslog` installs a `middleware/recorder` whenever it reads the response body, and the recorder forwarded its buffer only when the handler had set `Transfer-Encoding: chunked`, which no SSE handler does. The recorder now opts out of streaming media types and honours `Disable()`; the regression test is `TestRecorderDoesNotBufferStreams` in `middleware/recorder/disable_test.go`. `middleware/compress` had the same shape.

**`EventSource` cannot POST a body.** Every LLM-style client is therefore built on `fetch`, and `Last-Event-ID` arrives as a query parameter as often as a header. `LastEventID` reads both.

**A retry after the first byte splices two answers.** Check `Committed()` before any failover.

**A handler that returns without `Close` ends the stream anyway.** The framework registers a `Context.OnRelease` hook, so the heartbeat, an attached `Coalescer`'s ticker and its drain are stopped and waited for before the response writer goes back to its pool, and the stream is marked closed: a late `Send` reports `ErrClosed`, and so does `Stream.Err`, which is how an abandoned stream is told apart from a cleanly closed one. Nothing is written, not even the terminator. That missing terminator is the truncation signal. `Stream.Err` is nil after a clean `Close(nil)`.

**`Stream.Commit()` writes the status line and the stream headers without an event**, for a turn that produced nothing and still needs to look like a stream that ended.

## Source Files

- `sse/sse.go` - `Open`, `Stream`, `Send`, `Close`, `Options`, `Event`
- `sse/stream.go` - Frame writing, the heartbeat, the send deadline
- `sse/coalescer.go` - `Coalesce`, the drain ticker
- `sse/read.go` - `Read`, `ReadOptions`, the WHATWG parser
- `sse/json.go` - `SendJSON`
- `sse/errorframe.go` - `ErrorFrameFunc`, `ReportErrorFrame`, `RawErrorFrame`
- `rest/errors/stream.go` - `errors.Report`, `ErrorCodeName.Report` (see [error-handling.md](./error-handling.md))
- `context/response_writer.go` - `FlushError`, `ResponseWriterFlushError`, `ResponseWriterDisabler`
- `context/context_response.go` - `DisableResponseBuffering`, `StreamWriter`
- `context/context_request.go`, `core/server/shutdown.go` - `ShuttingDown`
- `middleware/recorder/response.go`, `middleware/compress/response_writer.go` - The buffering wrappers that step aside
- `_examples/05-response/streaming-sse` - Ticks, batching, and the guarded route
- [book/06-response-handling.md](https://github.com/kataras/iris-book/blob/main/book/06-response-handling.md) - The long-form Server-Sent Events section
