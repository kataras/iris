package ai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/sse"
)

// Decoder turns one upstream event into zero or more neutral parts.
//
// This is where a provider's dialect is understood, and it is the caller's code
// rather than the framework's: the framework has no business knowing what a
// particular vendor calls a delta this month.
type Decoder interface {
	Decode(ev sse.Event) ([]Part, error)
}

// DecoderFunc adapts a function to Decoder.
type DecoderFunc func(ev sse.Event) ([]Part, error)

// Decode implements Decoder.
func (f DecoderFunc) Decode(ev sse.Event) ([]Part, error) { return f(ev) }

// RelayOptions tune Relay.
type RelayOptions struct {
	// Read configures the upstream parser. Set Terminators to []string{"[DONE]"}
	// for an OpenAI-dialect upstream.
	//
	// When Terminators are configured Relay requires one, whatever
	// RequireTerminator says: an upstream that closes without its sentinel was
	// cut, and the client is told so instead of being handed three deltas and
	// a terminator that says the answer is complete. There is no opt-out by
	// design; a producer whose stream legitimately ends without the sentinel
	// has no sentinel, so leave Terminators empty.
	Read sse.ReadOptions

	// AllowNonEventStream accepts an upstream that answers 200 with something
	// other than text/event-stream, handing the body to the decoder as one
	// event instead of failing.
	//
	// Off by default, because the usual cause is a gateway that returned a JSON
	// error with a 200 status, and silently parsing that as an empty stream is
	// how a failed generation becomes a blank answer.
	AllowNonEventStream bool
}

// ErrNonEventStream reports that an upstream answered 200 with a body that is
// not an event stream.
var ErrNonEventStream = errors.New("iris: ai: upstream response is not an event stream")

// Relay reads an upstream event stream, decodes it and forwards the parts to r.
//
// It closes resp.Body. It classifies the response first, so a non-2xx upstream
// comes back as a typed *UpstreamError rather than as an empty stream, and it
// checks the content type, so a gateway that answers 200 with a JSON error does
// not silently become a blank answer. A failed read of the upstream body - a
// reset, a timeout, a stream cut mid-event or closed without the sentinel
// RelayOptions.Read.Terminators promised - comes back through Classify as a
// transient *UpstreamError wrapping the cause (io.ErrUnexpectedEOF for the last
// two), so the client gets a BAD_GATEWAY error frame rather than a terminator
// claiming the answer was complete; a client that hangs up mid-read comes back
// as ClassCanceled. Decoder errors are returned as they are: they are the
// application's own.
//
//	resp, err := httpClient.Do(req)
//	if ue := ai.Classify(resp, err); ue != nil {
//		return res.Close(ue)
//	}
//
//	return res.Close(ai.Relay(res, resp, decoder, ai.RelayOptions{
//		Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
//	}))
func Relay(r *Response, resp *http.Response, dec Decoder, opts ...RelayOptions) error {
	o := RelayOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}

	if r == nil {
		return errors.New("iris: ai: nil response")
	}
	if resp == nil {
		return errors.New("iris: ai: nil upstream response")
	}
	defer resp.Body.Close()

	if ue := Classify(resp, nil); ue != nil {
		return ue
	}

	if mediaType := context.TrimHeaderValue(resp.Header.Get("Content-Type")); mediaType != "text/event-stream" {
		if !o.AllowNonEventStream {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, DefaultMaxErrorBody))
			return fmt.Errorf("%w: content type %q: %s",
				ErrNonEventStream, mediaType, strings.TrimSpace(string(body)))
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, DefaultMaxErrorBody))
		if err != nil {
			return err
		}

		return relayParts(r, dec, sse.Event{Data: body})
	}

	if len(o.Read.Terminators) > 0 {
		// A sentinel the upstream promised and never sent is a cut stream.
		// Forced, with no opt-out, whenever terminators are configured: a
		// relay that accepts a missing sentinel hands its client three deltas
		// and a terminator that says the answer was complete.
		o.Read.RequireTerminator = true
	}

	for ev, err := range sse.Read(resp.Body, o.Read) {
		if err != nil {
			// The upstream body failed us - reset, timeout, cut mid-event,
			// missing sentinel - which is an upstream failure like any other:
			// typed, transient, BAD_GATEWAY (or DEADLINE_EXCEEDED for a
			// timeout) to the client. Classified rather than flattened, so a
			// caller that hung up mid-read (context.Canceled from the body)
			// stays ClassCanceled: not retried, not paged, no frame. errors.Is
			// still reaches the cause.
			return Classify(nil, err)
		}

		if err := relayParts(r, dec, ev); err != nil {
			return err
		}

		// A client that has gone away is the one case where continuing is pure
		// waste, unless the caller asked to keep metering the upstream.
		if ctxErr := r.Context().Err(); r.opts.OnDisconnect == DisconnectCancel && ctxErr != nil {
			// Canceled whatever ended it. A server-side deadline on the request
			// lands here as context.DeadlineExceeded, which Classify would call
			// transient and a caller would retry - but the response is over and
			// nobody is waiting for the second attempt either. The cause is
			// kept, so errors.Is still reaches it.
			ue := Classify(nil, ctxErr)
			ue.Class = ClassCanceled
			return ue
		}
	}

	return nil
}

func relayParts(r *Response, dec Decoder, ev sse.Event) error {
	parts, err := dec.Decode(ev)
	if err != nil {
		return err
	}

	for _, p := range parts {
		if err := r.Send(p); err != nil {
			return err
		}
	}

	return nil
}
