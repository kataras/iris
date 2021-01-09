package requestid

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httputil"

	"github.com/kataras/iris/v12/context"

	"github.com/google/uuid"
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
// See `Get` package-level function too.
var DefaultGenerator Generator = func(ctx *context.Context) string {
	id := ctx.ResponseWriter().Header().Get(xRequestIDHeaderKey)
	if id != "" {
		return id
	}

	id = ctx.GetHeader(xRequestIDHeaderKey)
	if id == "" {
		uid, err := uuid.NewRandom()
		if err != nil {
			ctx.StopWithStatus(500)
			return ""
		}

		id = uid.String()
	}

	ctx.Header(xRequestIDHeaderKey, id)
	return id
}

// HashGenerator uses the request's hash to generate a fixed-length Request ID.
// Note that one or many requests may contain the same ID, so it's not unique.
func HashGenerator(includeBody bool) Generator {
	return func(ctx *context.Context) string {
		ctx.Header(xRequestIDHeaderKey, Hash(ctx, includeBody))
		return DefaultGenerator(ctx)
	}
}

// New returns a new request id middleware.
// It optionally accepts an ID Generator.
// The Generator can stop the handlers chain with an error or
// return a valid ID (string).
// If it's nil then the `DefaultGenerator` will be used instead.
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
// A shortcut of `context.GetID().(string)`.
func Get(ctx *context.Context) string {
	v := ctx.GetID()
	if v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}

	return ""
}

// Hash returns the sha1 hash of the request.
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
