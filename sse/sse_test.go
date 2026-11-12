package sse_test

import (
	"bufio"
	stdcontext "context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	irishttptest "github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

// serve builds app and puts it behind a real listener. A real socket is not
// optional here: most of what these tests assert is whether bytes reach the
// client before the handler returns, and an in-process recorder cannot see that.
func serve(t *testing.T, app *iris.Application) *httptest.Server {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)

	return srv
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

// collect drains an event stream into a slice.
func collect(t *testing.T, r io.Reader, opts ...sse.ReadOptions) []sse.Event {
	t.Helper()

	var events []sse.Event
	for ev, err := range sse.Read(r, opts...) {
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		events = append(events, ev)
	}

	return events
}

// --------------------------------------------------------------------------
// Open writes nothing
// --------------------------------------------------------------------------

// TestOpenWritesNothing is the property the whole design rests on. Every
// complaint about a framework's SSE support traces back to opening the stream
// too early: once the status line is out, a handler can no longer refuse.
func TestOpenWritesNothing(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if _, err := sse.Open(ctx); err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		// Deliberately send nothing and return.
	})

	resp := get(t, serve(t, app).URL)

	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want the stream headers to be absent until the first Send", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}
}

// TestErrorAfterOpenIsAnOrdinaryResponse is the same property from the angle
// that matters in production: an authorization or policy check that runs after
// Open must still be able to answer with a real status code and a real body.
func TestErrorAfterOpenIsAnOrdinaryResponse(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, err := sse.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		defer stream.Close(nil)

		// The check a real handler would do between opening and streaming.
		ctx.StopWithJSON(http.StatusUnauthorized, iris.Map{"error": "unauthorized"})
	})

	resp := get(t, serve(t, app).URL)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unauthorized") {
		t.Errorf("body = %q, want the JSON error", body)
	}
}

// --------------------------------------------------------------------------
// The wire format
// --------------------------------------------------------------------------

func TestSendCommitsTheStreamHeaders(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, err := sse.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		defer stream.Close(nil)

		if stream.Committed() {
			t.Error("Committed reported true before the first Send")
		}
		if err := stream.Send(sse.Event{Name: "token", Data: []byte("hi")}); err != nil {
			t.Errorf("Send: %v", err)
		}
		if !stream.Committed() {
			t.Error("Committed reported false after a Send")
		}
	})

	resp := get(t, serve(t, app).URL)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	for header, want := range map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache, no-transform",
		"X-Accel-Buffering": "no",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	if resp.Header.Get("Content-Length") != "" {
		t.Error("Content-Length is set on a stream")
	}
}

// TestTerminalFrameOnSuccess guards the signal that lets a client tell a
// finished answer from a severed one. An idle-timeout kill by a proxy closes the
// connection cleanly, so the absence of a terminator is the only evidence.
func TestTerminalFrameOnSuccess(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)

		stream.Send(sse.Event{Name: "token", Data: []byte("a")})
		stream.Send(sse.Event{Name: "token", Data: []byte("b")})
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (two tokens and the terminator): %+v", len(events), events)
	}
	if events[2].Name != "done" {
		t.Errorf("last event name = %q, want %q", events[2].Name, "done")
	}

	// AutoID is on by default, and it is what makes truncation detectable.
	for i, ev := range events {
		if ev.ID == "" {
			t.Errorf("event %d has no id", i)
		}
	}
}

func TestCloseWithErrorAfterCommitWritesAnErrorFrame(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Name: "token", Data: []byte("partial")})
		stream.Close(stdErrors.New("upstream exploded"))
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(events), events)
	}
	if events[1].Name != "error" {
		t.Errorf("event 1 name = %q, want %q", events[1].Name, "error")
	}
	if events[2].Name != "done" {
		t.Errorf("event 2 name = %q, want the terminator", events[2].Name)
	}

	// The frame carries the rest/errors envelope, so a client parses the same
	// shape whether the failure happened before the first byte or after it.
	var reported errors.Error
	if err := json.Unmarshal(events[1].Data, &reported); err != nil {
		t.Fatalf("error frame is not an errors.Error: %v (%q)", err, events[1].Data)
	}

	if reported.ID == "" {
		t.Error("the error frame carries no correlation id")
	}
	if reported.ErrorCode.CanonicalName != "INTERNAL" {
		t.Errorf("canonical name = %q, want INTERNAL for an unmapped error", reported.ErrorCode.CanonicalName)
	}

	// The point of the whole exercise: an unmapped error is redacted, so an
	// upstream message naming a vendor, a model or a host cannot leak into a
	// stream by default.
	if strings.Contains(string(events[1].Data), "upstream exploded") {
		t.Errorf("the raw error text reached the client: %q", events[1].Data)
	}
}

// TestErrorFrameReportsToTheLogger is the half that must not be skipped: a
// mid-stream failure is invisible unless it reaches the logger, which is the
// hook a cloud sink replaces.
func TestErrorFrameReportsToTheLogger(t *testing.T) {
	original := errors.LogError
	t.Cleanup(func() { errors.LogError = original })

	logged := make(chan error, 1)
	errors.LogError = func(_ iris.Context, err error) {
		select {
		case logged <- err:
		default:
		}
	}

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Name: "token", Data: []byte("partial")})
		stream.Close(stdErrors.New("anthropic is out of capacity"))
	})

	resp := get(t, serve(t, app).URL)
	io.Copy(io.Discard, resp.Body)

	select {
	case err := <-logged:
		// The logger gets the real error, before any redaction.
		if !strings.Contains(err.Error(), "anthropic is out of capacity") {
			t.Errorf("logged error = %v, want the unredacted original", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the mid-stream failure never reached the logger")
	}
}

// TestRawErrorFrameOptOut keeps the previous behaviour reachable for text that
// is already known to be safe.
func TestRawErrorFrameOptOut(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.ErrorFrame = sse.RawErrorFrame

		stream, _ := sse.Open(ctx, o)
		stream.Send(sse.Event{Name: "token", Data: []byte("partial")})
		stream.Close(stdErrors.New("safe to show"))
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) < 2 || string(events[1].Data) != "safe to show" {
		t.Fatalf("events = %+v, want the raw text", events)
	}
}

// TestMappedErrorKeepsItsMessage: an application that classified an error on
// purpose gets the message it chose, not a generic one.
func TestMappedErrorKeepsItsMessage(t *testing.T) {
	sentinel := stdErrors.New("the model is busy, try again shortly")

	app := iris.New()
	app.UseRouter(errors.NewOptions().MapErrors(errors.ResourceExhausted, sentinel).Handler())
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Name: "token", Data: []byte("partial")})
		stream.Close(sentinel)
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) < 2 {
		t.Fatalf("events = %+v", events)
	}

	var reported errors.Error
	if err := json.Unmarshal(events[1].Data, &reported); err != nil {
		t.Fatalf("error frame is not an errors.Error: %v", err)
	}

	if reported.ErrorCode.CanonicalName != "RESOURCE_EXHAUSTED" {
		t.Errorf("canonical name = %q, want the mapped code", reported.ErrorCode.CanonicalName)
	}
	if !strings.Contains(reported.Message, "the model is busy") {
		t.Errorf("message = %q, want the mapped error's own text", reported.Message)
	}
}

func TestCloseWithErrorBeforeCommitWritesNothing(t *testing.T) {
	sentinel := stdErrors.New("nope")

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		if err := stream.Close(sentinel); !stdErrors.Is(err, sentinel) {
			t.Errorf("Close returned %v, want the original error back", err)
		}
		ctx.StopWithStatus(http.StatusBadGateway)
	})

	resp := get(t, serve(t, app).URL)

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "event-stream") {
		t.Errorf("Content-Type = %q, want no stream headers", ct)
	}
}

func TestSendRejectsFramingCharacters(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)

		for _, ev := range []sse.Event{
			{ID: "a\nid: forged", Data: []byte("x")},
			{Name: "a\ndata: forged", Data: []byte("x")},
			{Comment: "a\r\ndata: forged", Data: []byte("x")},
		} {
			if err := stream.Send(ev); !stdErrors.Is(err, sse.ErrInvalidField) {
				t.Errorf("Send(%+v) error = %v, want ErrInvalidField", ev, err)
			}
		}
	})

	get(t, serve(t, app).URL)
}

func TestMultiLineDataRoundTrips(t *testing.T) {
	payload := "first\nsecond\nthird"

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)
		stream.Send(sse.Event{Name: "chunk", Data: []byte(payload)})
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) < 1 {
		t.Fatal("no events")
	}
	if string(events[0].Data) != payload {
		t.Errorf("data = %q, want %q", events[0].Data, payload)
	}
}

// --------------------------------------------------------------------------
// Streaming survives the middleware that would otherwise buffer it
// --------------------------------------------------------------------------

// TestStreamsUnderLogRequests is the regression test for the bug that the
// framework's own recommended chain de-streamed every SSE response: accesslog
// installs a response recorder, and a recorder buffers.
func TestStreamsUnderLogRequests(t *testing.T) {
	ac := accesslog.New(io.Discard)
	ac.ResponseBody = true // what the builder's LogRequests(true) turns on.
	t.Cleanup(func() { ac.Close() })

	release := make(chan struct{})

	app := iris.New()
	app.UseRouter(ac.Handler)
	app.Get("/", func(ctx iris.Context) {
		stream, err := sse.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		defer stream.Close(nil)

		stream.Send(sse.Event{Name: "token", Data: []byte("early")})
		<-release
		stream.Send(sse.Event{Name: "token", Data: []byte("late")})
	})

	resp := get(t, serve(t, app).URL)

	first := make(chan string, 1)
	go func() {
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(first)
				return
			}
			if strings.HasPrefix(line, "data:") {
				first <- line
				return
			}
		}
	}()

	select {
	case line, ok := <-first:
		if !ok {
			t.Fatal("the stream ended before any data arrived")
		}
		if !strings.Contains(line, "early") {
			t.Fatalf("first data line = %q", line)
		}
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("no event arrived while the handler was still running: something buffered the stream")
	}

	close(release)
}

// TestStreamsUnderCompression checks the other global middleware that holds
// bytes. The response must come back uncompressed rather than compressed-and-
// buffered.
func TestStreamsUnderCompression(t *testing.T) {
	release := make(chan struct{})

	app := iris.New()
	app.UseRouter(compress.Handler)
	app.Get("/", func(ctx iris.Context) {
		stream, err := sse.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		defer stream.Close(nil)

		stream.Send(sse.Event{Name: "token", Data: []byte("early")})
		<-release
		stream.Send(sse.Event{Name: "token", Data: []byte("late")})
	})

	srv := serve(t, app)

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	// Stop the transport from transparently decompressing, so the assertion is
	// about what the server actually sent.
	resp, err := (&http.Transport{DisableCompression: true}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if enc := resp.Header.Get("Content-Encoding"); enc != "" {
		t.Errorf("Content-Encoding = %q, want the stream to be uncompressed", enc)
	}

	first := make(chan string, 1)
	go func() {
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(first)
				return
			}
			if strings.HasPrefix(line, "data:") {
				first <- line
				return
			}
		}
	}()

	select {
	case line, ok := <-first:
		if !ok {
			t.Fatal("the stream ended before any data arrived")
		}
		if !strings.Contains(line, "early") {
			t.Fatalf("first data line = %q", line)
		}
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("no event arrived while the handler was still running: compression buffered the stream")
	}

	close(release)
}

// TestOpenRefusesWhenTheResponseCannotFlush covers Timeout.HandlerTimeout, under
// which http.TimeoutHandler's writer implements neither Flusher nor Hijacker.
// The point is that it is reported before a byte is written, so the handler can
// still answer honestly.
func TestOpenRefusesWhenTheResponseCannotFlush(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		_, err := sse.Open(ctx)
		if !stdErrors.Is(err, sse.ErrNotStreamable) {
			t.Errorf("Open error = %v, want ErrNotStreamable", err)
			return
		}

		ctx.StopWithJSON(http.StatusNotImplemented, iris.Map{"error": "streaming unavailable"})
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	// Wrap exactly as core/server does for Timeout.HandlerTimeout.
	srv := httptest.NewServer(http.TimeoutHandler(app, time.Minute, "timeout"))
	defer srv.Close()

	resp := get(t, srv.URL)

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", resp.StatusCode)
	}
}

// --------------------------------------------------------------------------
// Resume cursor
// --------------------------------------------------------------------------

func TestLastEventIDFromHeaderAndQuery(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)
		stream.Send(sse.Event{Name: "cursor", Data: []byte(stream.LastEventID())})
	})

	srv := serve(t, app)

	t.Run("header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		req.Header.Set("Last-Event-ID", "42")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		events := collect(t, resp.Body)
		if len(events) == 0 || string(events[0].Data) != "42" {
			t.Fatalf("events = %+v, want the cursor 42", events)
		}
	})

	// A browser EventSource can send the header but cannot POST; a fetch-based
	// client can do neither conveniently and puts the cursor in the URL.
	t.Run("query", func(t *testing.T) {
		resp := get(t, srv.URL+"?lastEventId=99")
		events := collect(t, resp.Body)
		if len(events) == 0 || string(events[0].Data) != "99" {
			t.Fatalf("events = %+v, want the cursor 99", events)
		}
	})

	t.Run("sanitised", func(t *testing.T) {
		resp := get(t, srv.URL+"?lastEventId=a%0Adata:+forged")
		events := collect(t, resp.Body)
		if len(events) == 0 || string(events[0].Data) != "" {
			t.Fatalf("events = %+v, want a rejected cursor", events)
		}
	})
}

// --------------------------------------------------------------------------
// Coalescing
// --------------------------------------------------------------------------

func TestCoalescerBatchesAndPreservesOrder(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)

		c := stream.Coalesce(50*time.Millisecond, "token")
		defer c.Close()

		for _, part := range []string{"Hel", "lo ", "wor", "ld"} {
			c.Send(sse.Event{Name: "token", Data: []byte(part)})
		}

		// An uncoalesced event flushes what is pending first, so ordering holds
		// without the caller having to think about it.
		c.Send(sse.Event{Name: "usage", Data: []byte("4")})
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (one batched token, usage, terminator): %+v", len(events), events)
	}
	if events[0].Name != "token" || string(events[0].Data) != "Hello world" {
		t.Errorf("event 0 = %q/%q, want token/\"Hello world\"", events[0].Name, events[0].Data)
	}
	if events[1].Name != "usage" {
		t.Errorf("event 1 name = %q, want usage", events[1].Name)
	}
}

// TestCoalescerTickerIsACeiling checks the property a hand-rolled batcher
// usually misses: a buffered delta must not wait for the next one to arrive.
func TestCoalescerTickerIsACeiling(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		defer stream.Close(nil)

		c := stream.Coalesce(40*time.Millisecond, "token")
		defer c.Close()

		c.Send(sse.Event{Name: "token", Data: []byte("alone")})
		// Nothing else is sent for a while. Without a draining ticker this event
		// would sit in the buffer until the handler returned.
		time.Sleep(400 * time.Millisecond)
	})

	resp := get(t, serve(t, app).URL)

	arrived := make(chan struct{})
	go func() {
		defer close(arrived)
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if strings.Contains(line, "alone") {
				return
			}
		}
	}()

	select {
	case <-arrived:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("the buffered event never drained: the interval is a floor, not a ceiling")
	}
}

// TestTimeoutErrorStillGetsAnErrorFrame: an upstream timeout looks like a
// cancellation to errors.Is, and used to be skipped as "nobody is waiting".
// The client is waiting; only its own disconnect means otherwise.
func TestTimeoutErrorStillGetsAnErrorFrame(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Name: "token", Data: []byte("partial")})
		stream.Close(fmt.Errorf("upstream: %w", stdcontext.DeadlineExceeded))
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) != 3 || events[1].Name != "error" || events[2].Name != "done" {
		t.Fatalf("events = %+v, want token, error, done", events)
	}
}

// --------------------------------------------------------------------------
// A handler that returns without Close
// --------------------------------------------------------------------------

// waitForGoroutines polls until the goroutine count is back at baseline, so a
// goroutine that exits a few milliseconds after the response does not fail the
// test, and one that never exits does.
func waitForGoroutines(t *testing.T, baseline int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("goroutines = %d, want at most the baseline of %d: a stream goroutine outlived its request", runtime.NumGoroutine(), baseline)
}

// TestForgottenCloseEndsTheStreamBeforeTheContextIsRecycled: the router
// releases the Context inside ServeHTTP, before net/http cancels the request
// context. A heartbeat still ticking in that window writes through a nil
// writer (a panic in a goroutine net/http cannot recover, so the process
// exits) or into a Context another request has already acquired. The router
// wrapper below holds that window open for long enough to be certain.
func TestForgottenCloseEndsTheStreamBeforeTheContextIsRecycled(t *testing.T) {
	var forgotten *sse.Stream

	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		next(w, r)
		if r.URL.Path == "/stream" {
			// The Context is back in the pool; the request context is not yet
			// cancelled. Ticks land here.
			time.Sleep(50 * time.Millisecond)
		}
	})
	app.Get("/stream", func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.Heartbeat = time.Millisecond

		stream, err := sse.Open(ctx, o)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		forgotten = stream

		stream.Send(sse.Event{Name: "token", Data: []byte("only")})
		// No Close. The handler simply returns.
	})
	app.Get("/plain", func(ctx iris.Context) { ctx.WriteString("plain") })

	srv := serve(t, app)

	// Warm the connection so the client and server keep-alive goroutines are
	// part of the baseline.
	io.Copy(io.Discard, get(t, srv.URL+"/plain").Body)
	baseline := runtime.NumGoroutine()

	body, _ := io.ReadAll(get(t, srv.URL+"/stream").Body)
	if strings.Contains(string(body), "ping") {
		t.Errorf("a heartbeat was written after the handler returned: %q", body)
	}
	if !strings.Contains(string(body), "only") {
		t.Errorf("body = %q, want the event the handler did send", body)
	}

	if err := forgotten.Send(sse.Event{Name: "late", Data: []byte("x")}); !stdErrors.Is(err, sse.ErrClosed) {
		t.Errorf("Send after the request ended = %v, want ErrClosed: the stream was not marked closed before the Context was recycled", err)
	}
	if err := forgotten.Err(); !stdErrors.Is(err, sse.ErrClosed) {
		t.Errorf("Err() of an abandoned stream = %v, want ErrClosed so it is distinguishable from a clean Close", err)
	}

	// The pooled Context is serving someone else now. Nothing of the stream may
	// reach them.
	resp := get(t, srv.URL+"/plain")
	plain, _ := io.ReadAll(resp.Body)
	if string(plain) != "plain" || strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		t.Errorf("the next request got %q with Content-Type %q", plain, resp.Header.Get("Content-Type"))
	}

	waitForGoroutines(t, baseline)
}

// --------------------------------------------------------------------------
// Close
// --------------------------------------------------------------------------

// TestCloseFlushesTheCoalescer: text still buffered when the stream closes is
// text the handler sent. Dropping it because the coalescer was closed second
// truncated the answer silently, terminator and all.
func TestCloseFlushesTheCoalescer(t *testing.T) {
	var coalescerCloseErr error

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		c := stream.Coalesce(time.Hour, "token") // never drains on its own.

		c.Send(sse.Event{Name: "token", Data: []byte("buffered")})
		stream.Close(nil)
		coalescerCloseErr = c.Close()
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	if len(events) != 2 || string(events[0].Data) != "buffered" || events[1].Name != "done" {
		t.Fatalf("events = %+v, want the buffered token and then the terminator", events)
	}
	if coalescerCloseErr != nil {
		t.Errorf("Coalescer.Close after Stream.Close = %v, want nil: there was nothing left to send", coalescerCloseErr)
	}
}

// TestErrIsNilAfterACleanClose: Close(nil) is not a failure, and Err must not
// dress it up as context.Canceled.
func TestErrIsNilAfterACleanClose(t *testing.T) {
	failure := stdErrors.New("boom")
	var afterClean, afterFailure error

	app := iris.New()
	app.Get("/clean", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Data: []byte("a")})
		stream.Close(nil)
		afterClean = stream.Err()
	})
	app.Get("/failed", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Send(sse.Event{Data: []byte("a")})
		stream.Close(failure)
		afterFailure = stream.Err()
	})

	srv := serve(t, app)
	io.Copy(io.Discard, get(t, srv.URL+"/clean").Body)
	io.Copy(io.Discard, get(t, srv.URL+"/failed").Body)

	if afterClean != nil {
		t.Errorf("Err() after Close(nil) = %v, want nil", afterClean)
	}
	if !stdErrors.Is(afterFailure, failure) {
		t.Errorf("Err() after Close(err) = %v, want the error", afterFailure)
	}
}

// TestOpenRejectsAForgedTerminator: the terminator is an event name on the
// wire, so a newline in it is a forged frame. Send checks its fields; the
// option was not checked anywhere.
func TestOpenRejectsAForgedTerminator(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.Terminator = "done\nevent: forged"

		_, err := sse.Open(ctx, o)
		if !stdErrors.Is(err, sse.ErrInvalidField) {
			t.Errorf("Open = %v, want ErrInvalidField", err)
		}
		ctx.StatusCode(http.StatusTeapot)
	})

	if resp := get(t, serve(t, app).URL); resp.StatusCode != http.StatusTeapot {
		t.Errorf("status = %d, want the handler's own", resp.StatusCode)
	}
}

// TestErrorFrameIsValidatedAndRunsUnlocked: the error frame is rendered by user
// code. Its output gets the same framing check as any Send, and it runs before
// the stream lock is taken, so a renderer that writes a diagnostic comment on
// the stream does not deadlock.
func TestErrorFrameIsValidatedAndRunsUnlocked(t *testing.T) {
	var stream *sse.Stream

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.ErrorFrame = func(_ iris.Context, err error) (sse.Event, bool) {
			stream.Comment("rendering the failure") // would deadlock under the lock.
			return sse.Event{Name: "error\nevent: forged", Data: []byte("x")}, true
		}

		stream, _ = sse.Open(ctx, o)
		stream.Send(sse.Event{Name: "token", Data: []byte("a")})
		stream.Close(stdErrors.New("boom"))
	})

	body, _ := io.ReadAll(get(t, serve(t, app).URL).Body)

	if strings.Contains(string(body), "forged") {
		t.Errorf("a forged frame reached the wire: %q", body)
	}
	if !strings.Contains(string(body), "rendering the failure") {
		t.Errorf("the renderer's own comment is missing: %q", body)
	}
	if !strings.Contains(string(body), "event: done") {
		t.Errorf("the terminator is missing after an invalid error frame: %q", body)
	}
}

// TestCommitWithoutAnEvent: Commit lets a stream that turns out to carry no
// events still end with its terminator, and the first real event still
// advertises the retry interval.
func TestCommitWithoutAnEvent(t *testing.T) {
	app := iris.New()
	app.Get("/empty", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		if err := stream.Commit(); err != nil {
			t.Errorf("Commit: %v", err)
		}
		stream.Close(nil)
	})
	app.Get("/one", func(ctx iris.Context) {
		stream, _ := sse.Open(ctx)
		stream.Commit()
		stream.Send(sse.Event{Name: "token", Data: []byte("a")})
		stream.Close(nil)
	})

	srv := serve(t, app)

	resp := get(t, srv.URL+"/empty")
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want the stream committed", ct)
	}
	if events := collect(t, resp.Body); len(events) != 1 || events[0].Name != "done" {
		t.Errorf("events = %+v, want only the terminator", events)
	}

	events := collect(t, get(t, srv.URL+"/one").Body)
	if len(events) != 2 || events[0].Retry != 3*time.Second {
		t.Errorf("events = %+v, want the first event to carry the retry interval", events)
	}
}

// TestForgottenCloseWithACoalescerAttached: the coalescer has its own ticker.
// Ending the stream at release must stop that one too, before the writer goes
// back to its pool, and must not flush what it holds.
func TestForgottenCloseWithACoalescerAttached(t *testing.T) {
	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		next(w, r)
		if r.URL.Path == "/stream" {
			time.Sleep(50 * time.Millisecond)
		}
	})
	app.Get("/stream", func(ctx iris.Context) {
		stream, err := sse.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}

		c := stream.Coalesce(time.Millisecond, "token")
		c.Send(sse.Event{Name: "token", Data: []byte("first")})
		c.Flush() // commits the stream, so the ticker has a writer to reach.
		c.Send(sse.Event{Name: "token", Data: []byte("pending")})
		// No Close on either.
	})
	app.Get("/plain", func(ctx iris.Context) { ctx.WriteString("plain") })

	srv := serve(t, app)
	io.Copy(io.Discard, get(t, srv.URL+"/plain").Body)
	baseline := runtime.NumGoroutine()

	body, _ := io.ReadAll(get(t, srv.URL+"/stream").Body)
	if !strings.Contains(string(body), "first") {
		t.Errorf("body = %q, want the flushed event", body)
	}

	plain, _ := io.ReadAll(get(t, srv.URL+"/plain").Body)
	if string(plain) != "plain" {
		t.Errorf("the next request got %q", plain)
	}

	waitForGoroutines(t, baseline)
}

// TestForgottenCloseUnderHttptestDo: httptest.Do hosts a handler without the
// router, and it releases the response writer itself. It must run the release
// hooks first, or a heartbeat tick after Do returns writes through the pooled
// writer whose underlying writer has just been set to nil.
func TestForgottenCloseUnderHttptestDo(t *testing.T) {
	rec := irishttptest.NewRecorder()
	req := irishttptest.NewRequest(http.MethodGet, "/", nil)

	irishttptest.Do(rec, req, func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.Heartbeat = time.Millisecond

		stream, err := sse.Open(ctx, o)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		stream.Send(sse.Event{Name: "token", Data: []byte("only")})
		// No Close.
	})

	// The tick lands here if the heartbeat is still running.
	time.Sleep(30 * time.Millisecond)

	if body := rec.Body.String(); !strings.Contains(body, "only") || strings.Contains(body, "ping") {
		t.Errorf("body = %q, want the event and no heartbeat after Do returned", body)
	}
}

// TestCloseReturnsItsErrorWhenTheStreamEndedMeanwhile: the error frame is
// rendered outside the lock, and the stream can end in that window. Close still
// owes its caller the error it was given.
func TestCloseReturnsItsErrorWhenTheStreamEndedMeanwhile(t *testing.T) {
	boom := stdErrors.New("boom")
	var got error

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		var stream *sse.Stream
		o := sse.DefaultOptions()
		o.ErrorFrame = func(_ iris.Context, err error) (sse.Event, bool) {
			stream.Close(nil) // the stream ends while the frame is being rendered.
			return sse.RawErrorFrame(nil, err)
		}

		stream, _ = sse.Open(ctx, o)
		stream.Send(sse.Event{Data: []byte("a")})
		got = stream.Close(boom)
	})

	io.Copy(io.Discard, get(t, serve(t, app).URL).Body)

	if !stdErrors.Is(got, boom) {
		t.Errorf("Close(boom) = %v, want boom", got)
	}
}

// TestReleaseWaitsForAnInFlightClose: a Close that is rendering its error frame
// on another goroutine when the handler returns must get to write the frame
// and the terminator before the response is recycled, rather than being told
// the stream is already closed.
func TestReleaseWaitsForAnInFlightClose(t *testing.T) {
	rendering := make(chan struct{})

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := sse.DefaultOptions()
		o.ErrorFrame = func(_ iris.Context, err error) (sse.Event, bool) {
			close(rendering)
			time.Sleep(50 * time.Millisecond)
			return sse.RawErrorFrame(nil, err)
		}

		stream, _ := sse.Open(ctx, o)
		stream.Send(sse.Event{Name: "token", Data: []byte("a")})

		go stream.Close(stdErrors.New("late failure"))
		<-rendering
		// The handler returns while Close is in its render window.
	})

	events := collect(t, get(t, serve(t, app).URL).Body)

	var names []string
	for _, ev := range events {
		names = append(names, ev.Name)
	}
	if len(names) != 3 || names[1] != "error" || names[2] != "done" {
		t.Errorf("events = %v, want [token error done]: the in-flight Close was cut off by the release", names)
	}
}
