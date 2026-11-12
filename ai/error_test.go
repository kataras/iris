package ai_test

import (
	stdcontext "context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14/ai"
	resterrors "github.com/kataras/iris/v14/rest/errors"
)

func response(status int, contentType, body string, headers ...string) *http.Response {
	h := http.Header{}
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		h.Set(headers[i], headers[i+1])
	}

	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// TestClassifyReadsTheResponseNotTheError is the reason the signature takes a
// response. A well-behaved Go client returns a non-2xx with a nil error, so a
// Classify(err) form would have to recover the status code from a string.
func TestClassifyReadsTheResponseNotTheError(t *testing.T) {
	ue := ai.Classify(response(429, "application/json", `{"error":{"message":"slow down","type":"rate_limit_error"}}`), nil)

	if ue == nil {
		t.Fatal("Classify returned nil for a 429 with a nil error")
	}
	if ue.Status != 429 {
		t.Errorf("status = %d", ue.Status)
	}
	if ue.Class != ai.ClassRateLimited {
		t.Errorf("class = %s, want rate_limited", ue.Class)
	}
	if ue.Message != "slow down" {
		t.Errorf("message = %q", ue.Message)
	}
	if ue.Code != "rate_limit_error" {
		t.Errorf("code = %q", ue.Code)
	}
}

func TestClassifySuccessIsNil(t *testing.T) {
	if ue := ai.Classify(response(200, "text/event-stream", ""), nil); ue != nil {
		t.Fatalf("Classify returned %v for a 200", ue)
	}
	if ue := ai.Classify(nil, nil); ue != nil {
		t.Fatalf("Classify returned %v for nothing at all", ue)
	}
}

func TestClassifyStatusMapping(t *testing.T) {
	for status, want := range map[int]ai.Class{
		400: ai.ClassFatal,
		404: ai.ClassFatal,
		422: ai.ClassFatal,
		401: ai.ClassExhausted,
		403: ai.ClassExhausted,
		402: ai.ClassExhausted,
		429: ai.ClassRateLimited,
		500: ai.ClassTransient,
		502: ai.ClassTransient,
		529: ai.ClassTransient, // Anthropic's "overloaded".
		499: ai.ClassCanceled,
	} {
		ue := ai.Classify(response(status, "", ""), nil)
		if ue == nil {
			t.Errorf("status %d: Classify returned nil", status)
			continue
		}
		if ue.Class != want {
			t.Errorf("status %d: class = %s, want %s", status, ue.Class, want)
		}
	}
}

// TestClassifyCredentialFailuresAreExhaustionNotFatal records a deliberate
// choice: in a gateway a 401 almost always means this credential is finished
// rather than this request is malformed, and treating it as fatal is how a
// rotated key takes the whole service down instead of failing over.
func TestClassifyCredentialFailuresAreExhaustionNotFatal(t *testing.T) {
	ue := ai.Classify(response(401, "", ""), nil)

	if ue.Class != ai.ClassExhausted {
		t.Fatalf("class = %s, want exhausted", ue.Class)
	}
	if ue.Retryable() {
		t.Error("an exhausted credential reported itself retryable")
	}
}

func TestClassifyRateLimitIsRetryable(t *testing.T) {
	ue := ai.Classify(response(429, "", "", "Retry-After", "7"), nil)

	if !ue.Retryable() {
		t.Error("a 429 reported itself not retryable")
	}
	if ue.RetryAfter != 7*time.Second {
		t.Errorf("RetryAfter = %s, want 7s", ue.RetryAfter)
	}
}

func TestClassifyRetryAfterHTTPDate(t *testing.T) {
	at := time.Now().UTC().Add(30 * time.Second).Format(http.TimeFormat)

	ue := ai.Classify(response(503, "", "", "Retry-After", at), nil)

	if ue.RetryAfter <= 0 || ue.RetryAfter > 31*time.Second {
		t.Fatalf("RetryAfter = %s, want roughly 30s", ue.RetryAfter)
	}
}

func TestClassifyCancellation(t *testing.T) {
	ue := ai.Classify(nil, stdcontext.Canceled)

	if ue.Class != ai.ClassCanceled {
		t.Fatalf("class = %s, want canceled", ue.Class)
	}
	if ue.Retryable() {
		t.Error("a cancelled request reported itself retryable: a retry would spend tokens on an answer nobody is waiting for")
	}
	if !errors.Is(ue, stdcontext.Canceled) {
		t.Error("the transport error is not reachable through errors.Is")
	}
}

// TestClassifyBothErrorBodyDialects: the two shapes providers use are mutually
// incompatible and both common, so a parser has to accept either.
func TestClassifyBothErrorBodyDialects(t *testing.T) {
	t.Run("nested with discriminator", func(t *testing.T) {
		ue := ai.Classify(response(400, "application/json",
			`{"type":"error","error":{"type":"invalid_request_error","message":"bad model"},"request_id":"req_123"}`), nil)

		if ue.Message != "bad model" {
			t.Errorf("message = %q", ue.Message)
		}
		if ue.RequestID != "req_123" {
			t.Errorf("request id = %q", ue.RequestID)
		}
	})

	t.Run("nested without discriminator", func(t *testing.T) {
		ue := ai.Classify(response(400, "application/json",
			`{"error":{"message":"bad key","type":"invalid_request_error","code":"invalid_api_key"}}`,
			"X-Request-Id", "req_456"), nil)

		if ue.Message != "bad key" {
			t.Errorf("message = %q", ue.Message)
		}
		if ue.RequestID != "req_456" {
			t.Errorf("request id = %q, want it read from the header", ue.RequestID)
		}
	})

	t.Run("plain text", func(t *testing.T) {
		ue := ai.Classify(response(502, "text/plain", "upstream connect error"), nil)

		if ue.Message != "upstream connect error" {
			t.Errorf("message = %q", ue.Message)
		}
	})

	t.Run("html error page is not a message", func(t *testing.T) {
		ue := ai.Classify(response(502, "text/html", "<html><head><title>502</title></head><body>...</body></html>"), nil)

		if ue.Message != "" {
			t.Errorf("message = %q, want an HTML page ignored", ue.Message)
		}
	})
}

func TestAsUpstream(t *testing.T) {
	ue := ai.Classify(response(500, "", ""), nil)
	wrapped := errors.Join(errors.New("relaying"), ue)

	got, ok := ai.AsUpstream(wrapped)
	if !ok {
		t.Fatal("AsUpstream did not find the error through a wrap")
	}
	if got.Status != 500 {
		t.Errorf("status = %d", got.Status)
	}
}

func TestUpstreamErrorMessage(t *testing.T) {
	ue := ai.Classify(response(429, "application/json", `{"error":{"message":"slow down","type":"rate_limit_error"}}`), nil)

	msg := ue.Error()
	for _, want := range []string{"rate_limited", "429", "rate_limit_error", "slow down"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}

func TestClassifyCapsTheBody(t *testing.T) {
	ue := ai.Classify(response(500, "text/plain", strings.Repeat("x", 10_000)), nil,
		ai.ClassifyOptions{MaxBody: 128})

	if len(ue.Body) != 128 {
		t.Fatalf("kept %d bytes, want 128", len(ue.Body))
	}
}

func TestUpstreamErrorHTTPStatus(t *testing.T) {
	// The client-facing status is not the upstream's. A provider 401 means this
	// gateway's credential is finished, and telling the caller they are
	// unauthorised would send them to check a key they do not own.
	for status, want := range map[int]int{
		401: http.StatusServiceUnavailable,
		403: http.StatusServiceUnavailable,
		429: http.StatusTooManyRequests,
		500: http.StatusBadGateway,
		400: http.StatusBadRequest,
		499: 499,
	} {
		got := ai.Classify(response(status, "", ""), nil).HTTPStatus()
		if got != want {
			t.Errorf("upstream %d: HTTPStatus() = %d, want %d", status, got, want)
		}
	}
}

// timeoutNetError is the minimal net.Error a stalled transport produces.
type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "i/o timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

// TestClassifyTimeoutIsTransient: a stalled upstream is not a caller that went
// away. net/http's own timeout error satisfies errors.Is(err,
// context.DeadlineExceeded), so treating every deadline as "nobody is waiting"
// turned a dead provider into a silent, unretried, unreported empty 200.
func TestClassifyTimeoutIsTransient(t *testing.T) {
	stalled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer stalled.Close()

	client := &http.Client{Timeout: 10 * time.Millisecond}
	_, clientErr := client.Get(stalled.URL)
	if clientErr == nil {
		t.Fatal("the stalled server answered: the fixture is wrong")
	}

	for name, err := range map[string]error{
		"context.DeadlineExceeded": stdcontext.DeadlineExceeded,
		"net.Error Timeout()":      fmt.Errorf("dial: %w", timeoutNetError{}),
		"http.Client.Timeout":      clientErr,
	} {
		t.Run(name, func(t *testing.T) {
			ue := ai.Classify(nil, err)

			if ue.Class != ai.ClassTransient {
				t.Errorf("class = %s, want transient", ue.Class)
			}
			if !ue.Retryable() {
				t.Error("a timeout reported itself not retryable")
			}
			if !ue.Timeout() {
				t.Error("Timeout() = false")
			}
			if code := ue.ErrorCode(); code != resterrors.DeadlineExceeded {
				t.Errorf("ErrorCode() = %q, want DEADLINE_EXCEEDED", code)
			}
			if ue.HTTPStatus() != http.StatusGatewayTimeout {
				t.Errorf("HTTPStatus() = %d, want 504", ue.HTTPStatus())
			}
		})
	}

	t.Run("a plain dial failure is not a timeout", func(t *testing.T) {
		ue := ai.Classify(nil, fmt.Errorf("dial tcp: connection refused"))
		if ue.Timeout() {
			t.Error("Timeout() = true for a connection refused")
		}
		if code := ue.ErrorCode(); code != resterrors.BadGateway {
			t.Errorf("ErrorCode() = %q, want BAD_GATEWAY", code)
		}
	})
}
