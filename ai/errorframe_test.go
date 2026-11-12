package ai_test

import (
	stdcontext "context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/ai"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

// errorFrameOf drives a streaming handler and returns the decoded error frame.
func errorFrameOf(t *testing.T, app *iris.Application) errors.Error {
	t.Helper()

	req, _ := http.NewRequest(http.MethodGet, serve(t, app).URL, nil)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var reported errors.Error
	var found bool

	for ev, readErr := range sse.Read(resp.Body) {
		if readErr != nil {
			t.Fatalf("reading stream: %v", readErr)
		}
		if ev.Name != "error" {
			continue
		}

		if jsonErr := json.Unmarshal(ev.Data, &reported); jsonErr != nil {
			t.Fatalf("error frame is not an errors.Error: %v (%q)", jsonErr, ev.Data)
		}
		found = true
	}

	if !found {
		t.Fatal("no error frame in the stream")
	}

	return reported
}

// streamThatFailsWith builds a handler that emits one part and then fails.
func streamThatFailsWith(failure error, edit func(*ai.Options)) *iris.Application {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		stream := true
		o := ai.DefaultOptions()
		o.Stream = &stream
		if edit != nil {
			edit(&o)
		}

		res, err := ai.Open(ctx, o)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		res.Send(ai.Text("m", "partial"))
		res.Close(failure)
	})

	return app
}

// TestUpstreamFailureKeepsItsCanonicalCode: a classified failure must not
// collapse into a generic internal error just because it happened mid-stream.
func TestUpstreamFailureKeepsItsCanonicalCode(t *testing.T) {
	upstream := ai.Classify(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"anthropic is out of capacity"}}`)),
	}, nil)

	reported := errorFrameOf(t, streamThatFailsWith(upstream, nil))

	if reported.ErrorCode.CanonicalName != "RESOURCE_EXHAUSTED" {
		t.Errorf("canonical name = %q, want RESOURCE_EXHAUSTED", reported.ErrorCode.CanonicalName)
	}
	if reported.ErrorCode.Status != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", reported.ErrorCode.Status)
	}
}

// TestUpstreamMessageDoesNotLeak is the whole point of routing through
// rest/errors: the provider's own wording never reaches the client unless the
// application deliberately mapped it.
func TestUpstreamMessageDoesNotLeak(t *testing.T) {
	upstream := ai.Classify(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"claude-opus-4 overloaded on vertex-ai"}}`)),
	}, nil)

	reported := errorFrameOf(t, streamThatFailsWith(upstream, nil))

	for _, leaked := range []string{"claude", "opus", "vertex"} {
		if strings.Contains(strings.ToLower(reported.Message+reported.Detail), leaked) {
			t.Errorf("the client was told %q, which names the upstream", reported.Message)
		}
	}
}

// TestRedactRewritesClientFacingText covers the case rest/errors cannot: an
// error the application mapped and described on purpose, whose description still
// names a vendor.
func TestRedactRewritesClientFacingText(t *testing.T) {
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
		res.Close(sentinel)
	})

	reported := errorFrameOf(t, app)

	if strings.Contains(reported.Message, "anthropic") {
		t.Errorf("message = %q, want the vendor name replaced", reported.Message)
	}
	if !strings.Contains(reported.Message, "the model provider") {
		t.Errorf("message = %q, want the replacement text", reported.Message)
	}
}

// TestOnPartSeesTheTerminalError closes the gap that made the OnPart
// documentation false: the error Close writes went straight to the wire without
// passing the one hook that governs everything else.
func TestOnPartSeesTheTerminalError(t *testing.T) {
	t.Run("rewrite", func(t *testing.T) {
		app := streamThatFailsWith(stdErrors.New("internal detail"), func(o *ai.Options) {
			o.OnPart = func(_ iris.Context, p ai.Part) (ai.Part, error) {
				if p.Kind() == ai.KindError {
					return ai.Error(stdErrors.New("the assistant is unavailable")), nil
				}
				return p, nil
			}
			// The rewritten error is the application's own words, so it is safe
			// to send as-is.
			o.SSE.ErrorFrame = sse.RawErrorFrame
		})

		req, _ := http.NewRequest(http.MethodGet, serve(t, app).URL, nil)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var errorData string
		for ev, readErr := range sse.Read(resp.Body) {
			if readErr != nil {
				t.Fatal(readErr)
			}
			if ev.Name == "error" {
				errorData = string(ev.Data)
			}
		}

		if errorData != "the assistant is unavailable" {
			t.Errorf("error frame = %q, want the rewritten text", errorData)
		}
	})

	t.Run("drop", func(t *testing.T) {
		app := streamThatFailsWith(stdErrors.New("internal detail"), func(o *ai.Options) {
			o.OnPart = func(_ iris.Context, p ai.Part) (ai.Part, error) {
				if p.Kind() == ai.KindError {
					return ai.Part{}, ai.ErrDropPart
				}
				return p, nil
			}
		})

		req, _ := http.NewRequest(http.MethodGet, serve(t, app).URL, nil)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var sawError, sawDone bool
		for ev, readErr := range sse.Read(resp.Body) {
			if readErr != nil {
				t.Fatal(readErr)
			}
			switch ev.Name {
			case "error":
				sawError = true
			case "done":
				sawDone = true
			}
		}

		if sawError {
			t.Error("an error frame was written after the hook dropped it")
		}
		if !sawDone {
			t.Error("the terminator is missing: a dropped error must not cost the truncation signal")
		}
	})
}

// TestCancelledRequestReportsNothing: nobody is listening, so nothing is
// written to the client and nobody is paged.
func TestCancelledRequestReportsNothing(t *testing.T) {
	cancelled := ai.Classify(nil, stdcontext.Canceled)

	if cancelled.Class != ai.ClassCanceled {
		t.Fatalf("class = %s, want canceled", cancelled.Class)
	}
	if code := cancelled.ErrorCode(); code.String() != "" {
		t.Errorf("ErrorCode() = %q, want empty so nothing is reported", code)
	}
	if cancelled.Report(nil) != nil {
		t.Error("Report returned an Error for a cancelled request")
	}
}

// TestLoggerSeesTheUnredactedUpstreamError: the client gets a generic message,
// and the sink gets the truth. Both halves matter.
func TestLoggerSeesTheUnredactedUpstreamError(t *testing.T) {
	original := errors.LogError
	t.Cleanup(func() { errors.LogError = original })

	logged := make(chan error, 1)
	errors.LogError = func(_ iris.Context, err error) {
		select {
		case logged <- err:
		default:
		}
	}

	upstream := ai.Classify(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"gpt-5 refused the request"}}`)),
	}, nil)

	app := streamThatFailsWith(upstream, nil)

	req, _ := http.NewRequest(http.MethodGet, serve(t, app).URL, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	select {
	case err := <-logged:
		if !strings.Contains(err.Error(), "gpt-5 refused the request") {
			t.Errorf("logged error = %v, want the unredacted upstream message", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the mid-stream failure never reached the logger")
	}
}

// TestUpstreamTimeoutIsReportedInBand: a provider that stalls after the first
// token is the failure a gateway sees most, and it used to end the stream with
// a bare terminator, no log line and no collector call.
func TestUpstreamTimeoutIsReportedInBand(t *testing.T) {
	timedOut := ai.Classify(nil, fmt.Errorf("read: %w", stdcontext.DeadlineExceeded))

	reported := errorFrameOf(t, streamThatFailsWith(timedOut, nil))

	if reported.ErrorCode.CanonicalName != "DEADLINE_EXCEEDED" {
		t.Errorf("canonical name = %q, want DEADLINE_EXCEEDED", reported.ErrorCode.CanonicalName)
	}
	if reported.ErrorCode.Status != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", reported.ErrorCode.Status)
	}
}
