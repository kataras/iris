package ai

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Class is what a caller actually needs to decide from an upstream failure:
// whether to retry, whether to try a different credential, and whether to give
// up.
//
// It exists because the alternative is what every AI gateway ends up with - a
// regular expression over err.Error(), because the status code was flattened
// into a string three packages ago and there is nowhere else left to read it.
type Class uint8

const (
	// ClassNone means the response was fine.
	ClassNone Class = iota

	// ClassFatal is a request-shaped failure: the request is wrong and would be
	// just as wrong on another credential or another model. Retrying spends
	// money to get the same answer.
	ClassFatal

	// ClassTransient is a failure that may not recur: a 5xx, a reset, a dial
	// error. Retrying is reasonable.
	ClassTransient

	// ClassRateLimited is a 429. It is deliberately its own class rather than a
	// kind of exhaustion: it is a routing signal, not a health signal, and a
	// credential pool that counts it as a fault will quarantine a perfectly
	// healthy key for being popular.
	ClassRateLimited

	// ClassExhausted means this credential or account cannot serve the request
	// at all: it is unauthorised, out of quota, or suspended. A different
	// credential might work; this one will not, for a while.
	ClassExhausted

	// ClassCanceled means the caller went away. It is not a failure of the
	// upstream, and a second attempt would spend tokens on a response nobody is
	// waiting for.
	ClassCanceled
)

// String returns a stable, lowercase name, suitable for a metric label.
func (c Class) String() string {
	switch c {
	case ClassNone:
		return "none"
	case ClassFatal:
		return "fatal"
	case ClassTransient:
		return "transient"
	case ClassRateLimited:
		return "rate_limited"
	case ClassExhausted:
		return "exhausted"
	case ClassCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// Retryable reports whether trying again could plausibly help.
func (c Class) Retryable() bool {
	return c == ClassTransient || c == ClassRateLimited
}

// DefaultMaxErrorBody caps how much of a failed response Classify keeps.
const DefaultMaxErrorBody = 64 << 10

// UpstreamError is a typed failure from a provider.
type UpstreamError struct {
	// Status is the HTTP status code, or 0 when the request never got one.
	Status int
	// Class is what to do about it.
	Class Class
	// Message is the provider's own message, when the body carried one in any of
	// the shapes providers use.
	Message string
	// Code is the provider's machine-readable code, when there was one.
	Code string
	// RequestID is the provider's request identifier, from the body or from one
	// of the usual headers. It is the only thing that makes a support
	// conversation about a single failed generation possible.
	RequestID string
	// RetryAfter is the delay the provider asked for, from Retry-After.
	RetryAfter time.Duration
	// Body is the response body, truncated. Nil when there was no response.
	Body []byte

	// err is the transport error, when the request never completed.
	err error
}

// Error implements error.
func (e *UpstreamError) Error() string {
	var b strings.Builder

	b.WriteString("upstream ")
	b.WriteString(e.Class.String())

	if e.Status > 0 {
		fmt.Fprintf(&b, " [%d]", e.Status)
	}
	if e.Code != "" {
		fmt.Fprintf(&b, " %s", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	} else if e.err != nil {
		fmt.Fprintf(&b, ": %v", e.err)
	}

	return b.String()
}

// Unwrap exposes the transport error, so errors.Is against context.Canceled and
// the net package's own errors keeps working.
func (e *UpstreamError) Unwrap() error { return e.err }

// Retryable reports whether trying again could plausibly help.
func (e *UpstreamError) Retryable() bool { return e.Class.Retryable() }

// Timeout reports whether the request failed because the upstream did not
// answer in time: a context deadline, or a transport error whose Timeout() is
// true. net/http's own timeout errors - http.Client.Timeout,
// Transport.ResponseHeaderTimeout - are both at once; the net.Error check is
// for timeouts from elsewhere, such as a net.Conn read deadline, which matches
// os.ErrDeadlineExceeded and not context.DeadlineExceeded. It is transient - a
// stalled provider is the failure a gateway sees most - and it is reported to
// the client as DEADLINE_EXCEEDED.
func (e *UpstreamError) Timeout() bool { return isTimeout(e.err) }

// isTimeout tells a stalled upstream apart from a caller that went away.
//
// net/http's timeout errors satisfy errors.Is(err, context.DeadlineExceeded) as
// well as net.Error, so a check on context.DeadlineExceeded alone cannot tell a
// stalled upstream from a deadline the caller set, and neither check alone
// covers a raw connection deadline. Neither is the caller's cancellation: that
// one is context.Canceled, and only that one. Relay is the single exception:
// when the turn's own context has ended under DisconnectCancel it reports
// ClassCanceled whatever ended it, keeping the cause, since nobody is waiting
// for a retry.
func isTimeout(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, stdcontext.DeadlineExceeded) {
		return true
	}

	netErr, ok := errors.AsType[net.Error](err)
	return ok && netErr.Timeout()
}

// HTTPStatus is the status code to give the client for this failure.
//
// It is not the upstream's status code, and the difference matters. A 401 from
// a provider means the gateway's credential is finished, which is a 502 to the
// caller: telling them they are unauthorised would send them off to check their
// own key. The one code that does pass through is 429, because rate limiting is
// the caller's problem too and the delay is worth forwarding.
func (e *UpstreamError) HTTPStatus() int {
	switch e.Class {
	case ClassRateLimited:
		return http.StatusTooManyRequests
	case ClassExhausted:
		return http.StatusServiceUnavailable
	case ClassFatal:
		if e.Status == http.StatusRequestEntityTooLarge {
			return http.StatusRequestEntityTooLarge
		}

		return http.StatusBadRequest
	case ClassCanceled:
		// The caller is gone, so nothing will read this. 499 is nginx's
		// convention and is what shows up usefully in an access log.
		return 499
	default:
		if e.Timeout() {
			return http.StatusGatewayTimeout
		}

		return http.StatusBadGateway
	}
}

// AsUpstream extracts an *UpstreamError from an error chain.
func AsUpstream(err error) (*UpstreamError, bool) {
	var ue *UpstreamError
	ok := errors.As(err, &ue)
	return ue, ok
}

// ClassifyOptions tune Classify.
type ClassifyOptions struct {
	// MaxBody caps how much of the response body is read and kept. Default
	// DefaultMaxErrorBody.
	MaxBody int64
}

// Classify turns an upstream exchange into a typed failure, or nil when nothing
// went wrong.
//
// It takes the response and not only the error, and that is the whole point. A
// well-behaved Go HTTP client returns a non-2xx response with a nil error, so a
// Classify(err) signature would have to recover the status code from a string
// and would recreate exactly the type erasure it exists to delete.
//
// It reads and closes the response body on failure, keeping up to MaxBody of it,
// because the body is where the provider put the reason. On success it leaves
// the body untouched for the caller to stream.
func Classify(resp *http.Response, err error, opts ...ClassifyOptions) *UpstreamError {
	o := ClassifyOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.MaxBody <= 0 {
		o.MaxBody = DefaultMaxErrorBody
	}

	if err != nil {
		return classifyTransportError(err)
	}

	if resp == nil || resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	ue := &UpstreamError{
		Status: resp.StatusCode,
		Class:  classForStatus(resp.StatusCode),
	}

	if resp.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, o.MaxBody))
		resp.Body.Close()
		ue.Body = body
		ue.Message, ue.Code, ue.RequestID = parseErrorBody(body)
	}

	if ue.RequestID == "" {
		ue.RequestID = requestIDFromHeader(resp.Header)
	}
	ue.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))

	return ue
}

// classifyTransportError handles the case where the request never completed.
func classifyTransportError(err error) *UpstreamError {
	ue := &UpstreamError{err: err}

	if errors.Is(err, stdcontext.Canceled) {
		// Nobody is waiting for this any more. A retry here is the one that
		// costs money for nothing.
		ue.Class = ClassCanceled
		return ue
	}

	// Everything else that stopped the request from completing - a dial failure,
	// a reset, a TLS error, a timeout - is worth another attempt, possibly
	// elsewhere. A deadline is deliberately not a cancellation: net/http's
	// timeout errors match context.DeadlineExceeded, and a stalled provider is
	// exactly the failure the next backend exists for. See Timeout.
	ue.Class = ClassTransient

	return ue
}

// classForStatus maps a status code to what a caller should do about it.
//
// The mapping is deliberately opinionated where the status alone is ambiguous.
// 401 and 403 are exhaustion rather than fatal, because in a gateway they almost
// always mean this credential is finished rather than this request is malformed,
// and treating them as fatal is how a rotated key takes the whole service down.
func classForStatus(status int) Class {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusPaymentRequired:
		return ClassExhausted
	case http.StatusTooManyRequests:
		return ClassRateLimited
	case http.StatusRequestTimeout, http.StatusConflict:
		return ClassTransient
	case 499: // nginx's client-closed-request.
		return ClassCanceled
	}

	switch {
	case status >= 500:
		// 529 is Anthropic's "overloaded" and behaves like any other 5xx.
		return ClassTransient
	case status >= 400:
		return ClassFatal
	default:
		return ClassNone
	}
}

// errorBody covers the shapes providers actually use, which are mutually
// incompatible and both common: a nested object with a discriminator, and a
// nested object without one. Every field is optional, so a shape this does not
// know still parses to empty rather than failing.
type errorBody struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
	Message   string `json:"message"`
	Detail    string `json:"detail"`
	RequestID string `json:"request_id"`
}

func parseErrorBody(body []byte) (message, code, requestID string) {
	if len(body) == 0 {
		return "", "", ""
	}

	var parsed errorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		// Not JSON. A short plain-text body is still better than nothing; a long
		// one is an HTML error page and worth nothing.
		text := strings.TrimSpace(string(body))
		if len(text) > 0 && len(text) <= 200 && !strings.HasPrefix(text, "<") {
			return text, "", ""
		}

		return "", "", ""
	}

	message = firstNonEmpty(parsed.Error.Message, parsed.Message, parsed.Detail)
	code = firstNonEmpty(parsed.Error.Type, stringify(parsed.Error.Code))
	requestID = parsed.RequestID

	return message, code, requestID
}

func requestIDFromHeader(h http.Header) string {
	for _, key := range []string{"Request-Id", "X-Request-Id", "X-Amzn-Requestid"} {
		if v := h.Get(key); v != "" {
			return v
		}
	}

	return ""
}

// parseRetryAfter accepts both forms RFC 9110 allows: a delay in seconds and an
// HTTP date.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}

	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		if secs < 0 {
			return 0
		}

		return time.Duration(secs) * time.Second
	}

	if at, err := http.ParseTime(v); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}

	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(t)
	}
}
