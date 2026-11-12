package ai_test

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
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/ai"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

func serve(t *testing.T, app *iris.Application) *httptest.Server {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)

	return srv
}

// parts drains an event stream and decodes the neutral part objects.
func parts(t *testing.T, body io.Reader) []map[string]any {
	t.Helper()

	var out []map[string]any
	for ev, err := range sse.Read(body) {
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		if ev.Name == "done" {
			continue
		}

		var m map[string]any
		if err := json.Unmarshal(ev.Data, &m); err != nil {
			t.Fatalf("event %q data %q: %v", ev.Name, ev.Data, err)
		}
		out = append(out, m)
	}

	return out
}

// --------------------------------------------------------------------------
// One handler, both shapes
// --------------------------------------------------------------------------

// TestOneHandlerServesBothShapes is the property that makes this package worth
// having: the handler between Open and Close does not know or care which shape
// the client asked for.
func TestOneHandlerServesBothShapes(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		res, err := ai.Open(ctx)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}

		res.Send(ai.Text("m1", "Hello, "))
		res.Send(ai.Text("m1", "world"))
		res.Send(ai.Meter(ai.Usage{InputTokens: 10, OutputTokens: 2}))

		if err := res.Close(nil); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	srv := serve(t, app)

	t.Run("non-streaming by default", func(t *testing.T) {
		resp, err := http.Get(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var msg ai.Message
		if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
			t.Fatal(err)
		}

		if msg.Text != "Hello, world" {
			t.Errorf("text = %q, want %q", msg.Text, "Hello, world")
		}
		if msg.Usage == nil || msg.Usage.InputTokens != 10 {
			t.Errorf("usage = %+v", msg.Usage)
		}
	})

	t.Run("streaming when the client asks", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
			t.Fatalf("Content-Type = %q", ct)
		}

		got := parts(t, resp.Body)
		if len(got) != 2 {
			t.Fatalf("got %d parts, want 2 (coalesced text, usage): %+v", len(got), got)
		}
		if got[0]["type"] != "text" || got[0]["delta"] != "Hello, world" {
			t.Errorf("part 0 = %+v, want the two deltas coalesced", got[0])
		}
		if got[1]["type"] != "usage" {
			t.Errorf("part 1 = %+v, want usage", got[1])
		}
	})
}

func TestStreamOptionOverridesAccept(t *testing.T) {
	forced := true

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := ai.DefaultOptions()
		o.Stream = &forced

		res, err := ai.Open(ctx, o)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		res.Send(ai.Text("", "x"))
		res.Close(nil)
	})

	// No Accept header at all, and it still streams.
	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want a stream", ct)
	}
}

// --------------------------------------------------------------------------
// The part vocabulary
// --------------------------------------------------------------------------

func TestPartsRoundTrip(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := ai.DefaultOptions()
		o.Coalesce = 0 // one frame per part, so the shapes are visible.
		stream := true
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m1", "answer"))
		res.Send(ai.Reasoning("m1", "thinking"))
		res.Send(ai.ToolCall("call_1", "get_weather", json.RawMessage(`{"city":"Athens"}`)))
		res.Send(ai.ToolResult("call_1", json.RawMessage(`{"c":31}`)))
		res.Send(ai.Data("service_tier", json.RawMessage(`"priority"`)))
		res.Close(nil)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var names []string
	for ev, err := range sse.Read(resp.Body) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, ev.Name)
	}

	want := []string{"text", "reasoning", "tool", "tool-result", "service_tier", "done"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("event names = %v, want %v", names, want)
	}
}

// TestUnknownPartsPassThroughNamed guards the open namespace: a producer that
// emits something this vocabulary does not model must not lose it.
func TestUnknownPartsPassThroughNamed(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Data("container_id", json.RawMessage(`"c_123"`)))
		res.Close(nil)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var found bool
	for ev, err := range sse.Read(resp.Body) {
		if err != nil {
			t.Fatal(err)
		}
		if ev.Name == "container_id" {
			found = true
			if !strings.Contains(string(ev.Data), "c_123") {
				t.Errorf("data = %q", ev.Data)
			}
		}
	}

	if !found {
		t.Error("the unknown event was dropped instead of passed through")
	}
}

// --------------------------------------------------------------------------
// Policy and accounting hooks
// --------------------------------------------------------------------------

func TestOnPartRewriteDropAndFail(t *testing.T) {
	t.Run("rewrite and drop", func(t *testing.T) {
		app := iris.New()
		app.Get("/", func(ctx iris.Context) {
			stream := true
			o := ai.DefaultOptions()
			o.Stream = &stream
			o.Coalesce = 0
			o.OnPart = func(_ iris.Context, p ai.Part) (ai.Part, error) {
				if p.Kind() == ai.KindReasoning {
					return ai.Part{}, ai.ErrDropPart
				}
				if p.Kind() == ai.KindText {
					return ai.Text(p.ID(), strings.ToUpper(p.Delta())), nil
				}
				return p, nil
			}

			res, _ := ai.Open(ctx, o)
			res.Send(ai.Reasoning("m", "secret"))
			res.Send(ai.Text("m", "shown"))
			res.Close(nil)
		})

		resp, err := http.Get(serve(t, app).URL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		got := parts(t, resp.Body)
		if len(got) != 1 {
			t.Fatalf("got %d parts, want 1: %+v", len(got), got)
		}
		if got[0]["delta"] != "SHOWN" {
			t.Errorf("delta = %v, want the rewritten text", got[0]["delta"])
		}
	})

	t.Run("fail", func(t *testing.T) {
		blocked := stdErrors.New("blocked by policy")

		app := iris.New()
		app.Get("/", func(ctx iris.Context) {
			stream := true
			o := ai.DefaultOptions()
			o.Stream = &stream
			o.OnPart = func(iris.Context, ai.Part) (ai.Part, error) { return ai.Part{}, blocked }

			res, _ := ai.Open(ctx, o)
			if err := res.Send(ai.Text("m", "x")); !stdErrors.Is(err, blocked) {
				t.Errorf("Send error = %v, want the policy error", err)
			}
			res.Close(nil)
		})

		resp, err := http.Get(serve(t, app).URL)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	})
}

// TestOnFinishRunsOnceWithTheAssembledTurn is the accounting contract: the
// stream is an optimisation, and this is where the durable copy comes from.
func TestOnFinishRunsOnceWithTheAssembledTurn(t *testing.T) {
	var (
		mu     sync.Mutex
		calls  int
		result ai.Result
	)

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.OnFinish = func(_ stdcontext.Context, r *ai.Result) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			result = *r
		}

		res, _ := ai.Open(ctx, o)
		// A real generation always has latency before its first token, and the
		// clock on some platforms is too coarse to measure zero of it.
		time.Sleep(2 * time.Millisecond)
		res.Send(ai.Text("m", "one "))
		res.Send(ai.Text("m", "two"))
		res.Send(ai.Reasoning("m", "hmm"))
		res.Send(ai.Meter(ai.Usage{InputTokens: 7, OutputTokens: 3}))
		res.Close(nil)
		res.Close(nil) // idempotent, and OnFinish must not run twice.
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	mu.Lock()
	defer mu.Unlock()

	if calls != 1 {
		t.Fatalf("OnFinish ran %d times, want exactly 1", calls)
	}
	if result.Text != "one two" {
		t.Errorf("text = %q, want %q", result.Text, "one two")
	}
	if result.Reasoning != "hmm" {
		t.Errorf("reasoning = %q", result.Reasoning)
	}
	if result.Usage.InputTokens != 7 || result.Usage.OutputTokens != 3 {
		t.Errorf("usage = %+v", result.Usage)
	}
	if !result.Committed {
		t.Error("Committed = false, want true after a streamed answer")
	}
	if result.TTFP <= 0 {
		t.Error("TTFP was not measured")
	}
}

// --------------------------------------------------------------------------
// Failure after the response has begun
// --------------------------------------------------------------------------

// TestErrorAfterFirstPartIsInBand: once the status code is spent, the only
// honest way to report a failure is inside the stream.
func TestErrorAfterFirstPartIsInBand(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m", "partial answer"))
		res.Close(stdErrors.New("upstream died"))
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: a committed stream cannot change its status code", resp.StatusCode)
	}

	var sawError, sawDone bool
	for ev, err := range sse.Read(resp.Body) {
		if err != nil {
			t.Fatal(err)
		}
		switch ev.Name {
		case "error":
			sawError = true

			// The frame is the rest/errors envelope, and an unmapped error is
			// redacted, so the upstream's own words cannot leak into a stream.
			var reported errors.Error
			if jsonErr := json.Unmarshal(ev.Data, &reported); jsonErr != nil {
				t.Fatalf("error frame is not an errors.Error: %v (%q)", jsonErr, ev.Data)
			}
			if reported.ID == "" {
				t.Error("the error frame carries no correlation id")
			}
			if strings.Contains(string(ev.Data), "upstream died") {
				t.Errorf("the raw error text reached the client: %q", ev.Data)
			}
		case "done":
			sawDone = true
		}
	}

	if !sawError {
		t.Error("no error frame")
	}
	if !sawDone {
		t.Error("no terminator after the error: a client cannot tell this from a truncated stream")
	}
}

// TestErrorBeforeFirstPartIsAnOrdinaryResponse: nothing was committed, so the
// handler still owns the status code.
func TestErrorBeforeFirstPartIsAnOrdinaryResponse(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)
		if err := res.Close(stdErrors.New("upstream refused")); err == nil {
			t.Error("Close returned nil, want the error back for the caller to render")
		}

		ctx.StopWithJSON(http.StatusBadGateway, iris.Map{"error": "upstream refused"})
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

// --------------------------------------------------------------------------
// Encoders
// --------------------------------------------------------------------------

func TestNDJSONEncoder(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0
		o.Encoder = ai.NDJSON()

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m", "a"))
		res.Send(ai.Text("m", "b"))
		res.Close(nil)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("Content-Type = %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (two deltas and the terminal line): %q", len(lines), body)
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		want := "text"
		if i == 2 {
			want = "done"
		}
		if m["type"] != want {
			t.Errorf("line %q has type %v, want %s", line, m["type"], want)
		}
	}
}

func TestPlainTextEncoder(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0
		o.Encoder = ai.PlainText()

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Reasoning("m", "hidden"))
		res.Send(ai.Text("m", "Hello "))
		res.Send(ai.Text("m", "world"))
		res.Close(nil)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello world" {
		t.Errorf("body = %q, want %q", body, "Hello world")
	}
}

// --------------------------------------------------------------------------
// Relay
// --------------------------------------------------------------------------

// TestRelayFromAFakeUpstream is the test no Go framework can write today: a
// provider-shaped stream, relayed through the neutral vocabulary, asserted on
// the client side.
func TestRelayFromAFakeUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Written without a space after "data:" on purpose: several
		// OpenAI-compatible gateways do exactly this, and a prefix-matching
		// reader sees an empty stream.
		io.WriteString(w, "data:{\"delta\":\"Hel\"}\n\n")
		io.WriteString(w, "data: {\"delta\":\"lo\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	decoder := ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
		var chunk struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(ev.Data, &chunk); err != nil {
			return nil, err
		}
		if chunk.Delta == "" {
			return nil, nil
		}

		return []ai.Part{ai.Text("m", chunk.Delta)}, nil
	})

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)

		resp, err := http.Get(upstream.URL)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			return
		}

		res.Close(ai.Relay(res, resp, decoder, ai.RelayOptions{
			Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
		}))
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got := parts(t, resp.Body)
	if len(got) != 1 {
		t.Fatalf("got %d parts, want the two deltas coalesced into 1: %+v", len(got), got)
	}
	if got[0]["delta"] != "Hello" {
		t.Errorf("delta = %v, want %q", got[0]["delta"], "Hello")
	}
}

// TestRelayRefusesANonEventStream200 covers the gateway that answers 200 with a
// JSON error. Parsing that as an empty stream turns a failed generation into a
// blank answer, which is the worst possible outcome.
func TestRelayRefusesANonEventStream200(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"error":{"message":"model overloaded"}}`)
	}))
	defer upstream.Close()

	var relayErr error

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)

		resp, err := http.Get(upstream.URL)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			return
		}

		relayErr = ai.Relay(res, resp, ai.DecoderFunc(func(sse.Event) ([]ai.Part, error) { return nil, nil }))
		res.Close(relayErr)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if !stdErrors.Is(relayErr, ai.ErrNonEventStream) {
		t.Fatalf("error = %v, want ErrNonEventStream", relayErr)
	}
	if !strings.Contains(relayErr.Error(), "model overloaded") {
		t.Errorf("error = %v, want it to carry the upstream body", relayErr)
	}
}

// TestRelaySurfacesDecoderErrors: a decoder that cannot make sense of an
// upstream event must fail the turn rather than quietly produce nothing.
func TestRelaySurfacesDecoderErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: not json at all\n\n")
	}))
	defer upstream.Close()

	var relayErr error

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)

		resp, err := http.Get(upstream.URL)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			return
		}

		relayErr = ai.Relay(res, resp, ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
			var m map[string]any
			return nil, json.Unmarshal(ev.Data, &m)
		}))
		res.Close(relayErr)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if relayErr == nil {
		t.Fatal("Relay returned nil, want the decoder error")
	}
}

// TestRelayRejectsANilResponse keeps the function total: a nil target is a
// programming error, not a panic in production.
func TestRelayRejectsANilResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	if err := ai.Relay(nil, resp, ai.DecoderFunc(func(sse.Event) ([]ai.Part, error) { return nil, nil })); err == nil {
		t.Fatal("Relay(nil, ...) returned nil, want an error")
	}
}

// TestRelayReportsATruncatedUpstream: an upstream that closes cleanly without
// its sentinel was cut, and the client must be told so. Ending with a plain
// terminator would present three deltas as the whole answer.
func TestRelayReportsATruncatedUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, delta := range []string{"one ", "two ", "three "} {
			io.WriteString(w, "data: {\"delta\":\""+delta+"\"}\n\n")
		}
		// No [DONE]: the worker died, or a proxy idled out, and closed cleanly.
	}))
	defer upstream.Close()

	decoder := ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
		var chunk struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(ev.Data, &chunk); err != nil {
			return nil, err
		}
		return []ai.Part{ai.Text("m", chunk.Delta)}, nil
	})

	var relayErr error

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream

		res, _ := ai.Open(ctx, o)

		resp, err := http.Get(upstream.URL)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			return
		}

		relayErr = ai.Relay(res, resp, decoder, ai.RelayOptions{
			Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
		})
		res.Close(relayErr)
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var names []string
	for ev, readErr := range sse.Read(resp.Body) {
		if readErr != nil {
			t.Fatal(readErr)
		}
		names = append(names, ev.Name)
	}

	if !stdErrors.Is(relayErr, io.ErrUnexpectedEOF) {
		t.Errorf("Relay returned %v, want an error matching io.ErrUnexpectedEOF", relayErr)
	}
	// A cut upstream stream is an upstream failure, typed like any other, so a
	// gateway's failover logic and the client's error frame both see BAD_GATEWAY
	// rather than an internal error.
	ue, ok := ai.AsUpstream(relayErr)
	if !ok || ue.Class != ai.ClassTransient || ue.ErrorCode() != errors.BadGateway {
		t.Errorf("Relay returned %#v, want a transient *UpstreamError mapped to BAD_GATEWAY", relayErr)
	}
	if !slices.Contains(names, "error") {
		t.Errorf("events = %v, want an error frame before the terminator", names)
	}
	if names[len(names)-1] != "done" {
		t.Errorf("events = %v, want the terminator last so the client can still detect a torn connection", names)
	}
}

// TestForgottenCloseStopsTheDrainBeforeTheContextIsRecycled: with Coalesce on,
// a text part sits pending until the ticker flushes it. A handler that returns
// without Close leaves that ticker running against a Context the router has
// already recycled, and the flush commits a whole text/event-stream 200 into
// whatever request holds that Context next. The router wrapper holds the window
// between the Context's release and the request context's cancellation open.
func TestForgottenCloseStopsTheDrainBeforeTheContextIsRecycled(t *testing.T) {
	var (
		forgotten *ai.Response
		finished  []ai.Result
	)

	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		next(w, r)
		if r.URL.Path == "/turn" {
			time.Sleep(250 * time.Millisecond) // longer than the default Coalesce interval.
		}
	})
	app.Get("/turn", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions() // Coalesce at its default.
		o.Stream = &stream
		o.OnFinish = func(_ stdcontext.Context, r *ai.Result) { finished = append(finished, *r) }

		res, err := ai.Open(ctx, o)
		if err != nil {
			t.Errorf("Open: %v", err)
			return
		}
		forgotten = res

		res.Send(ai.Text("m", "pending")) // coalesced: nothing is on the wire yet.
		// No Close.
	})
	app.Get("/plain", func(ctx iris.Context) { ctx.WriteString("plain") })

	srv := serve(t, app)

	resp, err := http.Get(srv.URL + "/plain")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	baseline := runtime.NumGoroutine()

	resp, err = http.Get(srv.URL + "/turn")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if len(body) != 0 {
		t.Errorf("body = %q, want nothing: the pending part must not be flushed after the handler returned", body)
	}
	if forgotten.Committed() {
		t.Error("the turn committed after the handler returned")
	}
	if err := forgotten.Send(ai.Text("m", "late")); !stdErrors.Is(err, sse.ErrClosed) {
		t.Errorf("Send after the request ended = %v, want ErrClosed", err)
	}

	// The accounting leg runs on success, failure and disconnect alike; a
	// handler that forgot Close is not a reason to lose the record.
	if len(finished) != 1 {
		t.Fatalf("OnFinish ran %d times for an abandoned turn, want 1", len(finished))
	}
	if !finished[0].Abandoned || finished[0].Text != "pending" || finished[0].Committed {
		t.Errorf("OnFinish saw %+v, want Abandoned with the accumulated text and nothing committed", finished[0])
	}

	resp, err = http.Get(srv.URL + "/plain")
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(plain) != "plain" || strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		t.Errorf("the next request got %q with Content-Type %q", plain, resp.Header.Get("Content-Type"))
	}

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline {
		t.Errorf("goroutines = %d, want at most %d: the drain outlived its request", n, baseline)
	}
}

// --------------------------------------------------------------------------
// Error parts and encoder terminators
// --------------------------------------------------------------------------

// streamingHandler builds a one-route app whose handler streams with the given
// encoder, runs body, and closes with closeErr.
func streamingHandler(enc ai.Encoder, body func(res *ai.Response), closeErr error) *iris.Application {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0
		o.Encoder = enc

		res, err := ai.Open(ctx, o)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		if body != nil {
			body(res)
		}
		res.Close(closeErr)
	})

	return app
}

func fetch(t *testing.T, app *iris.Application) (*http.Response, string) {
	t.Helper()

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

// TestSentErrorPartGoesThroughPolicy: Send(ai.Error(err)) used to put
// err.Error() on the wire verbatim, while the same error given to Close was
// reported, redacted and rewritten. One error, two policies. Now a KindError
// part takes the same road as the terminal error, on every encoder.
func TestSentErrorPartGoesThroughPolicy(t *testing.T) {
	leak := stdErrors.New("openai: model gpt-x at api.openai.com failed")

	for name, enc := range map[string]ai.Encoder{"SSEJSON": ai.SSEJSON(), "NDJSON": ai.NDJSON()} {
		t.Run(name, func(t *testing.T) {
			app := streamingHandler(enc, func(res *ai.Response) {
				res.Send(ai.Text("m", "partial"))
				res.Send(ai.Error(leak))
			}, nil)

			_, body := fetch(t, app)

			for _, word := range []string{"openai", "gpt-x", "api.openai.com"} {
				if strings.Contains(body, word) {
					t.Errorf("the wire carries %q: %s", word, body)
				}
			}
			if !strings.Contains(body, `"http_error_code"`) {
				t.Errorf("no rest/errors envelope on the wire: %s", body)
			}
		})
	}

	t.Run("Redact applies to a sent error too", func(t *testing.T) {
		sentinel := stdErrors.New("the anthropic backend is busy")

		app := iris.New()
		app.UseRouter(errors.NewOptions().MapErrors(errors.Unavailable, sentinel).Handler())
		app.Get("/", func(ctx iris.Context) {
			stream := true
			o := ai.DefaultOptions()
			o.Stream = &stream
			o.Redact = strings.NewReplacer("anthropic", "the model provider").Replace

			res, _ := ai.Open(ctx, o)
			res.Send(ai.Text("m", "partial"))
			res.Send(ai.Error(sentinel))
			res.Close(nil)
		})

		_, body := fetch(t, app)
		if strings.Contains(body, "anthropic") || !strings.Contains(body, "the model provider") {
			t.Errorf("Redact did not run over the sent error: %s", body)
		}
	})
}

// TestEncodersOnClose pins the wire of every shipped encoder on a clean close
// and on a failure after the first byte. NDJSON gets an error line and a
// terminal line; PlainText has no room for either and says so in its docs.
func TestEncodersOnClose(t *testing.T) {
	failure := stdErrors.New("upstream exploded")
	send := func(res *ai.Response) {
		res.Send(ai.Text("m", "Hello "))
		res.Send(ai.Text("m", "world"))
	}

	t.Run("SSEJSON", func(t *testing.T) {
		_, ok := fetch(t, streamingHandler(ai.SSEJSON(), send, nil))
		if names := eventNames(t, ok); !slices.Equal(names, []string{"text", "text", "done"}) {
			t.Errorf("Close(nil) events = %v, want [text text done]", names)
		}

		_, failed := fetch(t, streamingHandler(ai.SSEJSON(), send, failure))
		if names := eventNames(t, failed); !slices.Equal(names, []string{"text", "text", "error", "done"}) {
			t.Errorf("Close(err) events = %v, want [text text error done]", names)
		}
		if strings.Contains(failed, "exploded") {
			t.Errorf("the raw error text reached the client: %s", failed)
		}
	})

	t.Run("NDJSON", func(t *testing.T) {
		_, ok := fetch(t, streamingHandler(ai.NDJSON(), send, nil))
		if types := lineTypes(t, ok); !slices.Equal(types, []string{"text", "text", "done"}) {
			t.Errorf("Close(nil) lines = %v, want [text text done]", types)
		}

		resp, failed := fetch(t, streamingHandler(ai.NDJSON(), send, failure))
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d: a committed stream cannot change its status code", resp.StatusCode)
		}
		if types := lineTypes(t, failed); !slices.Equal(types, []string{"text", "text", "error", "done"}) {
			t.Errorf("Close(err) lines = %v, want [text text error done]", types)
		}
		if strings.Contains(failed, "exploded") {
			t.Errorf("the raw error text reached the client: %s", failed)
		}
		if !strings.Contains(failed, `"http_error_code"`) {
			t.Errorf("the error line carries no rest/errors envelope: %s", failed)
		}
	})

	t.Run("PlainText", func(t *testing.T) {
		original := errors.LogError
		t.Cleanup(func() { errors.LogError = original })
		var logged error
		errors.LogError = func(_ iris.Context, err error) { logged = err }

		_, ok := fetch(t, streamingHandler(ai.PlainText(), send, nil))
		if ok != "Hello world" {
			t.Errorf("Close(nil) body = %q, want %q", ok, "Hello world")
		}

		_, failed := fetch(t, streamingHandler(ai.PlainText(), send, failure))
		if failed != "Hello world" {
			t.Errorf("Close(err) body = %q, want the prose and nothing else: this dialect has no in-band error", failed)
		}
		if logged == nil || !strings.Contains(logged.Error(), "exploded") {
			t.Errorf("logged = %v, want the failure reported even though the wire cannot carry it", logged)
		}
	})

	t.Run("error before the first byte is still an ordinary response", func(t *testing.T) {
		app := iris.New()
		app.Get("/", func(ctx iris.Context) {
			stream := true
			o := ai.DefaultOptions()
			o.Stream = &stream
			o.Encoder = ai.NDJSON()

			res, _ := ai.Open(ctx, o)
			if err := res.Close(failure); err != nil {
				ctx.StopWithJSON(http.StatusBadGateway, iris.Map{"error": "upstream"})
			}
		})

		resp, body := fetch(t, app)
		if resp.StatusCode != http.StatusBadGateway || strings.Contains(body, "done") {
			t.Errorf("status = %d body = %q, want a 502 with no terminal line", resp.StatusCode, body)
		}
	})
}

// TestZeroPartStreamStillTerminates: the non-streaming shape writes {} for an
// empty answer; the streaming one must commit and write its terminator, or the
// client cannot tell an empty answer from a connection that dropped before the
// first byte.
func TestZeroPartStreamStillTerminates(t *testing.T) {
	t.Run("SSEJSON", func(t *testing.T) {
		resp, body := fetch(t, streamingHandler(ai.SSEJSON(), nil, nil))
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
			t.Errorf("Content-Type = %q, want text/event-stream", ct)
		}
		if names := eventNames(t, body); !slices.Equal(names, []string{"done"}) {
			t.Errorf("events = %v, want [done]", names)
		}
	})

	t.Run("NDJSON", func(t *testing.T) {
		resp, body := fetch(t, streamingHandler(ai.NDJSON(), nil, nil))
		if ct := resp.Header.Get("Content-Type"); ct != "application/x-ndjson" {
			t.Errorf("Content-Type = %q", ct)
		}
		if types := lineTypes(t, body); !slices.Equal(types, []string{"done"}) {
			t.Errorf("lines = %v, want [done]", types)
		}
	})

	t.Run("PlainText", func(t *testing.T) {
		resp, body := fetch(t, streamingHandler(ai.PlainText(), nil, nil))
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/plain") {
			t.Errorf("Content-Type = %q, want text/plain committed even for an empty answer", ct)
		}
		if body != "" {
			t.Errorf("body = %q, want empty", body)
		}
	})
}

// TestZeroPartStreamReportsCommitted: the terminator is written inside Close,
// after the Result snapshot is taken, so a turn that sent nothing used to tell
// OnFinish that nothing was delivered - the accounting leg disagreeing with the
// wire.
func TestZeroPartStreamReportsCommitted(t *testing.T) {
	for name, enc := range map[string]ai.Encoder{
		"SSEJSON":   ai.SSEJSON(),
		"NDJSON":    ai.NDJSON(),
		"PlainText": ai.PlainText(),
	} {
		t.Run(name, func(t *testing.T) {
			var (
				mu       sync.Mutex
				finished []ai.Result
			)

			app := iris.New()
			app.Get("/", func(ctx iris.Context) {
				stream := true
				o := ai.DefaultOptions()
				o.Stream = &stream
				o.Coalesce = 0
				o.Encoder = enc
				o.OnFinish = func(_ stdcontext.Context, r *ai.Result) {
					mu.Lock()
					defer mu.Unlock()
					finished = append(finished, *r)
				}

				res, err := ai.Open(ctx, o)
				if err != nil {
					ctx.StopWithError(iris.StatusInternalServerError, err)
					return
				}
				res.Close(nil)
			})

			fetch(t, app)

			mu.Lock()
			defer mu.Unlock()

			if len(finished) != 1 {
				t.Fatalf("OnFinish ran %d times, want 1", len(finished))
			}
			if !finished[0].Committed {
				t.Error("Committed = false, want true: Close committed the response and wrote the terminator")
			}
		})
	}
}

// TestSendAfterCloseDoesNotRunOnPart: the closed check runs before the hook, so
// a hook with side effects does not fire for a part that can never be sent.
func TestSendAfterCloseDoesNotRunOnPart(t *testing.T) {
	var hookCalls int

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.OnPart = func(_ iris.Context, p ai.Part) (ai.Part, error) {
			hookCalls++
			return p, nil
		}

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m", "a"))
		res.Close(nil)

		if err := res.Send(ai.Text("m", "late")); !stdErrors.Is(err, sse.ErrClosed) {
			t.Errorf("Send after Close = %v, want ErrClosed", err)
		}
	})

	fetch(t, app)

	if hookCalls != 1 {
		t.Errorf("OnPart ran %d times, want 1: it must not see a part sent after Close", hookCalls)
	}
}

// TestResultIsACopy: Result hands out a value, so a caller cannot race Close's
// own writes to the turn's record. A snapshot taken before Close must not
// change under the caller's feet when Close runs.
func TestResultIsACopy(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		res, _ := ai.Open(ctx)
		res.Send(ai.Text("m", "answer"))

		snapshot := res.Result()
		res.Close(nil)

		if snapshot.Committed {
			t.Error("a Result taken before Close changed when Close ran: it is not a copy")
		}
		if !res.Result().Committed {
			t.Error("Result after Close does not report the document as committed")
		}
	})

	fetch(t, app)
}

// eventNames lists the event names of an SSE body, in order.
func eventNames(t *testing.T, body string) []string {
	t.Helper()

	var names []string
	for ev, err := range sse.Read(strings.NewReader(body)) {
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		names = append(names, ev.Name)
	}

	return names
}

// lineTypes lists the "type" of every NDJSON line, in order.
func lineTypes(t *testing.T, body string) []string {
	t.Helper()

	var types []string
	for line := range strings.SplitSeq(strings.TrimSpace(body), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		types = append(types, fmt.Sprint(m["type"]))
	}

	return types
}

// TestRelayCanceledClientStaysCanceled: a caller that hangs up while the
// upstream body is being read surfaces as context.Canceled from the read. That
// is not an upstream failure: it must not be retried, must not become a
// BAD_GATEWAY frame, and must not page anyone.
func TestRelayCanceledClientStaysCanceled(t *testing.T) {
	original := errors.LogError
	t.Cleanup(func() { errors.LogError = original })
	var logged atomic.Int32
	errors.LogError = func(iris.Context, error) { logged.Add(1) }

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "data: {\"delta\":\"one\"}\n\n")
		http.NewResponseController(w).Flush()
		<-r.Context().Done() // stall until the gateway gives up on us.
	}))
	defer upstream.Close()

	decoder := ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
		var chunk struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(ev.Data, &chunk); err != nil {
			return nil, err
		}
		return []ai.Part{ai.Text("m", chunk.Delta)}, nil
	})

	relayErrs := make(chan error, 1)

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0

		res, _ := ai.Open(ctx, o)

		req, _ := http.NewRequestWithContext(ctx.Request().Context(), http.MethodGet, upstream.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			relayErrs <- ue
			return
		}

		relayErr := ai.Relay(res, resp, decoder, ai.RelayOptions{
			Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
		})
		res.Close(relayErr)
		relayErrs <- relayErr
	})

	cctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, serve(t, app).URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	// Read the first delta, then hang up.
	br := bufio.NewReader(resp.Body)
	for {
		line, readErr := br.ReadString('\n')
		if readErr != nil {
			t.Fatalf("never saw the first delta: %v", readErr)
		}
		if strings.Contains(line, "one") {
			break
		}
	}
	cancel()
	resp.Body.Close()

	var relayErr error
	select {
	case relayErr = <-relayErrs:
	case <-time.After(5 * time.Second):
		t.Fatal("the relay never noticed the client leaving")
	}

	ue, ok := ai.AsUpstream(relayErr)
	if !ok {
		t.Fatalf("Relay returned %v (%T), want a typed *UpstreamError", relayErr, relayErr)
	}
	if ue.Class != ai.ClassCanceled || ue.Retryable() || ue.ErrorCode() != "" {
		t.Errorf("class = %s retryable = %t code = %q, want canceled, not retryable, no code", ue.Class, ue.Retryable(), ue.ErrorCode())
	}
	if n := logged.Load(); n != 0 {
		t.Errorf("the logger was paged %d times for a client that hung up", n)
	}
}

// TestRelayRequestDeadlineIsNotRetryable: the DisconnectCancel check fires on
// any ended request context, and a server-side deadline ends one just as a
// client hanging up does. Either way the response is over and nobody is
// waiting, so the relay must not hand its caller a retryable failure and send
// it shopping for another backend.
func TestRelayRequestDeadlineIsNotRetryable(t *testing.T) {
	const deadline = 50 * time.Millisecond

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "data: {\"delta\":\"one\"}\n\n")
		http.NewResponseController(w).Flush()

		// Long after the request deadline, so the second part is relayed with
		// the turn's context already expired.
		select {
		case <-time.After(6 * deadline):
		case <-r.Context().Done():
			return
		}

		io.WriteString(w, "data: {\"delta\":\"two\"}\n\n")
		http.NewResponseController(w).Flush()
		<-r.Context().Done()
	}))
	defer upstream.Close()

	decoder := ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
		var chunk struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(ev.Data, &chunk); err != nil {
			return nil, err
		}
		return []ai.Part{ai.Text("m", chunk.Delta)}, nil
	})

	relayErrs := make(chan error, 1)

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		// The server's own deadline on this request, the shape a timeout
		// middleware installs. The upstream call deliberately does not use it:
		// this is the turn expiring, not the upstream stalling.
		deadlined, cancel := stdcontext.WithTimeout(ctx.Request().Context(), deadline)
		defer cancel()
		ctx.SetRequest(ctx.Request().WithContext(deadlined))

		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0

		res, _ := ai.Open(ctx, o)

		req, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, upstream.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if ue := ai.Classify(resp, err); ue != nil {
			res.Close(ue)
			relayErrs <- ue
			return
		}
		defer resp.Body.Close()

		relayErr := ai.Relay(res, resp, decoder)
		res.Close(relayErr)
		relayErrs <- relayErr
	})

	resp, err := http.Get(serve(t, app).URL)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	var relayErr error
	select {
	case relayErr = <-relayErrs:
	case <-time.After(10 * time.Second):
		t.Fatal("the relay never returned")
	}

	ue, ok := ai.AsUpstream(relayErr)
	if !ok {
		t.Fatalf("Relay returned %v (%T), want a typed *UpstreamError", relayErr, relayErr)
	}
	if ue.Class != ai.ClassCanceled || ue.Retryable() {
		t.Errorf("class = %s retryable = %t, want canceled and not retryable", ue.Class, ue.Retryable())
	}
	if !stdErrors.Is(ue, stdcontext.DeadlineExceeded) {
		t.Errorf("the cause was dropped: errors.Is(err, context.DeadlineExceeded) = false (%v)", ue)
	}
}

// TestAbandonedNonStreamingTurnRunsOnFinish: the accounting promise holds for
// both shapes. A non-streaming handler that returns without Close writes no
// document, and OnFinish still gets the record, marked Abandoned.
func TestAbandonedNonStreamingTurnRunsOnFinish(t *testing.T) {
	var finished []ai.Result

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		o := ai.DefaultOptions()
		o.OnFinish = func(_ stdcontext.Context, r *ai.Result) { finished = append(finished, *r) }

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m", "answer"))
		// No Close.
	})

	_, body := fetch(t, app)

	if body != "" {
		t.Errorf("body = %q, want nothing: Close was never called", body)
	}
	if len(finished) != 1 {
		t.Fatalf("OnFinish ran %d times, want 1", len(finished))
	}
	if r := finished[0]; !r.Abandoned || r.Committed || r.Streaming || r.Text != "answer" {
		t.Errorf("OnFinish saw %+v, want Abandoned, not committed, not streaming, with the text", r)
	}
}

// TestReleaseWaitsForAnInFlightClose: a Close still writing its terminal
// frames from another goroutine when the handler returns must finish before
// the router recycles the writer. NDJSON writes through the raw sink with no
// stream lock to hide behind, so this is where a race would show.
func TestReleaseWaitsForAnInFlightClose(t *testing.T) {
	closeStarted := make(chan struct{})

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		o.Coalesce = 0
		o.Encoder = ai.NDJSON()
		o.OnPart = func(_ iris.Context, p ai.Part) (ai.Part, error) {
			if p.Kind() == ai.KindError {
				// Close is now past the point where it marked the turn closed
				// and before it has written anything terminal.
				close(closeStarted)
				time.Sleep(50 * time.Millisecond)
			}
			return p, nil
		}

		res, _ := ai.Open(ctx, o)
		res.Send(ai.Text("m", "a"))

		go res.Close(stdErrors.New("late failure"))
		<-closeStarted
		// The handler returns while Close is mid-flight.
	})

	_, body := fetch(t, app)

	if types := lineTypes(t, body); !slices.Equal(types, []string{"text", "error", "done"}) {
		t.Errorf("lines = %v, want [text error done]: the in-flight Close was cut off by the release", types)
	}
}
