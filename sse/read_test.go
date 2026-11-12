package sse_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14/sse"
)

// read drains a stream and returns the events plus the terminal error.
func read(t *testing.T, raw string, opts ...sse.ReadOptions) ([]sse.Event, error) {
	t.Helper()

	var (
		events []sse.Event
		last   error
	)

	for ev, err := range sse.Read(strings.NewReader(raw), opts...) {
		if err != nil {
			last = err
			break
		}
		events = append(events, ev)
	}

	return events, last
}

// TestReadAcceptsBothDataSpellings is the whole reason this reader exists. Both
// forms are legal, and a parser that matches "data: " sees an empty stream from
// a producer that omits the space - which several OpenAI-compatible gateways do.
func TestReadAcceptsBothDataSpellings(t *testing.T) {
	events, err := read(t, "data: with\n\ndata:without\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(events), events)
	}
	if string(events[0].Data) != "with" {
		t.Errorf("event 0 data = %q, want %q", events[0].Data, "with")
	}
	if string(events[1].Data) != "without" {
		t.Errorf("event 1 data = %q, want %q", events[1].Data, "without")
	}
}

// TestReadOnlyStripsOneSpace: a second space is payload, not framing.
func TestReadOnlyStripsOneSpace(t *testing.T) {
	events, err := read(t, "data:  indented\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 || string(events[0].Data) != " indented" {
		t.Fatalf("events = %+v, want a single leading space preserved", events)
	}
}

func TestReadJoinsMultiLineData(t *testing.T) {
	events, err := read(t, "data: one\ndata: two\ndata: three\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if got, want := string(events[0].Data), "one\ntwo\nthree"; got != want {
		t.Errorf("data = %q, want %q", got, want)
	}
}

func TestReadLineTerminators(t *testing.T) {
	for name, raw := range map[string]string{
		"LF":   "data: a\n\ndata: b\n\n",
		"CRLF": "data: a\r\n\r\ndata: b\r\n\r\n",
		"CR":   "data: a\r\rdata: b\r\r",
	} {
		t.Run(name, func(t *testing.T) {
			events, err := read(t, raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 2 {
				t.Fatalf("got %d events, want 2: %+v", len(events), events)
			}
			if string(events[0].Data) != "a" || string(events[1].Data) != "b" {
				t.Errorf("events = %q, %q", events[0].Data, events[1].Data)
			}
		})
	}
}

func TestReadStripsByteOrderMark(t *testing.T) {
	events, err := read(t, "\xef\xbb\xbfdata: hello\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 || string(events[0].Data) != "hello" {
		t.Fatalf("events = %+v, want the BOM stripped", events)
	}
}

func TestReadComments(t *testing.T) {
	raw := ": ping\ndata: real\n\n"

	t.Run("skipped by default", func(t *testing.T) {
		events, err := read(t, raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 || string(events[0].Data) != "real" {
			t.Fatalf("events = %+v, want only the real event", events)
		}
	})

	t.Run("kept on request", func(t *testing.T) {
		events, err := read(t, raw, sse.ReadOptions{KeepComments: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 2 {
			t.Fatalf("got %d events, want 2: %+v", len(events), events)
		}
		if events[0].Comment != "ping" {
			t.Errorf("comment = %q, want %q", events[0].Comment, "ping")
		}
	})
}

func TestReadIgnoresUnknownFieldsAndKeepsID(t *testing.T) {
	// A producer adding a field must not break this reader, and the last event
	// ID persists across events because it is a cursor, not a frame property.
	events, err := read(t, "id: 7\nsequence_number: 3\nevent: token\ndata: a\n\ndata: b\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].ID != "7" || events[0].Name != "token" {
		t.Errorf("event 0 = %+v", events[0])
	}
	if events[1].ID != "7" {
		t.Errorf("event 1 id = %q, want the cursor to persist", events[1].ID)
	}
	if events[1].Name != "" {
		t.Errorf("event 1 name = %q, want it reset between events", events[1].Name)
	}
}

func TestReadRetry(t *testing.T) {
	events, err := read(t, "retry: 2500\ndata: a\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 || events[0].Retry != 2500*time.Millisecond {
		t.Fatalf("events = %+v, want a 2.5s retry", events)
	}
}

// TestReadNoDataFieldDispatchesNothing follows the algorithm: an event with no
// data field is not dispatched, and what it collected is discarded.
func TestReadNoDataFieldDispatchesNothing(t *testing.T) {
	events, err := read(t, "event: token\n\ndata: real\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 || string(events[0].Data) != "real" {
		t.Fatalf("events = %+v, want only the event that carried data", events)
	}
}

// TestReadNeverYieldsAPartialEvent: a stream cut before the blank line has not
// produced an event, and reporting one would hand the caller half an answer.
// Ending quietly is no better: the caller would take three deltas for a whole
// answer, so the cut is reported as io.ErrUnexpectedEOF.
func TestReadNeverYieldsAPartialEvent(t *testing.T) {
	events, err := read(t, "data: complete\n\ndata: cut off")
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want io.ErrUnexpectedEOF for a stream cut mid-event", err)
	}

	if len(events) != 1 || string(events[0].Data) != "complete" {
		t.Fatalf("events = %+v, want only the framed event", events)
	}
}

// TestReadRequireTerminator: an upstream that promised a sentinel and closed
// without it was cut, whatever the transport says.
func TestReadRequireTerminator(t *testing.T) {
	opts := sse.ReadOptions{Terminators: []string{"[DONE]"}, RequireTerminator: true}

	t.Run("missing", func(t *testing.T) {
		events, err := read(t, "data: a\n\ndata: b\n\n", opts)
		if !errors.Is(err, sse.ErrNoTerminator) {
			t.Fatalf("error = %v, want ErrNoTerminator", err)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Error("ErrNoTerminator does not match io.ErrUnexpectedEOF, so a caller checking for a cut stream misses it")
		}
		if len(events) != 2 {
			t.Fatalf("got %d events, want both events before the error", len(events))
		}
	})

	t.Run("present", func(t *testing.T) {
		events, err := read(t, "data: a\n\ndata: [DONE]\n\n", opts)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("got %d events, want 1", len(events))
		}
	})

	t.Run("without terminators configured it is not required", func(t *testing.T) {
		if _, err := read(t, "data: a\n\n", sse.ReadOptions{RequireTerminator: true}); err != nil {
			t.Fatalf("error = %v, want nil: there is no terminator to require", err)
		}
	})
}

func TestReadTerminators(t *testing.T) {
	events, err := read(t,
		"data: a\n\ndata: [DONE]\n\ndata: never\n\n",
		sse.ReadOptions{Terminators: []string{"[DONE]"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 || string(events[0].Data) != "a" {
		t.Fatalf("events = %+v, want iteration to stop at the sentinel", events)
	}
}

func TestReadMaxEventBytes(t *testing.T) {
	raw := "data: " + strings.Repeat("x", 200) + "\n\n"

	_, err := read(t, raw, sse.ReadOptions{MaxEventBytes: 64})
	if !errors.Is(err, sse.ErrTooLarge) {
		t.Fatalf("error = %v, want ErrTooLarge", err)
	}
}

func TestReadEmptyStream(t *testing.T) {
	events, err := read(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %+v, want none", events)
	}
}

// TestReadBreakStopsEarly checks the iterator contract: abandoning the range
// must not panic or leak.
func TestReadBreakStopsEarly(t *testing.T) {
	var seen int

	for ev, err := range sse.Read(strings.NewReader("data: a\n\ndata: b\n\ndata: c\n\n")) {
		if err != nil {
			t.Fatal(err)
		}
		_ = ev
		seen++
		break
	}

	if seen != 1 {
		t.Fatalf("saw %d events, want 1", seen)
	}
}

// TestReadSurfacesReaderErrors: the terminal error comes last, and io.EOF is not
// one of them.
func TestReadSurfacesReaderErrors(t *testing.T) {
	sentinel := errors.New("connection reset")

	r := io.MultiReader(strings.NewReader("data: a\n\n"), &failingReader{err: sentinel})

	var (
		events []sse.Event
		got    error
	)
	for ev, err := range sse.Read(r) {
		if err != nil {
			got = err
			break
		}
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want the one complete event first", len(events))
	}
	if !errors.Is(got, sentinel) {
		t.Fatalf("terminal error = %v, want %v", got, sentinel)
	}
}

type failingReader struct{ err error }

func (r *failingReader) Read([]byte) (int, error) { return 0, r.err }

// TestReadSizeLimitIsPerEvent: MaxEventBytes caps one event, not the whole
// stream. A reader that never reset its counter killed every long stream at 1
// MiB of cumulative data, which a chat answer crosses in a few minutes.
func TestReadSizeLimitIsPerEvent(t *testing.T) {
	const eventSize = 600 << 10 // three of these cross DefaultMaxEventBytes together.
	big := strings.Repeat("x", eventSize)
	raw := strings.Repeat("data: "+big+"\n\n", 3)

	events, err := read(t, raw)
	if err != nil {
		t.Fatalf("error = %v, want three events under the default limit", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	for i, ev := range events {
		if len(ev.Data) != eventSize {
			t.Errorf("event %d has %d bytes, want %d", i, len(ev.Data), eventSize)
		}
	}

	// One event of MaxEventBytes+1, split over two data lines so the check that
	// fires is the per-event counter and not the scanner's line limit.
	half := sse.DefaultMaxEventBytes / 2
	raw = "data: " + strings.Repeat("a", half) + "\ndata: " + strings.Repeat("b", half) + "\n\n"

	_, err = read(t, raw)
	if !errors.Is(err, sse.ErrTooLarge) {
		t.Fatalf("error = %v, want ErrTooLarge for one event over the limit", err)
	}
}
