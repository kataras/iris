// Package ai renders and relays AI model streams over HTTP.
//
// It is transport-aware and provider-agnostic. It understands the shapes that
// streamed generation takes - text deltas, reasoning traces, tool-call
// fragments, usage trailers, an error that arrives after the response has
// already begun - and it ships no vendor SDK, no HTTP client for any AI service,
// and no vendor request or response schema. What travels over the wire is your
// business; the shape of the wire is this package's.
//
// The core is a small neutral vocabulary, Part, and a pluggable Encoder that
// renders it in a wire dialect. One handler serves both a streaming and a
// non-streaming request, because that is what every OpenAI-shaped API has to do
// and nothing in Go offers it.
//
// It deliberately does not include an agent loop, a conversation store, a
// prompt template format, a token counter or a cost table. Those vary too much
// between applications to belong in a framework, and every framework that
// shipped one has spent the years since adding escape hatches to it.
package ai

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/rest/errors"
)

// Kind identifies what a Part carries.
type Kind uint8

// The part vocabulary. Seven kinds, not twenty: a richer grammar of
// start/delta/end triples only buys compile-time sequence safety to a framework
// that owns the generation loop, and this one deliberately does not. The id
// field gives a interface the same correlation for a fraction of the surface.
const (
	// KindText is a delta of the answer itself.
	KindText Kind = iota + 1
	// KindReasoning is a delta of a reasoning trace, which many models emit
	// separately from the answer and most interfaces render differently.
	KindReasoning
	// KindToolCall is a request from the model to call a tool.
	KindToolCall
	// KindToolResult is what a tool returned.
	KindToolResult
	// KindUsage is what the turn cost.
	KindUsage
	// KindError is a failure that happened after the response began.
	KindError
	// KindData is the open namespace: a producer emitting something this
	// vocabulary does not model passes it through named rather than losing it.
	//
	// It exists because a closed Go union would reject a valid-but-newer
	// producer, and providers add event types continuously. Strict on emit,
	// lenient on parse.
	KindData
)

// String returns the wire name of the kind, which is also the SSE event name
// the default encoder uses.
func (k Kind) String() string {
	switch k {
	case KindText:
		return "text"
	case KindReasoning:
		return "reasoning"
	case KindToolCall:
		return "tool"
	case KindToolResult:
		return "tool-result"
	case KindUsage:
		return "usage"
	case KindError:
		return "error"
	case KindData:
		return "data"
	default:
		return "unknown"
	}
}

// Part is one increment of a model's answer.
//
// Its fields are unexported and it is built through the constructors below, so
// an invalid combination - a tool call carrying reasoning text, a usage part
// with a delta - cannot be constructed, and no encoder has to defend against
// one. That is as close as Go gets to a sum type, and it turns the grammar into
// an invariant rather than a doc comment.
type Part struct {
	kind  Kind
	id    string
	name  string
	delta string
	raw   jsontext.Value
	usage Usage
	err   error
}

// Text returns a delta of the answer. The id correlates deltas that belong to
// the same message, and may be empty when there is only one.
func Text(id, delta string) Part {
	return Part{kind: KindText, id: id, delta: delta}
}

// Reasoning returns a delta of a reasoning trace.
func Reasoning(id, delta string) Part {
	return Part{kind: KindReasoning, id: id, delta: delta}
}

// ToolCall returns a request to call a tool. args is raw JSON so that a
// fragment which is not yet valid JSON can still be forwarded.
func ToolCall(id, name string, args jsontext.Value) Part {
	return Part{kind: KindToolCall, id: id, name: name, raw: args}
}

// ToolResult returns what a tool produced.
func ToolResult(id string, result jsontext.Value) Part {
	return Part{kind: KindToolResult, id: id, raw: result}
}

// Meter returns the usage of a turn. It is named Meter rather than Usage so
// that the constructor and the type do not collide.
func Meter(u Usage) Part {
	return Part{kind: KindUsage, usage: u}
}

// Error returns a failure that happened after the response began.
//
// On a streaming turn it is sent through the same policy as the error given to
// Response.Close: the error is reported through rest/errors, an unmapped or
// private one is redacted, Options.Redact runs last, and what reaches the wire
// is the resulting envelope. The text of err itself never does.
func Error(err error) Part {
	return Part{kind: KindError, err: err}
}

// Data returns an event this vocabulary does not model, passed through under its
// own name.
func Data(name string, raw jsontext.Value) Part {
	return Part{kind: KindData, name: name, raw: raw}
}

// Kind reports what this part carries.
func (p Part) Kind() Kind { return p.kind }

// ID is the correlation id, for the kinds that have one.
func (p Part) ID() string { return p.id }

// Name is the tool name for KindToolCall and the event name for KindData.
func (p Part) Name() string { return p.name }

// Delta is the text increment for KindText and KindReasoning.
func (p Part) Delta() string { return p.delta }

// Args is the raw tool arguments for KindToolCall.
func (p Part) Args() jsontext.Value { return p.raw }

// ArgsAs decodes the tool arguments of a KindToolCall part into T,
// under the framework's unmarshal defaults (jsonutil).
func (p Part) ArgsAs[T any]() (T, error) {
	var v T
	err := json.Unmarshal(p.raw, &v, jsonutil.DefaultUnmarshalOptionsSlice...)
	return v, err
}

// Result is the raw tool result for KindToolResult.
func (p Part) Result() jsontext.Value { return p.raw }

// ResultAs decodes the tool result of a KindToolResult part into T,
// under the framework's unmarshal defaults (jsonutil).
func (p Part) ResultAs[T any]() (T, error) {
	var v T
	err := json.Unmarshal(p.raw, &v, jsonutil.DefaultUnmarshalOptionsSlice...)
	return v, err
}

// Usage is what the turn cost, for KindUsage.
func (p Part) Usage() Usage { return p.usage }

// Err is the failure, for KindError.
func (p Part) Err() error { return p.err }

// Raw is the passthrough payload, for KindData.
func (p Part) Raw() jsontext.Value { return p.raw }

// IsZero reports whether this part was never constructed.
func (p Part) IsZero() bool { return p.kind == 0 }

// String is for logs and test failures, not for the wire.
func (p Part) String() string {
	switch p.kind {
	case KindText, KindReasoning:
		return fmt.Sprintf("%s(%q)", p.kind, p.delta)
	case KindToolCall:
		return fmt.Sprintf("tool(%s %s)", p.name, p.raw)
	case KindError:
		return fmt.Sprintf("error(%v)", p.err)
	default:
		return p.kind.String()
	}
}

// wire is the neutral JSON shape of a part.
//
// The type discriminator is always present, even for the SSE encoder where the
// event name carries it too. It costs a few bytes and it means a consumer that
// receives one of these objects out of context - from a log, from an NDJSON
// line, from a test fixture - can still tell what it is.
type wire struct {
	Type   string         `json:"type"`
	ID     string         `json:"id,omitempty"`
	Name   string         `json:"name,omitempty"`
	Delta  string         `json:"delta,omitempty"`
	Args   jsontext.Value `json:"args,omitempty"`
	Result jsontext.Value `json:"result,omitempty"`
	Usage  *Usage         `json:"usage,omitempty"`
	// Error is the rest/errors envelope: correlation id, canonical code, message
	// and detail, the same object an ordinary failed response carries. A part
	// that never went through the policy (one marshalled for a log, say) shows
	// its error's text as the message.
	Error *errors.Error  `json:"error,omitempty"`
	Data  jsontext.Value `json:"data,omitempty"`
}

func (p Part) wireView() wire {
	w := wire{Type: p.kind.String(), ID: p.id}

	switch p.kind {
	case KindText, KindReasoning:
		w.Delta = p.delta
	case KindToolCall:
		w.Name = p.name
		w.Args = p.raw
	case KindToolResult:
		w.Result = p.raw
	case KindUsage:
		u := p.usage
		w.Usage = &u
	case KindError:
		if reported, ok := p.err.(*errors.Error); ok {
			w.Error = reported
		} else if p.err != nil {
			w.Error = &errors.Error{Message: p.err.Error()}
		}
	case KindData:
		w.Name = p.name
		w.Data = p.raw
	}

	return w
}

// MarshalJSONTo renders the part in the neutral wire shape, streaming
// straight into the encoder — the fast path every encoder in this package
// and ctx.JSON take, with no intermediate []byte.
func (p Part) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, p.wireView())
}

// MarshalJSON renders the part in the neutral wire shape. It exists for
// encoding/json (v1) interop — a downstream logger calling the old
// json.Marshal on a Part still gets the same bytes.
func (p Part) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.wireView())
}

// Usage is what one turn cost.
//
// Cost in money is deliberately absent: pricing a provider means tracking every
// provider's price list forever, and it changes more often than this package
// would ship. Compute it in OnFinish, where you already know which model ran.
type Usage struct {
	// omitzero, not omitempty: under encoding/json/v2 omitempty no longer
	// omits a zero number.
	InputTokens       int64 `json:"input_tokens,omitzero"`
	OutputTokens      int64 `json:"output_tokens,omitzero"`
	CachedInputTokens int64 `json:"cached_input_tokens,omitzero"`
	ReasoningTokens   int64 `json:"reasoning_tokens,omitzero"`

	// Extra carries counters a provider reports that this struct does not name,
	// so nothing is lost in translation.
	Extra map[string]int64 `json:"extra,omitempty"`
}

// Add accumulates u2 into u. Providers report usage in pieces - some at the
// start of a stream, some at the end - and a relay has to add them up.
func (u *Usage) Add(u2 Usage) {
	u.InputTokens += u2.InputTokens
	u.OutputTokens += u2.OutputTokens
	u.CachedInputTokens += u2.CachedInputTokens
	u.ReasoningTokens += u2.ReasoningTokens

	for k, v := range u2.Extra {
		if u.Extra == nil {
			u.Extra = make(map[string]int64, len(u2.Extra))
		}
		u.Extra[k] += v
	}
}

// IsZero reports whether anything was counted.
func (u Usage) IsZero() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CachedInputTokens == 0 && u.ReasoningTokens == 0 && len(u.Extra) == 0
}
