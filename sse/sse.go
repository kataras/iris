// Package sse implements Server-Sent Events for Iris, in both directions: a
// push-based response stream, and a tolerant reader for consuming somebody
// else's.
//
// It is transport only. It ships no application vocabulary, no replay store and
// no pub/sub fan-out. Those are application concerns that vary per service, and
// keeping them out is what lets this package stay small enough to be correct.
// The ai package builds the AI-shaped layer on top.
//
// The writer half exists because the details that break a stream are neither
// obvious nor local. Compression buffers; a response recorder buffers; the
// server's absolute WriteTimeout guillotines a long generation; reverse proxies
// buffer unless told not to; and a client that vanishes without closing the
// connection stays invisible until the kernel gives up retransmitting, which
// takes about fifteen minutes. Open handles all of it for one response.
//
// The reader half exists because every Go SDK that consumes a model's output
// reimplements event-stream parsing, usually by matching a "data: " prefix, and
// then breaks against a producer that omits the space.
//
//	stream, err := sse.Open(ctx)
//	if err != nil {
//		return
//	}
//	defer stream.Close(nil)
//
//	for token := range tokens {
//		if err := stream.Send(sse.Event{Name: "token", Data: []byte(token)}); err != nil {
//			return
//		}
//	}
package sse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Errors reported by this package. Match them with errors.Is.
var (
	// ErrNotStreamable reports that this response cannot flush, so nothing
	// written to it would reach the client until the handler returned.
	//
	// The usual cause is Timeout.HandlerTimeout, which wraps the router in
	// http.TimeoutHandler; its response writer implements neither http.Flusher
	// nor http.Hijacker. Open reports it before writing a byte, so the handler
	// can still answer with an ordinary status code.
	ErrNotStreamable = errors.New("iris: sse: response writer cannot flush")

	// ErrInvalidField reports a CR, LF or NUL in a field that is terminated by a
	// newline on the wire. Without the check a caller-supplied value forges
	// frames, and LastEventID echoes something the client chose.
	ErrInvalidField = errors.New("iris: sse: field contains CR, LF or NUL")

	// ErrSlowClient reports that a single Send exceeded SendTimeout, meaning the
	// client is not draining the connection.
	ErrSlowClient = errors.New("iris: sse: write deadline exceeded")

	// ErrClosed reports a Send on a stream that has already been closed.
	ErrClosed = errors.New("iris: sse: stream closed")

	// ErrTooLarge reports an event larger than ReadOptions.MaxEventBytes.
	ErrTooLarge = errors.New("iris: sse: event exceeds the maximum size")

	// ErrNoTerminator reports a stream that ended cleanly without any of the
	// configured ReadOptions.Terminators while RequireTerminator was set. It
	// matches io.ErrUnexpectedEOF as well, because that is what it is: the
	// producer stopped before it said it was done.
	ErrNoTerminator = fmt.Errorf("iris: sse: stream ended without a terminator: %w", io.ErrUnexpectedEOF)
)

// Options configure one stream. Start from DefaultOptions and change what you
// need: a zero field means "off" rather than "default", so that disabling the
// heartbeat or the terminator stays expressible.
type Options struct {
	// Heartbeat is the longest the stream may go without writing anything.
	// Default 15s. Zero disables it.
	//
	// 15s is not a round number, it is the largest value that survives every
	// common intermediary at once: nginx proxy_read_timeout is 60s, an AWS
	// Application Load Balancer idles out at 60s, Cloudflare returns a 524 at
	// 100s, and Heroku's router applies a rolling 55s. Note that a Google Cloud
	// Application Load Balancer applies its backend timeout to the whole request
	// and response, so no heartbeat rescues it; raise timeoutSec there instead.
	//
	// Its larger job is liveness, not keep-alive. Only a real FIN or RST, or a
	// failing write, cancels a request context. A client that disappears without
	// one - a closed lid, a dropped Wi-Fi link, an evicted NAT entry - stays
	// invisible until TCP gives up, which on Linux means tcp_retries2 = 15 and
	// roughly 924 seconds. Writing something is what lets the kernel fail and
	// the handler find out, and it is why this bounds idle time rather than
	// firing on a fixed tick.
	Heartbeat time.Duration

	// HeartbeatFrame is written verbatim when the stream has been idle for
	// Heartbeat. Default ": ping\n\n", an SSE comment, which conforming clients
	// ignore.
	//
	// It is configurable rather than fixed because not every client conforms:
	// sashabaranov/go-openai, widely used against OpenAI-compatible servers,
	// counts comment frames toward emptyMessagesLimit and fails the stream
	// instead of ignoring them. Serving such a client, send a real event.
	HeartbeatFrame []byte

	// Retry advertises how long the client should wait before reconnecting,
	// written once with the first event. Default 3s. Zero omits the field and
	// leaves the client's own default in place.
	Retry time.Duration

	// AutoID stamps a monotonically increasing id on every event that does not
	// carry one. Default true.
	//
	// It makes gaps detectable within one connection, for the cost of a few
	// bytes per frame: a client that sees id 7 after id 5 knows it missed one.
	// It is not resumption. The counter starts at 1 on every stream, so a
	// Last-Event-ID from a previous connection means nothing to it; a resumable
	// stream needs ids of the application's own, tied to its store.
	AutoID bool

	// Terminator is the event name of the frame Close writes. Default "done".
	// Empty writes no terminal frame.
	//
	// A reserved event name, never an in-band data sentinel. A sentinel such as
	// [DONE] travels on the same channel as the payload, so a model that happens
	// to emit the literal string ends the client's stream.
	Terminator string

	// ErrorFrame renders the event Close writes when the stream failed after it
	// had already committed. Nil selects ReportErrorFrame.
	//
	// The default runs the error through rest/errors, so a mid-stream failure is
	// mapped, logged, collected and redacted exactly like any other failure in
	// the application. Set RawErrorFrame to send the error's own text instead,
	// which is safe only when that text did not come from an upstream.
	ErrorFrame ErrorFrameFunc

	// SendTimeout bounds one Send with a write deadline. Default 30s; zero
	// disables it.
	//
	// This is the backpressure mechanism, and it is a deadline rather than a
	// queue on purpose. Send is synchronous, so there is no buffer of ours to
	// budget; a client that will not drain shows up as a write that does not
	// complete. Exceeding it fails the stream with ErrSlowClient and never skips
	// an event, because a dropped delta corrupts the assembled message with no
	// way for the client to detect it.
	SendTimeout time.Duration
}

// DefaultOptions returns the recommended settings. See each field for why.
func DefaultOptions() Options {
	return Options{
		Heartbeat:      15 * time.Second,
		HeartbeatFrame: []byte(": ping\n\n"),
		Retry:          3 * time.Second,
		AutoID:         true,
		Terminator:     "done",
		SendTimeout:    30 * time.Second,
		ErrorFrame:     ReportErrorFrame,
	}
}

// withDefaults fills in the fields that cannot meaningfully be zero.
func (o Options) withDefaults() Options {
	if o.Heartbeat > 0 && len(o.HeartbeatFrame) == 0 {
		o.HeartbeatFrame = []byte(": ping\n\n")
	}

	if o.ErrorFrame == nil {
		// A nil renderer would silently drop the only notice a client gets that
		// the answer it is reading was cut short.
		o.ErrorFrame = ReportErrorFrame
	}

	return o
}

// Event is one frame. A zero field is absent from the wire.
type Event struct {
	// ID is the "id" field. It becomes the client's Last-Event-ID.
	ID string
	// Name is the "event" field. Empty means the default, "message", which is
	// what an EventSource onmessage handler receives.
	Name string
	// Data is the payload. A multi-line value is split across several "data"
	// fields and rejoined by the client with "\n" between them.
	//
	// A nil Data with a Comment writes a comment-only frame, which dispatches no
	// event. A non-nil but empty Data dispatches an event with an empty payload.
	Data []byte
	// Retry sets the client's reconnection time for this and later events.
	Retry time.Duration
	// Comment is written as an SSE comment before the fields. Conforming clients
	// ignore it; it exists for keep-alives and for human-readable markers.
	Comment string
}

// validate rejects the characters that would let a field value break framing.
func (ev *Event) validate() error {
	const forbidden = "\r\n\x00"

	if strings.ContainsAny(ev.ID, forbidden) ||
		strings.ContainsAny(ev.Name, forbidden) ||
		strings.ContainsAny(ev.Comment, forbidden) {
		return ErrInvalidField
	}

	return nil
}

// appendTo encodes ev in the event-stream format and appends it to dst.
func (ev *Event) appendTo(dst []byte) []byte {
	if ev.Comment != "" {
		dst = append(dst, ':', ' ')
		dst = append(dst, ev.Comment...)
		dst = append(dst, '\n')
	}

	if ev.ID != "" {
		dst = append(dst, "id: "...)
		dst = append(dst, ev.ID...)
		dst = append(dst, '\n')
	}

	if ev.Name != "" {
		dst = append(dst, "event: "...)
		dst = append(dst, ev.Name...)
		dst = append(dst, '\n')
	}

	if ev.Retry > 0 {
		dst = append(dst, "retry: "...)
		dst = strconv.AppendInt(dst, ev.Retry.Milliseconds(), 10)
		dst = append(dst, '\n')
	}

	if ev.Data != nil {
		// Every line of the payload becomes its own data field. A lone CR and a
		// CRLF are line terminators in this format too, so normalise both to LF
		// before splitting; otherwise a Windows-style payload emits a stray CR at
		// the end of each line and the client reassembles it verbatim.
		data := ev.Data
		if bytes.IndexByte(data, '\r') >= 0 {
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
			data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
		}

		for line := range bytes.SplitSeq(data, []byte("\n")) {
			dst = append(dst, "data: "...)
			dst = append(dst, line...)
			dst = append(dst, '\n')
		}
	}

	// The blank line is what dispatches the event.
	return append(dst, '\n')
}
