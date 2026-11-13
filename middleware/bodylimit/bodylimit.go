// Package bodylimit caps the size of a request body.
//
// Its entry point is New(limitOverBytes int64), which returns a Handler. The
// single setting is the limit itself, so it is a plain argument and there is no
// Options struct.
package bodylimit

import (
	"fmt"
	"io"
	"net/http"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/bodylimit.*", "iris.bodylimit")
}

// New returns a new request body limitter middleware.
// Usage:
//
//	import "github.com/kataras/iris/v14/middleware/bodylimit"
//	[...]
//	app.Use(bodylimit.New(2 * iris.MB))
func New(limitOverBytes int64) context.Handler {
	return func(ctx *context.Context) {
		r := ctx.Request()

		if r.ContentLength > limitOverBytes {
			// The error Context.Body returns when the Reader below finds the
			// same thing while reading: both sentinels, in that order. It is
			// context.ErrRequestEntityTooLarge that rest/errors maps to 413
			// TOO_LARGE with "request body too large", so a request that
			// declares its length and one that does not read the same answer.
			ctx.StopWithPlainError(http.StatusRequestEntityTooLarge,
				fmt.Errorf("%w: %w (content-length: %d, content-type: %s)", context.ErrReadRequestBody,
					context.ErrRequestEntityTooLarge, ctx.ContentLength(), ctx.ContentType().String()))
			return
		}

		// Note(@kataras): the Reader is NOT pooled.
		//
		// It used to be taken from a sync.Pool, installed as r.Body, and put back
		// as soon as ctx.Next returned. But net/http drains and closes r.Body
		// AFTER the handler returns, and anything that reads the body
		// asynchronously (accesslog's Async mode, a goroutine spawned by a
		// handler) outlives it by even longer. By then another in-flight request
		// may have taken the same *Reader out of the pool and Reset it to its own
		// body - so the server's drain read from, and closed, a different
		// connection's request body. That is cross-request body corruption plus a
		// data race, to save one three-word allocation per request.
		r.Body = &Reader{reader: r.Body, limit: limitOverBytes}
		ctx.Next()
	}
}

// IsBodyLimited reports whether the request body size is limited.
func IsBodyLimited(ctx *context.Context) bool {
	_, ok := ctx.Request().Body.(*Reader)
	return ok
}

// Reader is a reader that reads from an io.ReadCloser and stops reading when the limit is reached.
// It is used to limit the request body size.
type Reader struct {
	reader io.ReadCloser
	read   int64
	limit  int64
}

// Read reads from the underlying reader and stops reading when the limit is reached.
//
// It returns the bytes up to the limit and not one past it. For a body that
// goes on, every Read from there returns context.ErrRequestEntityTooLarge
// and no data.
func (r *Reader) Read(b []byte) (n int, err error) {
	if r.read > r.limit {
		return 0, context.ErrRequestEntityTooLarge
	}

	// Ask for one byte past the limit at most, like http.MaxBytesReader: it
	// is what tells a body of exactly the limit from a longer one.
	if remaining := r.limit - r.read; int64(len(b))-1 > remaining {
		b = b[:remaining+1]
	}

	n, err = r.reader.Read(b)
	r.read += int64(n)
	if r.read <= r.limit {
		return n, err
	}

	// That byte was read and is not returned, and the error waits for a Read
	// that has nothing else to return. It used to travel with the data, the
	// bytes past the limit included, and a decoder that keeps the data of
	// such a Read and drops its error (encoding/json/v2 does) decoded a body
	// of any size when its declared length did not give it away: a chunked
	// one, or a compressed one behind the compress middleware.
	if n--; n > 0 {
		return n, nil
	}

	return 0, context.ErrRequestEntityTooLarge
}

// Close closes the underlying reader.
func (r *Reader) Close() error {
	return r.reader.Close()
}

// Reset resets the reader with a new underline io.Reader.
func (r *Reader) Reset(reader io.ReadCloser) {
	r.reader = reader
	r.read = 0
}
