package requestid

import (
	"github.com/kataras/iris/v12/context"

	"github.com/google/uuid"
)

func init() {
	context.SetHandlerName("iris/middleware/requestid.*", "iris.request.id")
}

const xRequestIDHeaderValue = "X-Request-Id"

// Generator defines the function which should extract or generate
// a Request ID. See `DefaultGenerator` and `New` package-level functions.
type Generator func(ctx context.Context) string

// DefaultGenerator is the default `Generator` that is used
// when nil is passed on `New` package-level function.
// It extracts the ID from the "X-Request-ID" request header value
// or, if missing, it generates a new UUID(v4) and sets the header and context value.
//
// See `Get` package-level function too.
var DefaultGenerator Generator = func(ctx context.Context) string {
	id := ctx.GetHeader(xRequestIDHeaderValue)

	if id == "" {
		uid, err := uuid.NewRandom()
		if err != nil {
			ctx.StopWithStatus(500)
			return ""
		}

		id = uid.String()
		ctx.Header(xRequestIDHeaderValue, id)
	}

	return id
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

	return func(ctx context.Context) {
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
func Get(ctx context.Context) string {
	v := ctx.GetID()
	if v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}

	return ""
}
