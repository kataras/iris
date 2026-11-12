package sse

import (
	"bufio"
	"bytes"
	"io"
	"iter"
	"slices"
	"strconv"
	"time"
)

// DefaultMaxEventBytes caps one event when ReadOptions does not say otherwise.
const DefaultMaxEventBytes = 1 << 20

// ReadOptions configure Read.
type ReadOptions struct {
	// Terminators end the iteration when an event's data equals one of them
	// exactly. Set []string{"[DONE]"} for an OpenAI-dialect upstream.
	//
	// Empty by default. An in-band sentinel is a compatibility concession on the
	// reading side and never a mechanism this package emits: it travels on the
	// same channel as the payload, so a producer that legitimately emits the
	// literal string ends the stream.
	Terminators []string

	// RequireTerminator ends the iteration with ErrNoTerminator when the stream
	// ends cleanly without one of the Terminators. It has no effect when
	// Terminators is empty.
	//
	// An upstream that promised a sentinel and closed without it was cut - by a
	// proxy idle timeout, by a crashed worker - however clean the close looked to
	// the transport. Off by default because a plain event stream has no
	// sentinel to miss; Relay turns it on whenever Terminators are configured.
	RequireTerminator bool

	// MaxEventBytes caps one event. Default DefaultMaxEventBytes. Exceeding it
	// ends the iteration with ErrTooLarge.
	MaxEventBytes int

	// KeepComments yields comment-only frames, such as an upstream's own
	// heartbeat, instead of skipping them. Off by default, because a conforming
	// client is required to ignore them.
	KeepComments bool
}

// Read parses an event stream and yields its events.
//
// It implements the WHATWG parsing algorithm rather than a prefix match, which
// matters more than it sounds: "data:" with and without a space are both legal,
// and a reader that matches "data: " sees an empty stream from a producer that
// omits it. It also handles multi-line data joined with "\n", comments, unknown
// fields, a leading byte order mark, and lines terminated by CR, LF or CRLF.
//
// It never yields a partially framed event. An event is dispatched by the blank
// line that ends it, so a stream cut mid-event ends the iteration with
// io.ErrUnexpectedEOF instead of yielding half an event, or worse, ending
// quietly as if the answer were complete. The terminal error, if any, comes
// last and io.EOF is not one: a clean end of stream simply ends the iteration,
// unless RequireTerminator says otherwise.
//
// Lenient on parse, strict on emit. A closed parser rejects a valid-but-newer
// producer, and an event stream is exactly the kind of format producers extend.
//
//	resp, err := client.Do(req)
//	// ...
//	defer resp.Body.Close()
//
//	for ev, err := range sse.Read(resp.Body, sse.ReadOptions{Terminators: []string{"[DONE]"}}) {
//		if err != nil {
//			return err
//		}
//		// ev.Name, ev.Data
//	}
func Read(r io.Reader, opts ...ReadOptions) iter.Seq2[Event, error] {
	o := ReadOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.MaxEventBytes <= 0 {
		o.MaxEventBytes = DefaultMaxEventBytes
	}

	return func(yield func(Event, error) bool) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 4096), o.MaxEventBytes)
		sc.Split(scanEventLines)

		var (
			name    string
			id      string
			retry   time.Duration
			data    []byte
			hasData bool
			size    int
			atStart = true
		)

		reset := func() {
			name, data, hasData, retry, size = "", data[:0], false, 0, 0
			// The last event ID persists across events by design: it is the
			// resume cursor, not a property of one frame.
		}

		for sc.Scan() {
			line := sc.Bytes()

			if atStart {
				line = bytes.TrimPrefix(line, []byte("\xef\xbb\xbf"))
				atStart = false
			}

			// A blank line dispatches whatever has been collected.
			if len(line) == 0 {
				if !hasData {
					// No data field means no event, per the algorithm. Anything
					// collected so far is discarded.
					reset()
					continue
				}

				// The trailing newline the algorithm appends is not part of the
				// payload.
				payload := bytes.TrimSuffix(data, []byte("\n"))

				if slices.Contains(o.Terminators, string(payload)) {
					return
				}

				ev := Event{
					ID:    id,
					Name:  name,
					Data:  slices.Clone(payload),
					Retry: retry,
				}
				reset()

				if !yield(ev, nil) {
					return
				}

				continue
			}

			if line[0] == ':' {
				if o.KeepComments {
					comment := string(bytes.TrimPrefix(line[1:], []byte(" ")))
					if !yield(Event{Comment: comment}, nil) {
						return
					}
				}

				continue
			}

			field, value, found := bytes.Cut(line, []byte(":"))
			if !found {
				// A line with no colon is a field with an empty value.
				field, value = line, nil
			} else {
				// Exactly one leading space is part of the framing, not the value.
				value = bytes.TrimPrefix(value, []byte(" "))
			}

			switch string(field) {
			case "event":
				name = string(value)
			case "data":
				size += len(value) + 1
				if size > o.MaxEventBytes {
					yield(Event{}, ErrTooLarge)
					return
				}

				data = append(data, value...)
				data = append(data, '\n')
				hasData = true
			case "id":
				// The algorithm ignores an id containing NUL and keeps the
				// previous one.
				if !bytes.ContainsRune(value, 0) {
					id = string(value)
				}
			case "retry":
				if ms, err := strconv.Atoi(string(value)); err == nil && ms >= 0 {
					retry = time.Duration(ms) * time.Millisecond
				}
			default:
				// Unknown field: ignored, which is what lets a producer add one
				// without breaking this reader.
			}
		}

		if err := sc.Err(); err != nil {
			if err == bufio.ErrTooLong {
				err = ErrTooLarge
			}

			yield(Event{}, err)
			return
		}

		// A clean end of stream. Data with no dispatching blank line behind it
		// is an event the producer never finished, and a producer that
		// promised a sentinel and did not send it did not finish either.
		switch {
		case hasData:
			yield(Event{}, io.ErrUnexpectedEOF)
		case o.RequireTerminator && len(o.Terminators) > 0:
			yield(Event{}, ErrNoTerminator)
		}
	}
}

// scanEventLines splits on CR, LF and CRLF, which are all line terminators in
// this format. bufio.ScanLines handles only LF and CRLF, so a producer that
// ends lines with a bare CR would look like one enormous line.
func scanEventLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i := range data {
		switch data[i] {
		case '\n':
			return i + 1, data[:i], nil
		case '\r':
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + 2, data[:i], nil
				}

				return i + 1, data[:i], nil
			}

			if atEOF {
				return i + 1, data[:i], nil
			}

			// One more byte decides between CR and CRLF; ask for it.
			return 0, nil, nil
		}
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}
