// Package requestid assigns an ID to every request and exposes it through
// Context.ID, the "X-Request-Id" header and the `Get` function.
//
// Its entry point is New(generator ...Generator), which returns a Handler. The
// variadic parameter is NOT a functional-options list: it is an optional
// domain-typed Generator (at most one is read), so that `requestid.New()` and
// `requestid.New(requestid.ServerOnlyGenerator)` both read naturally. The
// package has no settings beyond that, so there is no Options struct.
package requestid

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httputil"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/uid"
)

func init() {
	context.SetHandlerName("iris/middleware/requestid.*", "iris.request.id")
}

const xRequestIDHeaderKey = "X-Request-Id"

// Generator defines the function which should extract or generate
// a Request ID. See `DefaultGenerator` and `New` package-level functions.
type Generator func(ctx *context.Context) string

// DefaultGenerator is the default `Generator` that is used
// when nil is passed on `New` package-level function.
// It extracts the ID from the "X-Request-ID" request header value
// or, if missing, it generates a new UUID(v4) and sets the header and context value.
//
// Security note: DefaultGenerator trusts a client-supplied "X-Request-ID"
// header. If request IDs end up in logs or are otherwise trusted downstream and
// clients should not be able to choose or forge them (log injection / forgery),
// use ServerOnlyGenerator instead.
//
// See `Get` package-level function too.
var DefaultGenerator Generator = func(ctx *context.Context) string {
	id := ctx.ResponseWriter().Header().Get(xRequestIDHeaderKey)
	if id != "" {
		return id
	}

	// Note(@kataras): a client-supplied id is VALIDATED before it is accepted.
	//
	// This value flows into ctx.SetID and from there into log lines and error
	// payloads. ctx.Header joins duplicate headers with ", " and nothing bounded
	// the length or the character set, so a request could inject separators,
	// control characters or a multi-kilobyte value into every log line written
	// for it. Anything that is not a plain, short token is discarded in favour of
	// a fresh server-generated id.
	if v := ctx.Header(xRequestIDHeaderKey); isValidRequestID(v) {
		id = v
	}

	if id == "" {
		// The standard library's uuid generator has no error path: an entropy
		// failure panics inside it and surfaces through the recovery
		// middleware (still a 500, via recovery).
		id = uid.NewString()
	}

	ctx.SetHeader(xRequestIDHeaderKey, id)
	return id
}

// ServerOnlyGenerator is a `Generator` that always generates a new server-side
// UUID(v4) and ignores any client-supplied "X-Request-ID" request header. Use it
// instead of DefaultGenerator when clients must not be able to choose or forge
// the request ID, for example to prevent log forgery via the header:
//
//	app.Use(requestid.New(requestid.ServerOnlyGenerator))
var ServerOnlyGenerator Generator = func(ctx *context.Context) string {
	// Honor an ID already assigned earlier in this response, but never the
	// inbound request header.
	if id := ctx.ResponseWriter().Header().Get(xRequestIDHeaderKey); id != "" {
		return id
	}

	id := uid.NewString() // an entropy failure panics, see DefaultGenerator.
	ctx.SetHeader(xRequestIDHeaderKey, id)
	return id
}

// HashGenerator uses the request's hash to generate a fixed-length Request ID.
// Note that one or many requests may contain the same ID, so it's not unique.
func HashGenerator(includeBody bool) Generator {
	return func(ctx *context.Context) string {
		ctx.SetHeader(xRequestIDHeaderKey, Hash(ctx, includeBody))
		return DefaultGenerator(ctx)
	}
}

// New returns a new request id middleware.
// It optionally accepts an ID Generator.
// The Generator can stop the handlers chain with an error or
// return a valid ID (string).
// If it's missing then the `DefaultGenerator` will be used instead.
//
// The variadic parameter is an optional Generator, not a functional-options
// list: only the first one is read.
func New(generator ...Generator) context.Handler {
	gen := DefaultGenerator
	if len(generator) > 0 {
		gen = generator[0]
	}

	return func(ctx *context.Context) {
		if Get(ctx) != "" {
			ctx.Next()
			return
		}

		id := gen(ctx)
		if ctx.IsStopped() {
			// ctx.Next checks that
			// but we don't want to call SetID if generator failed.
			return
		}

		ctx.SetID(id)
		ctx.Next()
	}
}

// Get returns the Request ID or empty string.
//
// A shortcut of `Context.ID().(string)`.
func Get(ctx *context.Context) string {
	v := ctx.ID()
	if v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}

	return ""
}

// Hash returns the hex-encoded sha256 hash of the request.
// It does not capture error, instead it returns an empty string.
func Hash(ctx *context.Context, includeBody bool) string {
	h := sha256.New() // sha1 fits here as well.
	b, err := httputil.DumpRequest(ctx.Request(), includeBody)
	if err != nil {
		return ""
	}
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// maxRequestIDLength bounds an accepted client-supplied request id.
const maxRequestIDLength = 64

// isValidRequestID reports whether a client-supplied id is safe to adopt: short,
// and made only of printable ASCII.
//
// Control characters are the ones that matter - a CR or LF here forges a whole
// log line, and an ESC rewrites a terminal - and the length cap stops a client
// from prepending kilobytes to every log entry written for its request. Spaces
// and other printable characters are allowed, so an id a client already uses for
// tracing still round-trips.
func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}

	for i := range len(id) {
		if c := id[i]; c < 0x20 || c >= 0x7f {
			return false
		}
	}

	return true
}
