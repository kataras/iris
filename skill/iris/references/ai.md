# AI Backends

Package `github.com/kataras/iris/v14/ai`. Renders and relays AI model streams over HTTP. Transport-aware and provider-agnostic: it understands the shapes streamed generation takes (text deltas, reasoning traces, tool-call fragments, usage trailers, an error that arrives after the response began) and ships no vendor SDK, no HTTP client for any AI service and no vendor request or response schema.

Built on [sse.md](./sse.md), which carries the transport-level defaults, traps and the reader. Read that one too for anything about framing, heartbeats or buffering.

## One handler, both shapes

```go
func chat(ctx iris.Context) {
    req, ok := ctx.BindJSON[ChatRequest]()
    if !ok {
        return
    }

    o := ai.DefaultOptions()
    o.Stream = &req.Stream // honour the request body, as an OpenAI-shaped API must.

    res, err := ai.Open(ctx, o)
    if errors.HandleError(ctx, err) {
        return
    }

    for delta := range generate(req) {
        if err := res.Send(ai.Text("m1", delta)); err != nil {
            res.Close(err)
            return
        }
    }

    res.Close(nil)
}
```

When the request asks to stream, parts go out as produced. When it does not, they are assembled and written as one JSON document by `Close`. **Nothing between `Open` and `Close` changes.** `Options.Stream` is a `*bool` because the zero value has to stay distinguishable from "not configured": nil negotiates on the `Accept` header.

Like `sse.Open`, `ai.Open` writes nothing, so an authorization failure or a policy veto after it still produces an ordinary status code with an ordinary body.

| Symbol | Signature | Notes |
|--------|-----------|-------|
| `ai.Open` | `Open(ctx *context.Context, opts ...Options) (*Response, error)` | Writes nothing |
| `Response.Send` | `Send(p Part) error` | Renders one part |
| `Response.Close` | `Close(err error) error` | Flushes, writes the terminator or the assembled document, runs `OnFinish` exactly once |
| `Response.Committed` | `Committed() bool` | Has any byte reached the client. Consult before any failover |
| `Response.Streaming` | `Streaming() bool` | Which shape this turn took |
| `Response.Context` | `Context() context.Context` | Done on client disconnect or turn end. Nil-safe to select on for a non-streaming turn |
| `Response.Result` | `Result() Result` | A snapshot of what the turn produced so far; complete after `Close` |

## The part vocabulary

Seven kinds, and constructors instead of a struct literal. `Part`'s fields are unexported, so an invalid combination (a tool call carrying reasoning text, a usage part with a delta) cannot be constructed and no encoder has to defend against one.

| Constructor | Kind | Wire/event name | Carries |
|-------------|------|-----------------|---------|
| `ai.Text(id, delta string)` | `KindText` | `text` | A delta of the answer |
| `ai.Reasoning(id, delta string)` | `KindReasoning` | `reasoning` | A delta of a reasoning trace |
| `ai.ToolCall(id, name string, args json.RawMessage)` | `KindToolCall` | `tool` | A tool call. `args` is raw so a not-yet-valid JSON fragment can still be forwarded |
| `ai.ToolResult(id string, result json.RawMessage)` | `KindToolResult` | `tool-result` | What a tool returned |
| `ai.Meter(u Usage)` | `KindUsage` | `usage` | What the turn cost. Named `Meter` so the constructor and the `Usage` type do not collide |
| `ai.Error(err error)` | `KindError` | `error` | A failure that happened after the response began. On a streaming turn it goes through the same policy as the error given to `Close`: reported through `rest/errors`, redacted when unmapped or private, then `Options.Redact`. What reaches the wire is the `rest/errors` envelope object (`{"id","http_error_code","message","detail"}`), never `err.Error()` |
| `ai.Data(name string, raw json.RawMessage)` | `KindData` | the producer's own name | The open namespace |

Accessors: `Kind()`, `ID()`, `Delta()`, `Name()`, `Args()`, `Result()`, `Usage()`, `Raw()`, `Err()`, `IsZero()`, `String()` (logs only), `MarshalJSON()`.

Seven kinds and not twenty: a richer grammar of start/delta/end triples only buys compile-time sequence safety to a framework that owns the generation loop, and this one does not. The `id` gives an interface the same correlation for a fraction of the surface. `KindData` exists because a closed union would reject a valid-but-newer producer and providers add event types continuously. **Strict on emit, lenient on parse.**

## Options

Start from `ai.DefaultOptions()`. A zero field means off, not default.

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `Stream` | `*bool` | nil | nil negotiates on `Accept`. Pass `&req.Stream` for an OpenAI-shaped body |
| `Encoder` | `Encoder` | `SSEJSON()` | A nil encoder also selects `SSEJSON()` |
| `SSE` | `sse.Options` | `sse.DefaultOptions()` | The transport, when the turn streams |
| `Coalesce` | `time.Duration` | `100ms` | Batches consecutive text and reasoning deltas, at most one of each per interval. Zero disables it. Batches **parts**, not bytes, because concatenating two encoded frames would destroy their structure |
| `OnPart` | `func(*context.Context, Part) (Part, error)` | nil | Runs before each part is encoded |
| `OnDisconnect` | `Disconnect` | `DisconnectCancel` | What a vanished client does to the upstream call |
| `OnFinish` | `func(context.Context, *Result)` | nil | Runs exactly once after the turn, under `context.WithoutCancel` |
| `Redact` | `func(string) string` | nil | Rewrites the client-facing text of an error, last, after the pipeline has already made it safe |
| `KeepParts` | `bool` | `false` | Retains every part in `Result.Parts`. Off by default so a long generation is not held twice |

### OnPart: block-only policy

Three outcomes, and no fourth:

```go
(p, nil)             // forward it, possibly rewritten
(_, ai.ErrDropPart)  // drop this part, keep streaming
(_, any other error) // fail the turn
```

It is block-only by construction: you cannot revise what has already been sent, so the restriction lives in the type rather than in a warning. A hook that could rewrite history would pass its tests and fail on the wire.

It also sees the terminal error `Close` writes, as a `KindError` part, so the failure a client is told about goes through the same gate everything else does. Returning `ErrDropPart` there writes no error frame at all, and the stream still gets its terminator.

### OnFinish: the durable copy

Runs on success, on failure and on client disconnect alike, under `context.WithoutCancel`, exactly once. Persistence, billing and metrics belong here. The stream is an optimisation and never the source of truth: a design in which the only copy of the answer is on the wire loses answers.

### Disconnect

| Value | Behaviour |
|-------|-----------|
| `DisconnectCancel` (default) | Ends the turn as soon as the client goes away, because an abandoned generation burns tokens nobody will read |
| `DisconnectDetach` | Keeps consuming the upstream so `OnFinish` still records what was spent. The handler still runs to completion; only the client is gone |

## Result

Handed to `OnFinish` and available from `Response.Result()`.

| Field | Type | Notes |
|-------|------|-------|
| `Streaming`, `Committed`, `Disconnected` | `bool` | `Committed` is the line between an error response and a truncated answer |
| `Abandoned` | `bool` | The handler returned without calling `Close`. The framework ended the turn before the response was recycled: nothing was written afterwards, a pending part was dropped, and the client got no terminator |
| `Text`, `Reasoning` | `string` | The assembled answer and trace |
| `ToolCalls` | `[]Part` | The tool-call parts themselves: they already hold the id, name and raw args |
| `Usage` | `Usage` | Accumulated across every usage part |
| `Parts` | `[]Part` | Only when `KeepParts` is set |
| `Err` | `error` | Why the turn ended badly, or nil |
| `TTFP` | `time.Duration` | Time to the first part: the latency a reader actually perceives |
| `Duration` | `time.Duration` | The whole turn, dominated by how long the answer is |

`Usage` carries `InputTokens`, `OutputTokens`, `CachedInputTokens`, `ReasoningTokens` and an `Extra map[string]int64` for counters a provider reports that the struct does not name. `Add` accumulates, because providers report usage in pieces. **Cost in money is deliberately absent**: pricing a provider means tracking every provider's price list forever. Compute it in `OnFinish`, where you already know which model ran.

`Message` is the non-streaming projection (`Text`, `Reasoning`, `ToolCalls []Part`, `Usage *Usage`); tool calls marshal through `Part`, so the same object shape appears streamed and assembled.

## Encoders

| Constructor | Wire | Use for |
|-------------|------|---------|
| `ai.SSEJSON()` | One SSE frame per part, named for its kind, carrying neutral JSON. A `KindData` part keeps its producer's name | The default. Browsers, anything speaking SSE |
| `ai.NDJSON()` | One neutral JSON object per line, no SSE framing. Ends with a terminal `{"type":"done"}` line, and reports a post-first-byte failure as `{"type":"error","error":{...}}` before it | Server-to-server: no `EventSource` on the other end, so parsing is one `json.Decoder` |
| `ai.PlainText()` | Text deltas only. No terminator and no in-band error: the failure reaches the logger and the collector, not the reader | `curl`, and clients that want prose rather than a protocol. Reasoning, tool calls and usage are dropped |

```go
type Encoder interface {
    ContentType() string          // empty means text/event-stream and selects the SSE transport
    Header(h http.Header)         // dialect headers, before anything is written
    Open(s Sink) error            // once, before the first part
    Encode(s Sink, p Part) error
    Close(s Sink, r *Result) error // the dialect's terminator, on success and failure alike
}

type Sink interface {
    Frame(ev sse.Event) error // or ErrNotFramed for a line-oriented dialect
    io.Writer                 // or ErrNotRaw for an SSE dialect
    Flush() error
}
```

`Encoder` is the one seam through which a vendor shape may enter, and it points **outward**. The AI SDK UI Message Stream, AG-UI and the OpenAI `chat.completion.chunk` shapes are vendor response schemas; a framework shipping one has signed up to track that vendor forever (that format alone changed its framing, its transport, its header name and its semantics inside eighteen months, and its header carries someone else's brand). They belong in `iris-contrib`, against this interface, pinned to a version.

## Reasoning fenced in a tag

Some models emit reasoning inline, fenced by a tag such as `<think>`, in the same channel as the answer. The tag arrives split across deltas as often as not (`"<th"` then `"ink>"`), so a `strings.Contains` per chunk both misses it and leaks its characters into the answer.

```go
splitter := ai.TagSplit("m1", "<think>", "</think>")

for _, p := range splitter.Write(delta) {
    res.Send(p)
}
// at the end of the stream:
for _, p := range splitter.Flush() {
    res.Send(p)
}
```

`Write` holds back at most `len(tag)-1` characters. `Flush` is not optional: a tag that never completed is not a tag, its characters are part of the answer, and dropping them truncates it silently. `Inside()` reports the current state. `StartInside()` covers models that emit a closing tag having never emitted an opening one, where everything before the first `</think>` is reasoning.

## Relaying an upstream

```go
resp, err := httpClient.Do(req)
if ue := ai.Classify(resp, err); ue != nil {
    return res.Close(ue)
}

return res.Close(ai.Relay(res, resp, decoder, ai.RelayOptions{
    Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
}))
```

`Relay(r *Response, resp *http.Response, dec Decoder, opts ...RelayOptions) error` closes `resp.Body`, classifies the response first so a non-2xx upstream comes back as a typed `*UpstreamError` rather than an empty stream, and checks the content type so a gateway answering 200 with a JSON error does not silently become a blank answer. `RelayOptions.AllowNonEventStream` turns that check off and hands the whole body to the decoder as one event; off by default for the reason just given, and it reports `ErrNonEventStream` otherwise.

Every failed read of the upstream body, a reset, a timeout, a stream cut mid-event, or one closed cleanly without the sentinel `RelayOptions.Read.Terminators` promised, comes back through `Classify` as a transient `*UpstreamError` wrapping the cause, so the client gets a `BAD_GATEWAY` error frame (`DEADLINE_EXCEEDED` when the cause was a timeout) instead of a terminator claiming the answer was complete, and a gateway's failover logic sees the upstream failure it is. A client that hangs up mid-read stays `ClassCanceled`. `Relay` sets `ReadOptions.RequireTerminator` whenever terminators are configured, with no opt-out. Decoder errors come back as they are: they are the application's own.

`Decoder` is caller code, `Decode(ev sse.Event) ([]Part, error)`, with `DecoderFunc` adapting a function. Returning zero parts is normal: providers emit keep-alives, role announcements and empty deltas. The framework has no business knowing what a vendor calls a delta this month.

## Typed upstream failures

`Classify(resp *http.Response, err error, opts ...ClassifyOptions) *UpstreamError` returns nil when nothing went wrong. **It takes the response and not only the error, and that is the whole point**: a well-behaved Go HTTP client returns a non-2xx response with a nil error, so a `Classify(err)` signature would have to recover the status code from a string and would recreate exactly the type erasure it deletes. The alternative is what AI gateways actually end up with: a regular expression over `err.Error()`. On failure it reads and closes the body, keeping up to `MaxBody` (`DefaultMaxErrorBody`, 64 KB), because the body is where the provider put the reason. On success it leaves the body for the caller to stream.

| Class | Meaning | `Retryable()` |
|-------|---------|---------------|
| `ClassNone` | The response was fine | false |
| `ClassFatal` | Request-shaped: wrong, and just as wrong on another credential or model | false |
| `ClassTransient` | A 5xx, a reset, a dial error, **or a timeout** | true |
| `ClassRateLimited` | A 429. Its own class because it is a routing signal, not a health signal: a pool that counts it as a fault quarantines a healthy key for being popular | true |
| `ClassExhausted` | This credential or account cannot serve the request at all. A different one might | true |
| `ClassCanceled` | The caller went away, and only that: `context.Canceled` alone. A retry spends tokens on an answer nobody waits for | false |

`Class.String()` returns a stable lowercase name suitable for a metric label.

**401 and 403 map to `ClassExhausted`, not `ClassFatal`.** In a gateway they almost always mean this credential is finished rather than this request is malformed, and treating them as fatal is how a rotated key takes the whole service down instead of failing over.

**A timeout is not a cancellation.** `context.DeadlineExceeded` and any `net.Error` whose `Timeout()` is true are `ClassTransient`, retryable, and reported as `DEADLINE_EXCEEDED` (504); `(*UpstreamError).Timeout()` reports it. Only `context.Canceled` is `ClassCanceled`, with one exception: `Relay` under `DisconnectCancel` reports `ClassCanceled` whenever the turn's own context has ended, whatever ended it, keeping the cause, because nobody is waiting for a retry. net/http's own timeouts, from `http.Client.Timeout` and `Transport.ResponseHeaderTimeout`, satisfy `errors.Is(err, context.DeadlineExceeded)`, so a check on that alone used to read a stalled provider as a client that had gone away, and the turn became an unretried, unreported empty 200.

`UpstreamError` fields: `Status` (0 when the request never got one), `Class`, `Message`, `Code`, `RequestID` (from the body or the usual headers; the only thing that makes a support conversation about one failed generation possible), `RetryAfter`, `Body` (truncated). `AsUpstream(err) (*UpstreamError, bool)` extracts it from a chain; `Unwrap` exposes the transport error so `errors.Is(err, context.Canceled)` keeps working.

`HTTPStatus()` is what to give **your** caller, not the upstream's code. A provider 401 means the gateway's credential is finished, which is a 502 to the caller: telling them they are unauthorised sends them off to check their own key. The one code that passes through is 429, because rate limiting is the caller's problem too and the delay is worth forwarding.

## Errors after the first byte

After the first byte the status code is spent, so a failure is reported in-band. `Close` renders it through `Options.SSE.ErrorFrame`; a turn opened with `ai.Open` installs an AI-aware renderer that keeps a classified upstream failure's canonical code instead of collapsing it into a generic internal error.

| `Class` | `ue.ErrorCode()` | Status |
|---------|------------------|--------|
| `ClassRateLimited` | `errors.ResourceExhausted` | 429 |
| `ClassExhausted` | `errors.Unavailable` | 503 |
| `ClassFatal` | `errors.InvalidArgument`, or `errors.TooLarge` when `Status == 413` | 400, 413 |
| `ClassTransient` with `ue.Timeout()` | `errors.DeadlineExceeded` | 504 |
| `ClassTransient`, anything else | `errors.BadGateway` | 502 |
| `ClassCanceled` | `""` (nothing to report) | none |

`ue.Report(ctx)` runs the `rest/errors` pipeline under that code and returns the client-safe `*errors.Error`, or nil when there is nothing to report. **The client is told what the failure means, never what the provider said.** A rate limit arrives as `RESOURCE_EXHAUSTED` with "the service is rate limited, try again shortly"; the provider's own message, code and request id go to the logger and the collector. The error is wrapped with `errors.Private` on the way in, so the pipeline redacts even if that wording were removed.

Because `Report` writes nothing, the same call serves both sides of the committed line, which is how a gateway hands its caller one envelope shape either way:

```go
if ue, ok := ai.AsUpstream(err); ok {
    reported := ue.Report(ctx)
    if reported == nil {
        return // cancelled: nobody is listening.
    }

    if ue.RetryAfter > 0 {
        ctx.SetHeader("Retry-After", strconv.Itoa(int(ue.RetryAfter.Seconds())))
    }

    ctx.StopWithJSON(reported.ErrorCode.Status, reported)
    return
}

errors.Internal.Err(ctx, err)
```

`Options.Redact` rewrites `Message` and `Detail` last, over text the pipeline already made safe. It exists for an error the application deliberately mapped and described, whose description still names a provider, a model or a host. A `strings.Replacer` is the natural implementation and is safe for concurrent use; hold it in an `atomic.Pointer` if the vocabulary changes at runtime. It applies to error text only: rewriting the model's own output is `OnPart`'s job.

```go
o.Redact = strings.NewReplacer("gpt-4o-mini", "the model", "api.openai.com", "the provider").Replace
```

## Failover

```go
for _, b := range g.chain {
    if res.Committed() {
        return lastErr // the answer has begun: a retry would splice two answers.
    }

    err := g.relayOnce(ctx, res, b)
    if err == nil {
        return nil
    }
    lastErr = err

    ue, ok := ai.AsUpstream(err)
    if !ok {
        return err // our own bug: do not paper over it.
    }

    switch ue.Class {
    case ai.ClassCanceled, ai.ClassFatal:
        return err
    }
    // transient, rate limited or exhausted: try the next backend.
}
```

Then, in the handler, spend the status code only when there is still one to spend. This is the branch a gateway forgets, and the symptom is a 200 with an empty body:

```go
err = res.Close(g.relayWithFailover(ctx, res))
if err == nil || res.Committed() {
    return // finished, either way.
}
// Nothing reached the client: report it as an ordinary response.
// See "Errors after the first byte" above for the body.
```

## Errors

| Error | Means |
|-------|-------|
| `ErrDropPart` | Return it from `OnPart` to discard the part and keep going |
| `ErrNonEventStream` | An upstream answered 200 with a body that is not an event stream |
| `ErrNotFramed` | A `Sink` does not carry SSE frames (a line-oriented dialect such as NDJSON) |
| `ErrNotRaw` | A `Sink` takes frames, not raw bytes (an SSE dialect) |

## Traps

**`http.Client.Timeout` truncates generations.** It covers the whole lifecycle including reading the body, so any non-zero value cuts a long answer off mid-sentence. Leave it zero and let cancellation ride the request context.

**The transport traps live in [sse.md](./sse.md)** and apply to every streaming turn: `Timeout.HandlerTimeout` makes streaming impossible, `Server.WriteTimeout` is absolute rather than idle, a buffering middleware de-streams the response unless the first send unbuffers it, a client that vanishes without a FIN is invisible for about 924 seconds without a heartbeat, and `http.Server.Shutdown` does not cancel a request context.

**Do not retry after `Committed()`.** Two attempts splice two different answers into one response.

**Do not forward a provider's message to your caller.** It names the vendor, the model or the host, and it is text this service did not write. `ue.Report(ctx)` sends it to the logger and the collector and gives the caller a sentence derived from the class; `sse.RawErrorFrame` is the opt-out and is wrong for anything upstream-derived.

**Usage arrives in pieces.** Add it rather than assigning it; `Usage.Add` exists for that, and `Result.Usage` is already accumulated.

## Deliberately not included

An agent loop (turns, stopping conditions and memory vary too much per application, and every framework that shipped one has spent the years since adding kill switches). Any LLM client, model catalog, token counter or cost table. A conversation, memory or vector store. Vendor wire encoders. OpenTelemetry GenAI attributes, because those conventions are still marked development and have already renamed their central attribute once; `Result` carries `TTFP`, `Duration` and `Usage`, so emit the spans you want against the version you chose.

## Source Files

- `ai/ai.go` - `Open`, `Response`, `Options`, `Result`, `Usage`, `Message`
- `ai/turn.go` - Part accumulation, the non-streaming projection, `OnFinish`
- `ai/encoder.go` - `Encoder`, `Sink`, `SSEJSON`, `NDJSON`, `PlainText`
- `ai/relay.go` - `Relay`, `Decoder`, `RelayOptions`
- `ai/error.go` - `Classify`, `Class`, `UpstreamError`, `HTTPStatus`
- `ai/errorframe.go` - `UpstreamError.ErrorCode`, `.Report`, the AI-aware error frame
- `rest/errors/stream.go` - `errors.Report`, `ErrorCodeName.Report` (see [error-handling.md](./error-handling.md))
- `ai/tagsplit.go` - `TagSplit`, `Write`, `Flush`, `StartInside`
- `sse/` - The transport (see [sse.md](./sse.md))
- `_examples/05-response/streaming-ai` - One handler, three formats, inline reasoning
- `_examples/16-advanced/ai-gateway` - Relay, classification, failover, accounting
- [book/21-ai-backends.md](https://github.com/kataras/iris-book/blob/main/book/21-ai-backends.md) - The long-form chapter this condenses
