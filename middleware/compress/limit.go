package compress

import (
	"errors"
	"io"
	"sync"

	"github.com/kataras/iris/v14/context"
)

// ErrDecompressedBodyTooLarge is returned to a reader of the request body when
// the decompressed size exceeds the configured limit.
var ErrDecompressedBodyTooLarge = errors.New("compress: decompressed request body too large")

// DefaultMaxDecompressedSize is the ceiling applied by `Safe` and by
// `Limited`'s zero value: 32MB, matching ContextOptions.PostMaxMemory.
const DefaultMaxDecompressedSize int64 = 32 << 20

// Limited returns a compression handler that refuses to decompress a request
// body beyond "maxDecompressedSize" bytes.
//
// This is the mitigation for a decompression bomb: a ~1MB gzip payload of zeros
// expands to roughly 1GB, and `Handler` hands the expanded stream to whatever
// reads ctx.Body()/ctx.ReadJSON with no ceiling at all. A handful of concurrent
// requests is enough to exhaust memory.
//
// Pass a value <= 0 to use DefaultMaxDecompressedSize.
//
// Usage:
//
//	app.Use(compress.Limited(8 << 20)) // 8MB decompressed
//
// The two other ways to bound it, both of which cap the DECOMPRESSED stream
// correctly because they wrap the body after this middleware replaced it:
//
//   - iris.WithContextOptions(func(o *iris.ContextOptions) { o.BodyMaxMemory = ... })
//   - the bodylimit middleware, registered AFTER this one
func Limited(maxDecompressedSize int64) context.Handler {
	if maxDecompressedSize <= 0 {
		maxDecompressedSize = DefaultMaxDecompressedSize
	}

	return func(ctx *context.Context) {
		handle(ctx, maxDecompressedSize)
	}
}

// Safe is Limited with DefaultMaxDecompressedSize.
//
// Prefer it over `Handler` for anything that accepts compressed request bodies
// from untrusted clients.
func Safe(ctx *context.Context) {
	handle(ctx, DefaultMaxDecompressedSize)
}

// limitedReadCloser stops after "remaining" bytes and reports
// ErrDecompressedBodyTooLarge rather than truncating silently.
type limitedReadCloser struct {
	io.ReadCloser

	remaining int64
}

func (r *limitedReadCloser) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, ErrDecompressedBodyTooLarge
	}

	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}

	n, err := r.ReadCloser.Read(p)
	r.remaining -= int64(n)

	if r.remaining <= 0 && err == nil {
		// Peek one more byte: if anything follows, the body genuinely exceeds the
		// limit rather than ending exactly on it.
		var probe [1]byte
		if extra, _ := r.ReadCloser.Read(probe[:]); extra > 0 {
			return n, ErrDecompressedBodyTooLarge
		}
	}

	return n, err
}

// warnUnboundedOnce keeps the startup warning to one line per process.
var warnUnboundedOnce sync.Once

// warnUnbounded logs once when compression is used with no decompression ceiling.
// It writes to the log only; no request is treated differently because of it.
func warnUnbounded(ctx *context.Context) {
	if ctx.OptionsReadOnly().BodyMaxMemory > 0 {
		return // already bounded application-wide.
	}

	warnUnboundedOnce.Do(func() {
		ctx.Application().Logger().Warn(
			"compress: request decompression is UNBOUNDED. A ~1MB gzip body can expand to ~1GB in memory. " +
				"Use compress.Safe / compress.Limited(n), or set ContextOptions.BodyMaxMemory.")
	})
}

func handle(ctx *context.Context, maxDecompressedSize int64) {
	// Setup reader.
	r := ctx.Request()
	if reader, ok, err := NewRequestReader(r); !ok { // request has no encoding or not supported.
		if err != nil {
			ctx.StopWithPlainError(415, err) // server missing the decompression algorithm.
			return
		}
	} else {
		defer reader.Close()

		if maxDecompressedSize > 0 {
			r.Body = &limitedReadCloser{ReadCloser: reader, remaining: maxDecompressedSize}
		} else {
			warnUnbounded(ctx)
			r.Body = reader
		}
	}

	// Setup writer.
	w := ctx.ResponseWriter()
	if writer, ok, err := AcquireResponseWriter(w, r, -1); !ok { // client does not accept encoding or not supported.
		if err != nil {
			ctx.StopWithPlainError(415, err) // server missing the compression algorithm.
			return
		}
	} else { // err is nil.
		// What the writer underneath reports before the compressor is given
		// anything. A different answer later means the compressor let something
		// through: the status line, or compressed bytes.
		writtenBefore := w.Written()

		defer func() {
			// Put back the writer that was on the Context before this handler
			// ran, on the normal path as on the panic one. The compression
			// writer is closed and returned to its pool just below, and whatever
			// is registered outside this handler (the access log, a recovery
			// handler) reads the Context's writer after the chain returns. Left
			// in place it is a pooled object with no writer under it, or one
			// already handed to another request.
			//
			// "w" and not writer.ResponseWriter: ReleaseResponseWriter sets that
			// field to nil.
			ctx.SetResponseWriter(w)

			if rec := recover(); rec != nil {
				if w.Written() != writtenBefore {
					// Compressed bytes already left the compressor (gzip sends
					// its header on the first write, any encoder sends bytes on
					// a flush or once the body outgrows its buffer). They are
					// what the client reads, on the wire or in a recorder that
					// has yet to send them, so the Content-Encoding stays: a
					// stream that ends early is something a client detects,
					// and compressed bytes labelled as plain text are not.
					// The writer is only marked, so the release below returns
					// it to the pool without closing the compressor: a footer
					// would make the cut stream look complete.
					writer.abandon()
				} else {
					// Whatever answers the panic is registered outside this
					// handler, so it writes outside compression: plain bytes.
					// Disable takes back the Content-Encoding and Vary this
					// middleware announced before the handler ran, or the
					// client is told to decompress a body that is not
					// compressed. It also marks the writer, so the release
					// below returns it to the pool without closing the
					// compressor: no footer is written, and what the handler
					// had handed to the compressor is dropped with it.
					writer.Disable()
				}

				ReleaseResponseWriter(writer)

				panic(rec) // fallthrough to any recovery handler/middleware.
			}

			ReleaseResponseWriter(writer)
		}()

		ctx.SetResponseWriter(writer)
	}

	ctx.Next()
}
