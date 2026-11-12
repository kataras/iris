package sse

import (
	"slices"
	"sync"
	"time"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
)

// Coalesce batches the named events and emits at most one frame per name per
// interval, concatenating their payloads.
//
// It exists because a fast model is a syscall amplifier. One frame per token
// means one write, one flush, one TCP segment and one client-side render per
// token, and the tokens arrive faster than any of those are worth doing.
//
// The payloads are concatenated, so this is for text-like deltas - the running
// prose of an answer, a reasoning trace - and not for events whose data is a
// self-contained JSON document. Anything not named here passes straight through,
// after the pending buffers are flushed, so ordering is preserved.
//
// A background ticker drains the buffers, which is what makes the interval a
// ceiling rather than a floor. Without one, a buffered delta is only emitted
// when the next one arrives, so text that already reached the server sits unsent
// through a mid-answer pause - and a pause is exactly when a reader is most
// likely to be waiting.
//
// The returned Coalescer owns the stream until it is closed. Calling
// Stream.Send directly still works and flushes the pending buffers first, which
// is why the ordering hazard a hand-rolled batcher has to document - close the
// batcher before writing to the same writer, or the two interleave - cannot be
// expressed here.
func (s *Stream) Coalesce(interval time.Duration, names ...string) *Coalescer {
	c := &Coalescer{
		s:        s,
		interval: interval,
		names:    slices.Clone(names),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}

	s.mu.Lock()
	s.coalescer = c
	c.lastFlush = time.Now()
	s.mu.Unlock()

	if interval > 0 {
		go c.drain()
	} else {
		close(c.done)
	}

	return c
}

// Coalescer batches named events on a stream. See Stream.Coalesce.
type Coalescer struct {
	s        *Stream
	interval time.Duration
	names    []string

	// pending holds one buffer per coalesced name, and order records the order
	// in which they first appeared, so a flush emits them as the handler
	// produced them rather than as a map ranges.
	pending   map[string][]byte
	order     []string
	lastFlush time.Time

	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
}

// Send buffers ev when its name is coalesced, and sends it immediately - after
// flushing whatever is pending - when it is not.
func (c *Coalescer) Send(ev Event) error {
	if err := ev.validate(); err != nil {
		return err
	}

	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if !c.tracks(ev.Name) {
		if err := c.flushLocked(); err != nil {
			return err
		}

		return c.s.sendLocked(ev)
	}

	if c.pending == nil {
		c.pending = make(map[string][]byte, len(c.names))
	}
	if _, seen := c.pending[ev.Name]; !seen {
		c.order = append(c.order, ev.Name)
	}
	c.pending[ev.Name] = append(c.pending[ev.Name], ev.Data...)

	return nil
}

// SendJSON marshals v and sends it as a named event. A coalesced name would
// concatenate two JSON documents into nonsense, so this always flushes first and
// sends immediately.
func (c *Coalescer) SendJSON(name string, v any) error {
	buf, err := marshalJSONTo(v)
	if err != nil {
		return err
	}
	defer jsonutil.ReleaseBuffer(buf) // sendLocked copies the bytes synchronously.

	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if err := c.flushLocked(); err != nil {
		return err
	}

	return c.s.sendLocked(Event{Name: name, Data: buf.Bytes()})
}

// Flush emits everything buffered right now.
func (c *Coalescer) Flush() error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	return c.flushLocked()
}

// Close flushes what is pending, stops the ticker and hands the stream back.
// It is idempotent.
func (c *Coalescer) Close() error {
	c.stopDrain()

	c.s.mu.Lock()
	err := c.flushLocked()
	if c.s.coalescer == c {
		c.s.coalescer = nil
	}
	c.s.mu.Unlock()

	return err
}

// stopDrain stops the ticker goroutine and waits for it. It takes no lock:
// the goroutine needs the stream's to exit a tick, so the caller must not
// hold it.
func (c *Coalescer) stopDrain() {
	c.stopOnce.Do(func() { close(c.stop) })
	<-c.done
}

func (c *Coalescer) tracks(name string) bool {
	return slices.Contains(c.names, name)
}

// flushLocked emits the pending buffers in the order they first appeared.
func (c *Coalescer) flushLocked() error {
	if len(c.order) == 0 {
		return nil
	}

	for _, name := range c.order {
		data := c.pending[name]
		delete(c.pending, name)

		if len(data) == 0 {
			continue
		}

		if err := c.s.sendLocked(Event{Name: name, Data: data}); err != nil {
			c.order = c.order[:0]
			return err
		}
	}

	c.order = c.order[:0]
	c.lastFlush = time.Now()

	return nil
}

// drain makes the interval a ceiling on how long a buffered delta can wait.
func (c *Coalescer) drain() {
	defer close(c.done)

	// A quarter of the interval bounds the lateness of a flush to interval plus
	// a quarter, without waking up on every token.
	ticker := time.NewTicker(tickInterval(c.interval))
	defer ticker.Stop()

	streamDone := c.s.stdCtx.Done()

	for {
		select {
		case <-c.stop:
			return
		case <-streamDone:
			return
		case <-ticker.C:
			c.s.mu.Lock()
			if !c.s.closed && time.Since(c.lastFlush) >= c.interval {
				_ = c.flushLocked()
			}
			c.s.mu.Unlock()
		}
	}
}
