package ai

import (
	stdcontext "context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/sse"
)

// ErrDropPart tells Open to discard the current part and keep going. Return it
// from Options.OnPart.
var ErrDropPart = errors.New("iris: ai: drop part")

// Disconnect decides what a vanished client does to the work in progress.
type Disconnect uint8

const (
	// DisconnectCancel ends the turn as soon as the client goes away. It is the
	// default because an abandoned generation burns tokens nobody will read.
	DisconnectCancel Disconnect = iota

	// DisconnectDetach keeps consuming the upstream after the client has gone,
	// so OnFinish still records what was actually spent.
	//
	// It does not mean the handler returns and work continues in the background.
	// The handler still runs to completion; only the client is gone.
	DisconnectDetach
)

// Options configure one turn. Start from DefaultOptions and change what you
// need: a zero field means "off" rather than "default".
type Options struct {
	// Stream forces or forbids streaming. Nil negotiates on the request's Accept
	// header. Pass &req.Stream to honour an OpenAI-shaped request body.
	//
	// It is a pointer rather than a bool because the zero value has to be
	// distinguishable from "not configured". A plain bool in a struct-valued
	// option silently means the opposite of what half of its authors intend.
	Stream *bool

	// Encoder renders parts on the wire. Nil selects SSEJSON.
	Encoder Encoder

	// SSE configures the transport when the turn streams over an event stream.
	SSE sse.Options

	// Coalesce batches consecutive text and reasoning deltas, emitting at most
	// one of each per interval. Zero disables it.
	//
	// It batches parts rather than bytes, because a part carries structure that
	// concatenating two encoded frames would destroy.
	Coalesce time.Duration

	// OnPart runs before each part is encoded. One hook, three outcomes:
	//
	//	(p, nil)          forward it, possibly rewritten
	//	(_, ErrDropPart)  drop this part and keep streaming
	//	(_, any other)    fail the turn
	//
	// This is the whole policy surface, and it is block-only by construction.
	// You cannot revise what you have already sent, so the restriction lives in
	// the type rather than in a warning: a hook that could rewrite history would
	// work in tests and fail on the wire.
	//
	// It also sees the terminal error that Close writes, as a KindError part, so
	// the failure a client is told about goes through the same gate everything
	// else does. Returning ErrDropPart there writes no error frame at all, and
	// the stream still gets its terminator.
	OnPart func(ctx *context.Context, p Part) (Part, error)

	// OnDisconnect decides what a vanished client does to the upstream call.
	OnDisconnect Disconnect

	// OnFinish runs exactly once after the turn ends, under
	// context.WithoutCancel, with the assembled answer and the usage. It runs on
	// success, on failure and on client disconnect alike, and for a turn the
	// handler abandoned without Close (see Result.Abandoned).
	//
	// Persistence, billing and metrics belong here. The stream is an
	// optimisation and never the source of truth: a design in which the only
	// copy of the answer is on the wire loses answers.
	OnFinish func(ctx stdcontext.Context, r *Result)

	// Redact rewrites the client-facing text of an error before it is sent.
	//
	// It runs last, over text the rest/errors pipeline has already made safe, and
	// it exists for the case that pipeline cannot cover on its own: an error the
	// application deliberately mapped and described, whose description still
	// names a provider, a model or a host that the client has no business
	// learning.
	//
	// A strings.Replacer is the natural implementation and is safe for concurrent
	// use. Hold it in an atomic.Pointer if the vocabulary changes at runtime.
	//
	// It applies to error text only. Rewriting the model's own output is
	// OnPart's job, where the decision can be made per part.
	Redact func(string) string

	// KeepParts retains every part in Result.Parts.
	//
	// Off by default: a long generation would otherwise be held twice, once as
	// assembled text and once as every delta that built it.
	KeepParts bool
}

// DefaultOptions returns the recommended settings.
func DefaultOptions() Options {
	o := Options{
		Encoder:  SSEJSON(),
		SSE:      sse.DefaultOptions(),
		Coalesce: 100 * time.Millisecond,
	}

	// Open installs its own, which keeps a classified upstream failure's
	// canonical code instead of flattening every failure to an internal error.
	// Leaving the transport default here would silently win, because it is not
	// nil.
	o.SSE.ErrorFrame = nil

	return o
}

// Result is what a turn produced. It is handed to OnFinish and available from
// Response.Result.
type Result struct {
	// Streaming reports which shape the turn took.
	Streaming bool
	// Committed reports whether any byte reached the client. It is the line
	// between an error response and a truncated answer.
	Committed bool
	// Disconnected reports that the client went away before the turn ended.
	Disconnected bool
	// Abandoned reports that the handler returned without calling Close. The
	// framework ended the turn before the response was recycled: nothing was
	// written after the handler returned, a part still pending was dropped,
	// and the client got no terminator. Committed says whether anything had
	// reached the client before that.
	Abandoned bool

	// Text is the assembled answer.
	Text string
	// Reasoning is the assembled reasoning trace.
	Reasoning string
	// ToolCalls are the tool calls the model asked for, as the parts that
	// carried them: a tool-call part already holds the id, the name and the raw
	// arguments, so a second type for the same data would only be one more thing
	// to keep in step.
	ToolCalls []Part
	// Usage is what the turn cost, accumulated across every usage part.
	Usage Usage
	// Parts is every part, when Options.KeepParts is set.
	Parts []Part
	// Err is why the turn ended badly, or nil.
	Err error

	// TTFP is the time to the first part, which is the latency a reader
	// actually perceives. Total duration is dominated by how long the answer is.
	TTFP time.Duration
	// Duration is the whole turn.
	Duration time.Duration
}

// Message is the non-streaming projection of a turn: what a client that asked
// for a single document receives.
type Message struct {
	Text      string `json:"text,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	// ToolCalls marshal through Part, so the same object shape appears here and
	// in a streamed tool frame.
	ToolCalls []Part `json:"tool_calls,omitempty"`
	Usage     *Usage `json:"usage,omitempty"`
}

// Response is one turn in progress.
type Response struct {
	ctx  *context.Context
	opts Options
	enc  Encoder

	stream *sse.Stream
	sink   Sink

	mu        sync.Mutex
	closed    bool
	started   time.Time
	firstPart time.Time
	text      strings.Builder
	reasoning strings.Builder
	toolCalls []Part
	usage     Usage
	parts     []Part

	// pending holds the one part being coalesced. Consecutive deltas of the same
	// kind and id merge into it; anything else flushes it first.
	pending    Part
	hasPending bool
	lastFlush  time.Time

	stop     chan struct{}
	stopOnce sync.Once
	drained  chan struct{}

	// closing is set by Close under mu, before it releases the lock to write its
	// terminal frames; closeDone is closed when Close has returned. release
	// waits on it, so a handler that returns during a Close on another
	// goroutine cannot let the writer be recycled mid-write.
	closing   bool
	closeDone chan struct{}

	finishOnce sync.Once
	result     Result
}

// Open begins a turn.
//
// One handler serves both shapes. When the request asks to stream - through
// Options.Stream, or through an Accept header naming text/event-stream - parts
// go out as they are produced. When it does not, they are assembled and written
// as a single JSON document by Close. Nothing in the handler between those two
// points changes.
//
// Like sse.Open it writes nothing: an authorization failure or a policy veto
// after Open still produces an ordinary status code with an ordinary body.
func Open(ctx *context.Context, opts ...Options) (*Response, error) {
	o := DefaultOptions()
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Encoder == nil {
		o.Encoder = SSEJSON()
	}

	r := &Response{
		ctx:       ctx,
		opts:      o,
		enc:       o.Encoder,
		started:   time.Now(),
		lastFlush: time.Now(),
		stop:      make(chan struct{}),
		drained:   make(chan struct{}),
		closeDone: make(chan struct{}),
	}
	r.result.Streaming = wantsStream(ctx, o)

	if r.opts.SSE.ErrorFrame == nil {
		// A classified upstream failure keeps its canonical code this way, so a
		// rate limit reaches the client as RESOURCE_EXHAUSTED rather than as a
		// generic internal error.
		r.opts.SSE.ErrorFrame = r.errorFrame
	}

	if r.result.Streaming {
		if o.Encoder.ContentType() == "" {
			stream, err := sse.Open(ctx, r.opts.SSE)
			if err != nil {
				return nil, err
			}
			r.stream = stream
			r.sink = &sseSink{stream: stream}
		} else {
			r.sink = &rawSink{ctx: ctx, enc: o.Encoder}
		}

		if err := o.Encoder.Open(r.sink); err != nil {
			return nil, err
		}
	}

	// A handler that returns without Close leaves the drain ticking against a
	// Context the router is about to recycle; with a part pending, the next
	// tick would commit a whole event stream into someone else's response. For
	// either shape, OnFinish is still owed its record. Registered after
	// sse.Open's own hook, so it runs before it.
	ctx.OnRelease(r.release)

	if o.Coalesce > 0 && r.result.Streaming {
		go r.drain()
	} else {
		close(r.drained)
	}

	return r, nil
}

// release ends a turn the handler forgot to close. It runs from
// Context.OnRelease, before the response writer is recycled: the drain is
// stopped and waited for, the turn is marked closed so a late Send reports
// sse.ErrClosed, and OnFinish runs with Result.Abandoned set, because the
// accounting leg is owed its record however the turn ended. Nothing pending is
// flushed: the handler has returned, and a part it never got to see delivered
// is not one to write into whatever request holds this Context next.
//
// A Close in flight on another goroutine is waited for instead, so its
// terminal frames reach the wire before the writer goes back to its pool.
func (r *Response) release() {
	r.stopOnce.Do(func() { close(r.stop) })
	<-r.drained

	r.mu.Lock()
	if r.closing {
		r.mu.Unlock()
		<-r.closeDone
		return
	}
	if r.closed {
		r.mu.Unlock()
		return
	}

	r.closed = true
	r.result.Abandoned = true
	r.fillResultLocked()
	r.mu.Unlock()

	r.runFinish()
}

// wantsStream decides the shape of the turn.
func wantsStream(ctx *context.Context, o Options) bool {
	if o.Stream != nil {
		return *o.Stream
	}

	return strings.Contains(ctx.Header("Accept"), "text/event-stream")
}

// Streaming reports which shape this turn took.
func (r *Response) Streaming() bool { return r.result.Streaming }

// Committed reports whether any byte has reached the client. A failover ladder
// must consult it before retrying an upstream: after the first delta, a second
// attempt splices two answers into one response.
func (r *Response) Committed() bool {
	if r.stream != nil {
		return r.stream.Committed()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.committedLocked()
}

// committedLocked is the body of Committed for callers that already hold the
// mutex. sync.Mutex is not reentrant, so the two cannot share one method.
func (r *Response) committedLocked() bool {
	if r.stream != nil {
		return r.stream.Committed()
	}

	if s, ok := r.sink.(*rawSink); ok {
		return s.committed
	}

	return false
}

// Context is done when the client disconnects or the turn ends. It is nil-safe
// to select on for a non-streaming turn, where it follows the request.
func (r *Response) Context() stdcontext.Context {
	if r.stream != nil {
		return r.stream.Context()
	}

	return r.ctx.Request().Context()
}

// Send renders one part.
//
// A KindError part on a streaming turn goes through the same policy as the
// error given to Close - reported through rest/errors, redacted unless
// deliberately mapped, rewritten by Options.Redact - and reaches the wire as
// the resulting envelope. Result.Err keeps the original. When the client has
// already gone there is nothing to report, and the part is dropped.
func (r *Response) Send(p Part) error {
	if p.IsZero() {
		return nil
	}

	// Closed first, so a hook with side effects does not run for a part that
	// can never be sent. The lock is released before the hook runs, because a
	// hook may call back into this Response; the check is repeated below.
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return sse.ErrClosed
	}

	if hook := r.opts.OnPart; hook != nil {
		rewritten, err := hook(r.ctx, p)
		if err != nil {
			if errors.Is(err, ErrDropPart) {
				return nil
			}

			return err
		}
		p = rewritten
	}

	// The policy runs outside the lock: it calls the logger and the collector.
	var clientSafe Part
	if p.Kind() == KindError && r.result.Streaming {
		if reported := r.reportError(p.Err()); reported != nil {
			clientSafe = Error(reported)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return sse.ErrClosed
	}

	r.accumulateLocked(p)

	if !r.result.Streaming {
		return nil
	}

	if p.Kind() == KindError {
		if clientSafe.IsZero() {
			return nil // nobody is listening, or nothing worth telling them.
		}
		p = clientSafe
	}

	if r.opts.Coalesce > 0 && mergeable(p) {
		if r.hasPending && sameRun(r.pending, p) {
			r.pending = merge(r.pending, p)
			return nil
		}

		if err := r.flushPendingLocked(); err != nil {
			return err
		}

		r.pending, r.hasPending = p, true
		return nil
	}

	if err := r.flushPendingLocked(); err != nil {
		return err
	}

	return r.encodeLocked(p)
}

// accumulateLocked keeps the assembled answer up to date, whatever the shape.
func (r *Response) accumulateLocked(p Part) {
	if r.firstPart.IsZero() {
		r.firstPart = time.Now()
	}

	if r.opts.KeepParts {
		r.parts = append(r.parts, p)
	}

	switch p.Kind() {
	case KindText:
		r.text.WriteString(p.Delta())
	case KindReasoning:
		r.reasoning.WriteString(p.Delta())
	case KindToolCall:
		r.toolCalls = append(r.toolCalls, p)
	case KindUsage:
		r.usage.Add(p.Usage())
	case KindError:
		if r.result.Err == nil {
			r.result.Err = p.Err()
		}
	}
}

func (r *Response) encodeLocked(p Part) error {
	return r.enc.Encode(r.sink, p)
}

func (r *Response) flushPendingLocked() error {
	if !r.hasPending {
		return nil
	}

	p := r.pending
	r.pending, r.hasPending = Part{}, false
	r.lastFlush = time.Now()

	return r.encodeLocked(p)
}

// mergeable reports whether a part is a text-like delta that can be batched.
func mergeable(p Part) bool {
	return p.Kind() == KindText || p.Kind() == KindReasoning
}

// sameRun reports whether b continues the run a belongs to.
func sameRun(a, b Part) bool {
	return a.Kind() == b.Kind() && a.ID() == b.ID()
}

func merge(a, b Part) Part {
	a.delta += b.delta
	return a
}

// drain makes Coalesce a ceiling rather than a floor. Without it a buffered
// delta is only emitted when the next one arrives, so text that already reached
// the server sits unsent through exactly the pause a reader notices.
func (r *Response) drain() {
	defer close(r.drained)

	interval := r.opts.Coalesce
	tick := interval / 4
	if tick < time.Millisecond {
		tick = time.Millisecond
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	done := r.Context().Done()

	for {
		select {
		case <-r.stop:
			return
		case <-done:
			return
		case <-ticker.C:
			r.mu.Lock()
			if !r.closed && r.hasPending && time.Since(r.lastFlush) >= interval {
				_ = r.flushPendingLocked()
			}
			r.mu.Unlock()
		}
	}
}

// Result returns a snapshot of what the turn produced so far. It is complete
// after Close.
//
// It is a copy, taken under the lock: Close keeps writing the turn's own record
// until OnFinish has run, and a pointer into it would race those writes. The
// ToolCalls and Parts slices are shared with the turn and are read-only.
func (r *Response) Result() Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fillResultLocked()
	return r.result
}

func (r *Response) fillResultLocked() {
	r.result.Text = r.text.String()
	r.result.Reasoning = r.reasoning.String()
	r.result.ToolCalls = r.toolCalls
	r.result.Usage = r.usage
	r.result.Parts = r.parts
	// Monotonic: the non-streaming Close records the written document here and
	// has no sink to ask afterwards.
	r.result.Committed = r.result.Committed || r.committedLocked()

	if !r.firstPart.IsZero() {
		r.result.TTFP = r.firstPart.Sub(r.started)
	}
	r.result.Duration = time.Since(r.started)
}

// Close ends the turn: it flushes what is pending, writes the terminator or the
// assembled document, and runs OnFinish exactly once.
//
// When err is non-nil and nothing has reached the client, Close writes nothing
// and returns err, so the caller renders an ordinary HTTP error. After the first
// byte the status code is spent, so the failure is reported in-band in whatever
// form the encoder has: an error frame on SSE, an error line on NDJSON, and
// nothing on PlainText, which has no room for one. In every case it goes
// through rest/errors first, so the logger and the collector hear of it.
//
// A streaming turn with no parts at all still commits the response and writes
// the terminator, as the non-streaming shape writes an empty document: an
// empty answer is an answer, and a client must be able to tell it from a
// connection that dropped before the first byte.
func (r *Response) Close(err error) error {
	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closing = true
	defer close(r.closeDone)

	if err != nil && r.result.Err == nil {
		r.result.Err = err
	}

	flushErr := r.flushPendingLocked()
	r.closed = true

	if reqErr := r.ctx.Request().Context().Err(); reqErr != nil {
		r.result.Disconnected = true
	}

	streaming := r.result.Streaming
	r.fillResultLocked()
	r.mu.Unlock()

	r.stopOnce.Do(func() { close(r.stop) })
	<-r.drained

	var (
		closeErr          error
		documentCommitted bool
	)
	switch {
	case streaming && r.stream != nil:
		if encErr := r.enc.Close(r.sink, &r.result); encErr != nil && flushErr == nil {
			flushErr = encErr
		}
		if err == nil && !r.stream.Committed() {
			// A zero-part answer still gets its terminator.
			if commitErr := r.stream.Commit(); commitErr != nil && flushErr == nil {
				flushErr = commitErr
			}
		}
		// The terminal error goes through the same hook every other part does,
		// so OnPart is the whole policy surface rather than almost all of it.
		// Result.Err keeps the original: only what the client sees is rewritten.
		closeErr = r.stream.Close(r.applyErrorPolicy(err))
	case streaming:
		closeErr = r.closeRaw(err)
	default:
		closeErr = r.writeDocument(err)
		documentCommitted = closeErr == nil && err == nil
	}

	// The snapshot above was taken before any of this ran, and committing is
	// the last thing Close does: a zero-part streaming turn commits inside the
	// branch, and the document shape only reaches the wire there too. Ask the
	// sink now, under the lock, so the accounting leg is not told that nothing
	// was delivered.
	r.mu.Lock()
	r.result.Committed = r.result.Committed || documentCommitted || r.committedLocked()
	r.mu.Unlock()

	r.runFinish()

	if err != nil {
		return err
	}
	if flushErr != nil {
		return flushErr
	}

	return closeErr
}

// closeRaw ends a turn on a line-oriented dialect, mirroring what sse.Stream.Close
// does for an event stream: nothing on an error before the first byte, an error
// part (post-policy) and then the encoder's terminator after it, and a commit
// plus terminator for a zero-part answer.
func (r *Response) closeRaw(err error) error {
	sink := r.sink.(*rawSink)

	if err != nil && !sink.committed {
		// Nothing went out, so this is still an ordinary response and the
		// caller owns the status code.
		return err
	}

	if err != nil {
		if clientErr := r.applyErrorPolicy(err); clientErr != nil {
			if reported := r.reportError(clientErr); reported != nil {
				if encErr := r.enc.Encode(r.sink, Error(reported)); encErr != nil {
					return encErr
				}
			}
		}
	} else if commitErr := sink.commit(); commitErr != nil {
		return commitErr
	}

	if closeErr := r.enc.Close(r.sink, &r.result); closeErr != nil {
		return closeErr
	}

	return err
}

// applyErrorPolicy runs a terminal error through OnPart and returns what should
// reach the client: the error itself, a rewritten one, or nil to write no error
// frame at all.
func (r *Response) applyErrorPolicy(err error) error {
	hook := r.opts.OnPart
	if err == nil || hook == nil {
		return err
	}

	rewritten, hookErr := hook(r.ctx, Error(err))
	if hookErr != nil {
		if errors.Is(hookErr, ErrDropPart) {
			return nil // the hook chose to say nothing.
		}

		// The hook failed on its own account. Reporting its failure instead of
		// the original would hide the thing that actually went wrong.
		return err
	}

	if rewritten.Kind() == KindError && rewritten.Err() != nil {
		return rewritten.Err()
	}

	return err
}

// writeDocument renders a non-streaming turn as one JSON document.
func (r *Response) writeDocument(err error) error {
	if err != nil {
		// Nothing was streamed, so this is still an ordinary error response and
		// the caller owns the status code and the body.
		return err
	}

	msg := Message{
		Text:      r.result.Text,
		Reasoning: r.result.Reasoning,
		ToolCalls: r.result.ToolCalls,
	}
	if !r.result.Usage.IsZero() {
		u := r.result.Usage
		msg.Usage = &u
	}

	return r.ctx.JSON(msg)
}

// runFinish calls OnFinish once, under a context that outlives the request.
func (r *Response) runFinish() {
	r.finishOnce.Do(func() {
		if r.opts.OnFinish == nil {
			return
		}

		// WithoutCancel because this is the accounting leg: a client that hung up
		// is the case where recording what the upstream already spent matters
		// most, and the request context is cancelled exactly then.
		r.opts.OnFinish(r.ctx.WithoutCancel(), &r.result)
	})
}

// --------------------------------------------------------------------------
// Sinks
// --------------------------------------------------------------------------

// sseSink carries frames on an event stream.
type sseSink struct {
	stream *sse.Stream
}

func (s *sseSink) Frame(ev sse.Event) error  { return s.stream.Send(ev) }
func (s *sseSink) Write([]byte) (int, error) { return 0, ErrNotRaw }
func (s *sseSink) Flush() error              { return nil }

// rawSink carries bytes on a plain streaming response, for a line-oriented
// dialect. It commits the same adjustments sse.Open does, on the first write.
type rawSink struct {
	ctx       *context.Context
	enc       Encoder
	committed bool

	// w is the writer as it was at commit time. Writes after the first go
	// through it, not through ctx, for the same reason sse.Stream keeps its own:
	// the drain goroutine must never read a recycled Context.
	w http.ResponseWriter
}

func (s *rawSink) Frame(sse.Event) error { return ErrNotFramed }

func (s *rawSink) Write(p []byte) (int, error) {
	if err := s.commit(); err != nil {
		return 0, err
	}

	return s.w.Write(p)
}

func (s *rawSink) Flush() error {
	if !s.committed {
		return nil
	}

	err := context.FlushResponseWriter(s.w)
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}

	return err
}

func (s *rawSink) commit() error {
	if s.committed {
		return nil
	}

	if err := s.ctx.DisableResponseBuffering(); err != nil {
		return err
	}

	w := s.ctx.ResponseWriter()
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	if req := s.ctx.Request(); req != nil && req.ProtoMajor == 1 && req.Body != nil {
		_ = rc.EnableFullDuplex()
	}

	h := w.Header()
	h.Set(context.ContentTypeHeaderKey, s.enc.ContentType())
	s.enc.Header(h)
	h.Del(context.ContentLengthHeaderKey)

	s.ctx.StatusCode(http.StatusOK)
	s.w = w
	s.committed = true

	return nil
}
