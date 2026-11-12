# Server-Sent Events

Server-Sent Events for Iris, in both directions: a push-based response stream, and a tolerant reader for consuming somebody else's.

This package is transport only. It ships no application vocabulary, no replay store and no pub/sub fan-out, because those vary per service and keeping them out is what lets the transport stay small enough to be correct. The [ai](../ai) package builds the AI-shaped layer on top.

```sh
go get github.com/kataras/iris/v14@latest
```

```go
import (
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)
```

## Writing a stream

```go
func chat(ctx iris.Context) {
	stream, err := sse.Open(ctx)
	if err != nil {
		errors.HandleError(ctx, err) // renders the 500 envelope, logs the cause.
		return
	}
	defer stream.Close(nil)

	for token := range tokens {
		if err := stream.Send(sse.Event{Name: "token", Data: []byte(token)}); err != nil {
			return // the client is gone.
		}
	}
}
```

`Open` writes nothing. The protocol headers commit on the first `Send`, so an authorization check, a validation failure or a policy veto after `Open` still produces an ordinary status code with an ordinary body, and a handler that emits nothing and returns leaves the response untouched.

On that first `Send`, and for that response only, it does five things no middleware can guess:

* removes every buffering response-writer wrapper, so compression and the response recorder that `accesslog` installs stop holding the events
* clears the server's absolute `WriteTimeout`, which is a total deadline and would otherwise end a long generation mid-sentence
* enables full duplex on HTTP/1, so a POST body can still be read after the response has started
* sets `Cache-Control: no-cache, no-transform` and `X-Accel-Buffering: no`, and deletes any `Content-Length`
* sets `Connection: keep-alive` only on HTTP/1, where it means something

`Open` does check up front that the response can flush, and reports `ErrNotStreamable` with zero bytes written when it cannot. The usual cause is `Timeout.HandlerTimeout`, which wraps the router in `http.TimeoutHandler`, whose writer implements neither `http.Flusher` nor `http.Hijacker`.

## Reading a stream

```go
for ev, err := range sse.Read(resp.Body, sse.ReadOptions{Terminators: []string{"[DONE]"}}) {
	if err != nil {
		return err
	}
	// ev.Name, ev.Data
}
```

`Read` implements the parsing algorithm rather than matching a prefix, which matters more than it sounds: `data:` with and without a space are both legal, and a reader that matches `data: ` sees an empty stream from a producer that omits it. It handles multi-line data, comments, unknown fields, a leading byte order mark, and lines terminated by CR, LF or CRLF.

It never yields a partially framed event. A stream cut before the blank line that dispatches an event ends with `io.ErrUnexpectedEOF` rather than quietly, because three deltas and silence look exactly like a complete answer. With `RequireTerminator` set, a stream that ends cleanly without one of the configured `Terminators` ends with `ErrNoTerminator`, which also matches `io.ErrUnexpectedEOF`; `ai.Relay` sets it whenever terminators are configured. `MaxEventBytes` caps one event, not the stream.

## Batching deltas

One frame per token means one write, one flush, one TCP segment and one client-side render per token, and the tokens arrive faster than any of those are worth doing.

```go
c := stream.Coalesce(100*time.Millisecond, "token")
defer c.Close()

for token := range tokens {
	c.Send(sse.Event{Name: "token", Data: []byte(token)})
}
```

A background ticker drains the buffer, which is what makes the interval a ceiling rather than a floor. Without one, a buffered delta is only emitted when the next one arrives, so text that already reached the server sits unsent through exactly the pause a reader notices. The coalescer owns the stream while it is open, and `Stream.Send` flushes it first, so the two cannot interleave. `Stream.Close` flushes it too, so text still buffered when the stream ends goes out before the terminator whichever of the two is closed first.

## The defaults, and why

| Option | Default | Why |
| --- | --- | --- |
| `Heartbeat` | `15s` | The largest interval that survives nginx `proxy_read_timeout` (60s), an AWS ALB idle timeout (60s), Cloudflare's 524 (100s) and Heroku's rolling 55s at the same time. A Google Cloud load balancer applies its backend timeout to the whole request and response, so raise `timeoutSec` there instead. |
| `HeartbeatFrame` | `": ping\n\n"` | A comment, which conforming clients ignore. Configurable because `sashabaranov/go-openai` counts comment frames toward `emptyMessagesLimit` and fails the stream instead. |
| `Retry` | `3s` | The reconnection time advertised to the client, written once with the first event. Zero omits the field and leaves the client's own default in place. |
| `AutoID` | `true` | An `id` on every frame makes gaps detectable within one connection. It is not resumption: the counter starts at 1 on every stream, so a resumable stream needs ids of the application's own. |
| `Terminator` | `"done"` | A reserved event name, never an in-band data sentinel. A sentinel such as `[DONE]` travels on the same channel as the payload, so a model that emits the literal string ends the client's stream. Validated at `Open`: a newline in it is a forged frame. |
| `ErrorFrame` | `ReportErrorFrame` | Renders the event `Close` writes for a failure after the first byte. The default runs the error through `rest/errors`; `RawErrorFrame` sends the error's own text. Its output gets the same framing check as any `Send`. |
| `SendTimeout` | `30s` | A write deadline, not a queue: `Send` is synchronous, so there is nothing of ours to budget. Exceeding it fails with `ErrSlowClient` and never skips an event, because a dropped delta corrupts the message the client is assembling with no way for it to notice. |

The heartbeat's larger job is liveness rather than keep-alive. Only a real FIN or RST, or a failing write, cancels a request context. A client that disappears without one, from a closed lid or a dropped link, stays invisible until TCP gives up, which on Linux is about 924 seconds of leaked goroutine and leaked upstream work.

## The terminal frame

`Close` writes it on success too, and that is the point. If the terminator is always present, its absence is how a client detects truncation. Nothing below it provides that signal: chunked framing catches a torn connection, but a proxy idle timeout closes the stream cleanly and the client cannot tell a finished answer from a severed one.

When `Close` is given an error and nothing has been committed, it writes nothing and returns the error, so the caller renders an ordinary HTTP response. After the first `Send` it writes an error event and then the terminator, because the status code is already spent. `Err` reports that error afterwards, and nil after a clean close.

A handler that returns without calling `Close` does not get a terminator: the stream is ended by the framework before the response is recycled, its heartbeat stopped, and the missing terminator is exactly what tells the client the answer was cut. `Commit` exists for the opposite case, a stream that turns out to carry no events and still needs its terminator.

## Errors after the first byte

Once an event has gone out the status code is spent, so a failure can only be
reported inside the stream. `Close(err)` writes an error event, and by default
that event is produced by `ReportErrorFrame`, which runs the error through
`rest/errors`:

* the route's error mappings apply, exactly as for an ordinary response
* the logger and the collector fire, so a cloud sink sees the failure. This is
  the half that is easy to lose: a stream that fails silently is a stream nobody
  knows failed
* a correlation id is generated
* an unmapped or private error is redacted to a generic message
* nothing is written or reported when the client itself has gone. That is decided by the request's own context, not by the error's shape: an upstream timeout looks like `context.DeadlineExceeded` to `errors.Is` and is still reported, because the client is waiting to hear about it

That last point is the one to notice. The payload is the `rest/errors` envelope,
so a client parses the same shape whether the failure happened before the first
byte or after it, and an upstream message naming a vendor, a model or a host
cannot leak into a stream by default. Map the errors you do want to describe, and
they keep their own words:

```go
app.UseRouter(errors.NewOptions().
    MapErrors(errors.ResourceExhausted, ErrModelBusy).
    Handler())
```

Set `Options.ErrorFrame = sse.RawErrorFrame` to send the error's own text
instead. That is appropriate only when the text is already known to be safe for
a client to read, which text derived from an upstream response is not.

## Resuming

`LastEventID` reads the `Last-Event-ID` header and the `lastEventId` query parameter. Both are load-bearing: a browser `EventSource` sends the header and cannot be made to send anything else, while a client built on fetch, which is every LLM SDK because `EventSource` cannot POST a body, usually has to put it in the URL. The value is sanitised, because it is echoed back on every frame.

Where the events come from on a resume is your application's business. This package does not store them.

## Draining on shutdown

`http.Server.Shutdown` does not cancel a request context, so a handler in a stream loop is never told the process is stopping. Select on `Context.ShuttingDown` as well as the stream's own context:

```go
for {
	select {
	case <-ctx.ShuttingDown():
		return // Close writes the terminal frame.
	case <-stream.Context().Done():
		return
	case token := <-tokens:
		stream.Send(sse.Event{Name: "token", Data: []byte(token)})
	}
}
```

## Example

[_examples/05-response/streaming-sse](../_examples/05-response/streaming-sse)
