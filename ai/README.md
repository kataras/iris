# AI backends

Renders and relays AI model streams over HTTP.

The package is transport-aware and provider-agnostic. It understands the shapes that streamed generation takes, including text deltas, reasoning traces, tool-call fragments, usage trailers and an error that arrives after the response has already begun. It ships no vendor SDK, no HTTP client for any AI service, and no vendor request or response schema. What travels over the wire is your business; the shape of the wire is this package's.

```sh
go get github.com/kataras/iris/v14@latest
```

```go
import (
	"github.com/kataras/iris/v14/ai"
	"github.com/kataras/iris/v14/rest/errors"
)
```

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
	if err != nil {
		errors.HandleError(ctx, err) // renders the 500 envelope, logs the cause.
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

When the request asks to stream, the parts go out as they are produced. When it does not, they are assembled and written as one JSON document. Nothing between `Open` and `Close` changes. `Options.Stream` is a `*bool` rather than a `bool` because the zero value has to stay distinguishable from "not configured".

Like `sse.Open`, `ai.Open` writes nothing, so an authorization failure or a policy veto after it still produces an ordinary status code with an ordinary body.

## The part vocabulary

Seven kinds, and constructors instead of a struct literal:

```go
ai.Text(id, delta)
ai.Reasoning(id, delta)
ai.ToolCall(id, name, args)
ai.ToolResult(id, result)
ai.Meter(usage)
ai.Error(err)
ai.Data(name, raw) // the open namespace
```

`Part` has unexported fields, so an invalid combination such as a tool call carrying reasoning text cannot be constructed and no encoder has to defend against one. That turns the grammar into an invariant rather than a doc comment.

`KindData` exists because a closed union would reject a valid-but-newer producer, and providers add event types continuously. Strict on emit, lenient on parse.

## Encoders

`SSEJSON` is the default: one event per part, named for its kind, carrying neutral JSON. `NDJSON` writes one object per line for a server-to-server consumer, and ends with a terminal line, `{"type":"done"}`, so the consumer can detect truncation the way an SSE client does. `PlainText` writes the answer and nothing else, for curl: no terminator, no in-band error, because any marker would read as prose.

A streaming turn with no parts at all still commits and writes its terminator, as the non-streaming shape writes `{}`: an empty answer must be distinguishable from a connection that dropped before the first byte.

The `Encoder` interface is the single seam through which a vendor dialect may enter, and it points outward. The AI SDK UI Message Stream, AG-UI and the OpenAI `chat.completion.chunk` shapes are vendor response schemas; a framework that ships one has signed up to track that vendor forever. That format alone changed its framing, its transport, its header name and its semantics inside eighteen months, and its header carries someone else's brand. Those belong in `iris-contrib`, against this interface, pinned to a version.

## Relaying an upstream

```go
resp, err := httpClient.Do(req)
if ue := ai.Classify(resp, err); ue != nil {
	res.Close(ue)
	return
}

res.Close(ai.Relay(res, resp, decoder, ai.RelayOptions{
	Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
}))
```

`Relay` classifies the response first, so a non-2xx upstream comes back as a typed error rather than as an empty stream, and it checks the content type, so a gateway that answers 200 with a JSON error does not silently become a blank answer. An upstream cut mid-event, or closed without the `[DONE]` it was configured to send, is an error too (matching `io.ErrUnexpectedEOF`), so the client gets an error frame rather than a terminator that says the answer was complete.

The `Decoder` is your code. The framework has no business knowing what a particular vendor calls a delta this month.

## Typed upstream failures

`Classify` takes the response and not only the error, and that is the whole point. A well-behaved Go HTTP client returns a non-2xx response with a nil error, so a `Classify(err)` form would have to recover the status code from a string and would recreate exactly the type erasure it exists to delete. The alternative is what AI gateways actually end up with: a regular expression over `err.Error()`.

| Class | Meaning |
| --- | --- |
| `ClassFatal` | The request is wrong and would be just as wrong on another credential. |
| `ClassTransient` | A 5xx, a reset, a dial error, a timeout. Retrying is reasonable. `Timeout()` tells a stalled provider apart from the rest; it is reported as `DEADLINE_EXCEEDED`. |
| `ClassRateLimited` | A 429. Its own class because it is a routing signal, not a health signal: a credential pool that counts it as a fault quarantines a healthy key for being popular. |
| `ClassExhausted` | This credential or account cannot serve the request at all. A different one might. |
| `ClassCanceled` | The caller went away (`context.Canceled`, and only that). A retry spends tokens on an answer nobody is waiting for. |

401 and 403 map to exhaustion rather than fatal, because in a gateway they almost always mean this credential is finished rather than this request is malformed, and treating them as fatal is how a rotated key takes the whole service down instead of failing over.

## What the client is told

Every failure, whether it arrives before the first byte or after it, and whether it ends the turn through `Close(err)` or is sent as a part with `ai.Error(err)`, goes through `rest/errors` and reaches the client as the same envelope:

```json
{"id":"5d09cc97-...","http_error_code":{"canonical_name":"RESOURCE_EXHAUSTED","status":429},
 "message":"too many requests","detail":"the service is rate limited, try again shortly"}
```

Two things happen there that you do not have to write.

**The failure is reported.** The logger and the collector fire with the real error, which is what carries it to a cloud sink. `errors.LogError` is the hook, so an existing setup keeps working unchanged. Without this a mid-stream failure is invisible: the render path cannot run once a status code is committed, and skipping the pipeline skips the reporting with it.

**The provider's words never reach the client.** A provider message is text this service did not write, about a system the caller cannot see, and it routinely names the vendor, the model or the host. It goes to the logger, where it is useful. The caller gets a sentence derived from the class, and a classified failure keeps its canonical code, so a rate limit is `RESOURCE_EXHAUSTED` rather than a generic internal error.

On SSE the envelope is the data of the `error` event. On NDJSON it is nested, `{"type":"error","error":{...}}`, because every line carries a type. On PlainText nothing is written: the logger and the collector still hear of the failure, the reader does not.

Only the client's own disconnect silences the report. An upstream timeout looks like `context.DeadlineExceeded` to `errors.Is`, and it is still classified, reported and written, because the client is waiting to hear about it.

For the remaining case, an error your application mapped and described on purpose whose description still names a vendor, `Options.Redact` rewrites the client-facing text:

```go
o.Redact = strings.NewReplacer("anthropic", "the model provider").Replace
```

A `strings.Replacer` is safe for concurrent use; hold it in an `atomic.Pointer` if the vocabulary changes at runtime. `Redact` applies to error text only. Rewriting the model's own output is `OnPart`'s job, where the decision can be made per part.

`OnPart` also sees the terminal error, as a `KindError` part, so one hook governs everything the client is told. Returning `ErrDropPart` there writes no error frame at all, and the stream still gets its terminator.

## Reasoning fenced in a tag

Several models emit their reasoning inline, fenced by a tag such as `<think>`. The tag arrives split across deltas as often as not, so a `strings.Contains` on each chunk both misses it and leaks its characters into the answer.

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

`StartInside()` covers models that emit a closing tag having never emitted an opening one.

## Policy and accounting

`Options.OnPart` runs before each part is encoded, with three outcomes: forward it, possibly rewritten; return `ErrDropPart` to drop it and keep going; return anything else to fail the turn. It is block-only by construction, because you cannot revise what you have already sent, and that restriction lives in the type rather than in a warning.

`Options.OnFinish` runs exactly once after the turn ends, under `context.WithoutCancel`, with the assembled answer and the usage. It runs on success, on failure and on client disconnect alike, and for a turn the handler abandoned without `Close`, which the framework ends before the response is recycled and marks with `Result.Abandoned`. Persistence, billing and metrics belong there. The stream is an optimisation and never the source of truth: a design in which the only copy of the answer is on the wire loses answers.

## What this package deliberately does not do

* **An agent loop.** Turns, stopping conditions and memory vary too much per application. Every framework that shipped one has spent the years since adding kill switches to it. The primitives here make a manual loop a few lines.
* **Any LLM client, model catalog, token counter or cost table.** `client.Client` is all the HTTP a relay needs, and pricing a provider means tracking every provider's price list forever.
* **A conversation, memory or vector store.** `OnFinish` hands you the assembled turn and the usage; the schema is yours.
* **Vendor wire encoders.** See above.
* **OpenTelemetry attributes.** The GenAI conventions left the main semantic-conventions repository and everything in the new home is still marked development, having already renamed its central attribute once. `Result` carries `TTFP`, `Duration` and `Usage`; emit the spans you want, pinned to the version you chose.

## Examples

[_examples/05-response/streaming-ai](../_examples/05-response/streaming-ai) and [_examples/16-advanced/ai-gateway](../_examples/16-advanced/ai-gateway)
