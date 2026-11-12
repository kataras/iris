package ai

import (
	json "encoding/json/v2"
	"errors"
	"io"
	"net/http"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/sse"
)

// ErrNotFramed reports that a Sink does not carry SSE frames, which is the case
// for a line-oriented dialect such as NDJSON.
var ErrNotFramed = errors.New("iris: ai: sink is not an event stream")

// ErrNotRaw reports that a Sink does not take raw bytes, which is the case for
// an SSE dialect, where everything must be a frame.
var ErrNotRaw = errors.New("iris: ai: sink takes frames, not raw bytes")

// Sink is what an Encoder writes into.
//
// An encoder uses whichever half matches the transport it asked for through
// ContentType: an empty content type gets an event stream and uses Frame, and
// anything else gets a plain streaming response and uses Write.
type Sink interface {
	// Frame writes one SSE frame, or returns ErrNotFramed.
	Frame(ev sse.Event) error
	// Write writes raw bytes, or returns ErrNotRaw.
	io.Writer
	// Flush pushes whatever has been written to the client.
	Flush() error
}

// Encoder renders neutral Parts in a wire dialect.
//
// It is the one seam through which a vendor shape may enter, and it points
// outward on purpose: this package ships neutral encoders only. The AI SDK UI
// Message Stream, AG-UI and the OpenAI chat.completion.chunk dialects are vendor
// response schemas, and a framework that ships one has signed up to track that
// vendor's changes forever - that format alone changed its framing, its
// transport, its header name and its semantics inside eighteen months, and its
// header carries someone else's brand. They belong in iris-contrib, against this
// interface, pinned to a version.
type Encoder interface {
	// ContentType is the response media type. Empty means text/event-stream and
	// selects the SSE transport.
	ContentType() string
	// Header applies any dialect headers before anything is written.
	Header(h http.Header)
	// Open is called once, before the first part.
	Open(s Sink) error
	// Encode renders one part.
	Encode(s Sink, p Part) error
	// Close writes the dialect's terminator, on success and on failure alike.
	Close(s Sink, r *Result) error
}

// SSEJSON is the default encoder: one SSE frame per part, named for its kind -
// text, reasoning, tool, tool-result, usage, error - carrying the neutral JSON
// of that part. A KindData part keeps the name its producer gave it.
//
// An error part is the exception to "the neutral JSON of that part": its frame
// carries the rest/errors envelope itself, which is exactly the frame
// Response.Close writes for a terminal failure, so a client handles one shape
// for an error whether it was sent mid-stream or ended the stream.
//
// Vendor-neutral and stable, which is the point: a dialect that changes is a
// dialect that belongs behind this interface rather than inside it.
func SSEJSON() Encoder { return sseJSONEncoder{} }

type sseJSONEncoder struct{}

func (sseJSONEncoder) ContentType() string { return "" }
func (sseJSONEncoder) Header(http.Header)  {}
func (sseJSONEncoder) Open(Sink) error     { return nil }
func (sseJSONEncoder) Close(Sink, *Result) error {
	// The terminator is the stream's own, so that truncation detection works
	// the same way for every dialect that rides on SSE.
	return nil
}

func (sseJSONEncoder) Encode(s Sink, p Part) error {
	// Releasing the pooled buffer here is safe only because Sink.Frame copies
	// the Event.Data synchronously (sse.Stream.Send appends it into the
	// stream's own frame buffer before returning); a Sink implementation that
	// retains Data past the Frame call would break this.
	buf := jsonutil.AcquireBuffer()
	defer jsonutil.ReleaseBuffer(buf)

	var payload any = p
	if p.Kind() == KindError {
		// The envelope itself, as Close writes it. See SSEJSON.
		payload = p.wireView().Error
	}

	if err := json.MarshalWrite(buf, payload, jsonutil.DefaultMarshalOptionsSlice...); err != nil {
		return err
	}

	name := p.Kind().String()
	if p.Kind() == KindData && p.Name() != "" {
		name = p.Name()
	}

	return s.Frame(sse.Event{Name: name, Data: buf.Bytes()})
}

// NDJSON writes one neutral JSON object per line with no SSE framing.
//
// It is the right choice for a server-to-server consumer: there is no
// EventSource on the other end, so the framing buys nothing and the parsing is
// one json.Decoder.
//
// The stream ends with a terminal line, {"type":"done"}, on success and on
// failure alike, for the same reason the SSE transport has a terminator: its
// absence is how the consumer detects truncation. A failure after the first
// byte is reported before it as {"type":"error","error":{...}}, where the
// object is the rest/errors envelope.
func NDJSON() Encoder { return ndjsonEncoder{} }

type ndjsonEncoder struct{}

func (ndjsonEncoder) ContentType() string { return "application/x-ndjson" }
func (ndjsonEncoder) Open(Sink) error     { return nil }

func (ndjsonEncoder) Close(s Sink, _ *Result) error {
	if _, err := s.Write(ndjsonDone); err != nil {
		return err
	}

	return s.Flush()
}

func (ndjsonEncoder) Header(h http.Header) {
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")
}

func (ndjsonEncoder) Encode(s Sink, p Part) error {
	// Streaming straight into the sink: a partial line on a marshal error is
	// acceptable here, the stream is already dead when that happens.
	if err := json.MarshalWrite(s, p, jsonutil.DefaultMarshalOptionsSlice...); err != nil {
		return err
	}

	if _, err := s.Write(ndjsonNewline); err != nil {
		return err
	}

	return s.Flush()
}

var (
	ndjsonNewline = []byte{'\n'}
	ndjsonDone    = []byte(`{"type":"done"}` + "\n")
)

// PlainText writes the answer's text deltas and nothing else.
//
// It exists for curl and for a client that wants prose rather than a protocol.
// Reasoning, tool calls and usage are dropped, because there is nowhere in this
// dialect to put them, and so are errors: there is no terminator and no
// in-band error frame, because any marker would read as part of the prose. A
// failure after the first byte still reaches the logger and the collector, but
// on the wire a truncated answer is indistinguishable from a complete one. Use
// it where that is acceptable, which is a human reading a terminal.
func PlainText() Encoder { return plainTextEncoder{} }

type plainTextEncoder struct{}

func (plainTextEncoder) ContentType() string       { return "text/plain; charset=utf-8" }
func (plainTextEncoder) Open(Sink) error           { return nil }
func (plainTextEncoder) Close(Sink, *Result) error { return nil }

func (plainTextEncoder) Header(h http.Header) {
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")
}

func (plainTextEncoder) Encode(s Sink, p Part) error {
	if p.Kind() != KindText || p.Delta() == "" {
		return nil
	}

	if _, err := s.Write([]byte(p.Delta())); err != nil {
		return err
	}

	return s.Flush()
}
