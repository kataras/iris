package sse

import (
	stdcontext "context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
)

// Open prepares an SSE response and writes nothing.
//
// The protocol headers and every adjustment below are applied on the first Send,
// not here, so an authorization check, a validation failure or a policy veto
// after Open still produces an ordinary status code with an ordinary body, and a
// handler that emits nothing and returns leaves the response untouched. That one
// decision is what every complaint about a framework's SSE support comes back
// to: open the stream before the handler runs and you can no longer return a
// 401, set a header, or let an error handler fire.
//
// Open does verify up front that this response can flush, because the answer is
// only available at flush time otherwise, which is far too late: under
// Timeout.HandlerTimeout the writer silently buffers into a 503 HTML page.
// ErrNotStreamable is reported with zero bytes written.
//
// On the first Send, and for this response only, it then does the five things no
// middleware can guess:
//
//   - removes every buffering response-writer wrapper, so compression and the
//     response recorder that accesslog installs stop holding the events;
//   - clears the server's absolute WriteTimeout, which is a total deadline and
//     would otherwise guillotine a long generation mid-sentence;
//   - enables full duplex on HTTP/1 so a POST body can still be read after the
//     response has started, which is how every LLM API is actually called;
//   - sets Cache-Control: no-cache, no-transform and X-Accel-Buffering: no, and
//     deletes any Content-Length;
//   - sets Connection: keep-alive only on HTTP/1, where it is meaningful. It is
//     a protocol error on HTTP/2.
//
// Close the stream, normally with defer.
func Open(ctx *context.Context, opts ...Options) (*Stream, error) {
	o := DefaultOptions()
	if len(opts) > 0 {
		o = opts[0]
	}
	o = o.withDefaults()

	// The terminator is an event name on the wire, so it gets the same framing
	// check as any field Send accepts.
	if strings.ContainsAny(o.Terminator, "\r\n\x00") {
		return nil, ErrInvalidField
	}

	// The pre-flight flusher check. http.ResponseController only reports
	// ErrNotSupported when a flush is attempted, and by then the status code is
	// committed and the handler has no way to answer honestly.
	if _, canFlush := context.UnwrapResponseWriter(ctx.ResponseWriter()).(http.Flusher); !canFlush {
		return nil, ErrNotStreamable
	}

	reqCtx := ctx.Request().Context()
	streamCtx, cancel := stdcontext.WithCancelCause(reqCtx)

	s := &Stream{
		ctx:         ctx,
		opts:        o,
		stdCtx:      streamCtx,
		cancel:      cancel,
		stop:        make(chan struct{}),
		closeDone:   make(chan struct{}),
		lastEventID: readLastEventID(ctx),
	}

	// A handler that returns without Close leaves the heartbeat ticking against
	// a Context the router is about to recycle. The hook ends the stream first.
	ctx.OnRelease(s.release)

	return s, nil
}

// readLastEventID reads the resume cursor from the header and from the query.
//
// Both spellings are load-bearing. A browser EventSource sends the header and
// cannot be made to send anything else, while a client built on fetch - which is
// every LLM SDK, because EventSource cannot POST a body - usually has to put it
// in the URL. The value is sanitised because it is echoed back on every frame.
func readLastEventID(ctx *context.Context) string {
	id := ctx.Header("Last-Event-ID")
	if id == "" {
		id = ctx.Query().Get("lastEventId")
	}
	if id == "" {
		return ""
	}

	if strings.ContainsAny(id, "\r\n\x00") {
		return ""
	}

	return id
}

// Stream is one open Server-Sent Events response.
//
// It is safe for concurrent use: Send, Comment and Close take the same lock the
// heartbeat does, so a keep-alive can never interleave with a partially written
// frame.
type Stream struct {
	ctx  *context.Context
	opts Options

	// w is the response writer as it was at commit time. Every write after the
	// first goes through it rather than through ctx, so the heartbeat goroutine
	// never reads a Context that may already belong to another request.
	w http.ResponseWriter

	mu        sync.Mutex
	buf       []byte // reused frame buffer, guarded by mu.
	lastWrite time.Time
	autoID    uint64
	closed    bool
	err       error // why the stream ended, once closed. Guarded by mu.

	// retryAdvertised records that the reconnection time went out with a frame,
	// which is once per stream and not necessarily on the committing one:
	// Commit writes headers without a frame.
	retryAdvertised bool

	committed atomic.Bool

	stdCtx stdcontext.Context
	cancel stdcontext.CancelCauseFunc

	stop     chan struct{}
	stopOnce sync.Once
	beating  bool
	beatDone chan struct{}

	// closing is set by Close under mu before it lets go of the lock to render
	// the error frame; closeDone is closed when Close has returned. release
	// waits on it, so a handler that returns during a Close on another goroutine
	// lets that Close finish its frames before the writer is recycled.
	closing   bool
	closeDone chan struct{}

	// coalescer, when set, owns the batching of named events. Send flushes it
	// before writing anything of its own, so a direct write can never overtake
	// text that is already buffered.
	coalescer *Coalescer

	lastEventID string
}

// tickInterval keeps a ticker period positive. time.NewTicker panics on zero,
// and a caller is entitled to ask for a very short interval.
func tickInterval(d time.Duration) time.Duration {
	if quarter := d / 4; quarter > time.Millisecond {
		return quarter
	}

	return time.Millisecond
}

// Send writes one event and flushes it. The first Send commits
// 200 text/event-stream.
//
// It rejects CR, LF and NUL in ID, Name and Comment with ErrInvalidField: those
// values are terminated by a newline on the wire, so without the check a
// caller-supplied id forges frames.
func (s *Stream) Send(ev Event) error {
	if err := ev.validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.coalescer != nil {
		if err := s.coalescer.flushLocked(); err != nil {
			return err
		}
	}

	return s.sendLocked(ev)
}

// SendJSON marshals v and sends it as the data of a named event.
func (s *Stream) SendJSON(name string, v any) error {
	buf, err := marshalJSONTo(v)
	if err != nil {
		return err
	}
	defer jsonutil.ReleaseBuffer(buf) // Send copies the bytes synchronously.

	return s.Send(Event{Name: name, Data: buf.Bytes()})
}

// Comment writes an SSE comment. Conforming clients ignore it.
func (s *Stream) Comment(text string) error {
	return s.Send(Event{Comment: text})
}

func (s *Stream) sendLocked(ev Event) error {
	if s.closed {
		return ErrClosed
	}

	if s.opts.AutoID && ev.ID == "" && ev.Data != nil {
		s.autoID++
		ev.ID = strconv.FormatUint(s.autoID, 10)
	}

	if !s.committed.Load() {
		if err := s.commitLocked(); err != nil {
			return err
		}
	}

	if !s.retryAdvertised {
		// Advertise the reconnection time once, with the first frame.
		s.retryAdvertised = true
		if ev.Retry == 0 {
			ev.Retry = s.opts.Retry
		}
	}

	s.buf = ev.appendTo(s.buf[:0])
	return s.writeLocked(s.buf)
}

// Commit writes the status line and the stream headers now, with no event.
//
// Send does this on its own, so most handlers never call it. It exists for a
// response that is a stream whether or not an event ever arrives - an answer
// that turned out to be empty - so that Close can still write the terminator
// and the client can tell an empty answer from a connection that dropped
// before the first byte. The ai package uses it for a zero-part turn.
//
// Everything Open defers to the first Send happens here instead. The
// reconnection time is still advertised with the first event.
func (s *Stream) Commit() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}
	if s.committed.Load() {
		return nil
	}

	return s.commitLocked()
}

// commitLocked applies everything that must happen before the first byte.
func (s *Stream) commitLocked() error {
	if err := s.ctx.DisableResponseBuffering(); err != nil {
		return err
	}

	w := s.ctx.ResponseWriter()
	rc := http.NewResponseController(w)

	// A total write deadline is wrong for a stream: it is not idle-based, so it
	// ends a long answer mid-sentence. Clearing it per response is exactly what
	// this control exists for.
	_ = rc.SetWriteDeadline(time.Time{})

	if r := s.ctx.Request(); r != nil && r.ProtoMajor == 1 && r.Body != nil {
		// HTTP/1 servers refuse to read a request body once the response has
		// started, unless told otherwise. Every LLM API is a POST with a body.
		_ = rc.EnableFullDuplex()
	}

	h := w.Header()
	h.Set(context.ContentTypeHeaderKey, "text/event-stream")
	// no-transform is the half that matters against intermediaries: it asks
	// proxies not to re-encode, which is what makes them buffer.
	h.Set("Cache-Control", "no-cache, no-transform")
	// nginx and several managed proxies buffer proxied responses by default.
	h.Set("X-Accel-Buffering", "no")
	h.Del(context.ContentLengthHeaderKey)

	if r := s.ctx.Request(); r != nil && r.ProtoMajor == 1 {
		h.Set("Connection", "keep-alive")
	}

	s.ctx.StatusCode(http.StatusOK)
	s.w = w
	s.committed.Store(true)
	s.startHeartbeatLocked()

	return nil
}

// writeLocked writes p and flushes it, translating the failure modes.
func (s *Stream) writeLocked(p []byte) error {
	w := s.w

	if s.opts.SendTimeout > 0 {
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Now().Add(s.opts.SendTimeout)); err == nil {
			defer rc.SetWriteDeadline(time.Time{})
		}
	}

	if _, err := w.Write(p); err != nil {
		return s.failLocked(err)
	}

	if err := context.FlushResponseWriter(w); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return s.failLocked(err)
	}

	s.lastWrite = time.Now()
	return nil
}

// failLocked records a terminal write failure and cancels the stream context.
func (s *Stream) failLocked(err error) error {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		// The client is not draining the connection. Failing is the only correct
		// answer: skipping an event would corrupt the message the client is
		// assembling, with no way for it to notice.
		err = ErrSlowClient
	}

	s.closed = true
	s.err = err
	s.cancel(err)
	return err
}

// startHeartbeatLocked launches the keep-alive once the stream is committed.
func (s *Stream) startHeartbeatLocked() {
	if s.opts.Heartbeat <= 0 || s.beating {
		return
	}

	s.beating = true
	s.beatDone = make(chan struct{})
	s.lastWrite = time.Now()

	interval := s.opts.Heartbeat
	done := s.stdCtx.Done()

	go func() {
		defer close(s.beatDone)

		// A quarter of the interval keeps the check cheap while still bounding
		// idle time close to what was asked for.
		ticker := time.NewTicker(tickInterval(interval))
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				return
			case <-done:
				return
			case <-ticker.C:
				s.mu.Lock()
				if !s.closed && time.Since(s.lastWrite) >= interval {
					_ = s.writeLocked(s.opts.HeartbeatFrame)
				}
				s.mu.Unlock()
			}
		}
	}()
}

// Committed reports whether any byte has reached the client.
//
// It is the line between "this is still an error response" and "this is a
// truncated answer", and it is what a failover ladder must consult before
// retrying an upstream: after the first delta has gone out, a second attempt
// splices two different answers into one response.
func (s *Stream) Committed() bool { return s.committed.Load() }

// LastEventID reports the resume cursor the client sent, from the Last-Event-ID
// header or the lastEventId query parameter. It is empty on a fresh connection
// and can never contain CR, LF or NUL.
func (s *Stream) LastEventID() string { return s.lastEventID }

// Context is done when the client disconnects, when a write fails, or when the
// stream is closed. Select on it together with Context.ShuttingDown so a
// graceful shutdown ends the stream cleanly rather than waiting for its deadline.
func (s *Stream) Context() stdcontext.Context { return s.stdCtx }

// Err reports why the stream ended: the error Close was given or the write
// failure that ended it, nil after a clean Close, and nil while the stream is
// healthy. Before Close, a client that went away shows up here as the request
// context's cause.
func (s *Stream) Err() error {
	s.mu.Lock()
	closed, err := s.closed, s.err
	s.mu.Unlock()

	if closed {
		return err
	}

	return stdcontext.Cause(s.stdCtx)
}

// Close writes the terminal frame and releases the stream. It is idempotent, so
// a deferred Close and an explicit one coexist, and a panicking handler still
// terminates the stream cleanly.
//
// The terminal frame is written on success too, and that is the point. If the
// terminator is always present, its absence is how a client detects truncation.
// Nothing below it provides that signal: chunked framing catches a torn
// connection, but an nginx or load-balancer idle timeout closes the stream
// cleanly and the client cannot tell a finished answer from a severed one.
//
// When err is non-nil and nothing has been committed, Close writes nothing and
// returns err, so the caller renders an ordinary HTTP error. After the first
// Send it flushes any text an attached Coalescer still holds, writes the error
// event Options.ErrorFrame renders, and then the terminator. This package has
// no view on which errors are safe to show a client: the default frame runs
// the error through rest/errors, which redacts what was not deliberately
// mapped; use ai.Open, or map the error yourself before passing it here.
func (s *Stream) Close(err error) error {
	s.mu.Lock()

	if s.closed || s.closing {
		// Already ended, or another Close is in flight and owns the ending.
		s.mu.Unlock()
		return nil
	}
	s.closing = true
	defer close(s.closeDone)

	if !s.committed.Load() && err == nil && s.coalescer != nil {
		// Text the handler sent and the coalescer still holds is the answer, or
		// the start of it, and it commits the stream like any Send would.
		if flushErr := s.coalescer.flushLocked(); flushErr != nil {
			// The write failed and ended the stream on its own.
			s.mu.Unlock()
			return flushErr
		}
	}

	if !s.committed.Load() {
		// Nothing went out, so this is still an ordinary response. Anything a
		// coalescer holds is dropped with it: the caller is about to render an
		// error, not an answer.
		s.closed = true
		s.err = err
		done := s.beatDone
		s.mu.Unlock()
		s.stopHeartbeat(done)
		s.cancel(err)
		return err
	}

	s.mu.Unlock()

	// The frame is rendered outside the lock: the renderer is user code, and
	// the default one runs the error through the logger and the collector,
	// neither of which belongs under a lock the heartbeat also takes. A
	// renderer may even write to the stream itself.
	var (
		frame      Event
		writeFrame bool
	)
	if err != nil {
		frame, writeFrame = s.opts.ErrorFrame(s.ctx, err)
		if writeFrame && frame.validate() != nil {
			// A renderer that produced a forged frame gets the same answer as a
			// Send would. The terminator still goes out.
			writeFrame = false
		}
	}

	s.mu.Lock()

	if s.closed {
		// A write failure, a concurrent Close or the end of the request got
		// here first. The caller is still owed the error it passed in.
		s.mu.Unlock()
		return err
	}

	var writeErr error
	if s.coalescer != nil {
		// Text the handler sent and the coalescer still holds is part of the
		// answer. It goes out before anything that says the answer is over.
		writeErr = s.coalescer.flushLocked()
	}

	if writeFrame && writeErr == nil {
		writeErr = s.sendLocked(frame)
	}

	if s.opts.Terminator != "" && writeErr == nil {
		// Data must be non-nil: an event whose data buffer stays empty is never
		// dispatched by a conforming client, so a terminator without it would be
		// invisible.
		writeErr = s.sendLocked(Event{Name: s.opts.Terminator, Data: []byte{}})
	}

	s.closed = true
	s.err = cmpErr(err, writeErr)
	done := s.beatDone
	s.mu.Unlock()

	s.stopHeartbeat(done)
	s.cancel(cmpErr(err, writeErr))

	if err != nil {
		return err
	}

	return writeErr
}

// release ends a stream the handler forgot to close. It runs from
// Context.OnRelease, after the handler chain has returned and before the
// response writer is recycled: the heartbeat and any attached Coalescer's
// ticker are stopped and waited for, and the stream is marked closed with
// ErrClosed as its Err, so a late Send reports ErrClosed rather than writing
// into another request's response and Err tells an abandoned stream from a
// cleanly closed one.
//
// It writes nothing, not even the terminator. A stream that was never closed
// was truncated as far as the client can tell, and the missing terminator is
// exactly the signal that says so. A Close in flight on another goroutine is a
// different matter: that one is waited for, so its frames reach the wire.
func (s *Stream) release() {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		<-s.closeDone
		return
	}
	if s.closed {
		s.mu.Unlock()
		return
	}

	s.closed = true
	s.err = ErrClosed
	done := s.beatDone
	coalescer := s.coalescer
	s.mu.Unlock()

	s.stopHeartbeat(done)
	if coalescer != nil {
		coalescer.stopDrain()
	}
	s.cancel(ErrClosed)
}

// stopHeartbeat signals the keep-alive goroutine and waits for it, so no write
// can land after the handler has returned and the response writer has gone back
// to its pool.
func (s *Stream) stopHeartbeat(done <-chan struct{}) {
	s.stopOnce.Do(func() { close(s.stop) })

	if done != nil {
		<-done
	}
}

func cmpErr(first, second error) error {
	if first != nil {
		return first
	}

	return second
}
