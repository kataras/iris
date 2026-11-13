package compress

import (
	"net/http"
	"sync"

	"github.com/kataras/iris/v14/context"
)

// ResponseWriter is a compressed data http.ResponseWriter.
type ResponseWriter struct {
	context.ResponseWriter

	Writer
	Encoding string
	Level    int

	// disabled reports that this writer has stepped out of the chain: writes and
	// flushes pass straight through and the compressor is never closed, so no
	// gzip or brotli footer lands in a response that is no longer compressed.
	disabled bool
	// mediaTypeChecked guards the one-time streaming media type check in Write.
	mediaTypeChecked bool
	// written counts the body bytes handed to the compressor. See Written.
	written int
}

// streamingMediaTypes are response media types a compressor must never touch.
// Compression buffers, and a stream that buffers is not a stream: the client
// waits for the compressor's window to fill instead of receiving each event.
//
// The key is the RESPONSE media type, not the request's Accept header, because
// the spec only says a client MAY send "Accept: text/event-stream" and
// Cloudflare rewrites the origin-leg Accept-Encoding regardless. It is compared
// after TrimHeaderValue has removed any parameters, so
// "text/event-stream;charset=utf-8" cannot fall through - the exact bug that
// broke net/http's own ReverseProxy and Caddy (golang/go#47359).
//
// Unexported and never written after init, so reading it on every response is
// not a data race.
var streamingMediaTypes = map[string]struct{}{
	"text/event-stream":    {},
	"application/x-ndjson": {},
}

var compressWritersPool = sync.Pool{New: func() any { return &ResponseWriter{} }}

// AcquireResponseWriter returns a Compress.ResponseWriter from the pool.
// It accepts an Iris response writer, a net/http request value and
// the level of compression (use -1 for default compression level).
//
// It returns the best candidate among "gzip", "deflate", "br", "snappy", "s2" and "zstd"
// based on the request's "Accept-Encoding" header value.
func AcquireResponseWriter(w context.ResponseWriter, r *http.Request, level int) (*ResponseWriter, bool, error) {
	encoding, ok, err := GetAcceptEncoding(r)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	v := compressWritersPool.Get().(*ResponseWriter)
	v.ResponseWriter = w
	v.disabled = false
	v.mediaTypeChecked = false
	v.written = 0

	if level == -1 {
		// Brotli and zstd have no -1 default of their own, so NewWriter maps them
		// to 6 and 3 respectively. Apply the same mapping before the level is
		// recorded, so Level reports the level in use.
		switch encoding {
		case BROTLI:
			level = 6
		case ZSTD:
			level = 3
		}
	}

	encWriter, err := NewWriter(w, encoding, level)
	if err != nil {
		return nil, false, err
	}
	v.Writer = encWriter

	v.Level = level

	v.Encoding = encoding

	setCompressHeaders(w.Header(), encoding)
	return v, true, nil
}

// ReleaseResponseWriter returns the ResponseWriter to the pool.
// It flushes and closes the underline writer.
// It MUST be called after the response is done.
//
// One response is not closed but handed back as it is, with the
// Content-Encoding and Vary taken off: one whose handler wrote nothing
// through w and declared a Content-Length of its own. See Disable.
//
// A caller that put w on a Context with SetResponseWriter puts the previous
// writer back before calling this. The released value has no writer under it
// any more and is reused by another request, and a handler registered outside
// the caller (an access log, a recovery handler) still reads the Context's
// writer after the chain returns:
//
//	previous := ctx.ResponseWriter()
//	if w, ok, _ := compress.AcquireResponseWriter(previous, ctx.Request(), -1); ok {
//		ctx.SetResponseWriter(w)
//		defer func() {
//			ctx.SetResponseWriter(previous)
//			compress.ReleaseResponseWriter(w)
//		}()
//	}
func ReleaseResponseWriter(w *ResponseWriter) {
	if w.declaredLengthUntouched() {
		// The handler declared its own Content-Length and gave the compressor
		// nothing, for example a standard handler relaying an upstream's empty
		// 401 with "Content-Length: 0". Closing the compressor would write an
		// empty compressed stream into that length, net/http would refuse it,
		// and the client would read a body of zero bytes labelled as
		// compressed. Step aside instead: the response is the handler's.
		w.Disable()
	}

	if !w.disabled && !w.IsHijacked() {
		w.Writer.Close() // flushes and closes.
	}
	// A disabled writer must not be closed: Close writes the format's footer, and
	// the response it would land in is no longer compressed. Neither must the
	// writer of a hijacked connection (a websocket upgrade): there is no response
	// left to close, and net/http logs every byte written to it.

	w.Writer = nil
	w.ResponseWriter = nil
	w.Encoding = ""
	w.Level = 0
	w.disabled = false
	w.mediaTypeChecked = false
	w.written = 0

	compressWritersPool.Put(w)
}

// Disable steps this writer out of the chain and returns the writer it wrapped,
// implementing context.ResponseWriterDisabler.
//
// It removes the Content-Encoding and the Vary entry that AcquireResponseWriter
// set optimistically, before the handler had chosen a Content-Type, and it marks
// the writer so the deferred ReleaseResponseWriter cannot close the compressor
// and write a footer into a response that is now uncompressed.
//
// It is idempotent and safe to call before the first byte, which is the only
// time it makes sense: bytes already handed to the compressor cannot be recalled.
func (w *ResponseWriter) Disable() context.ResponseWriter {
	underline := w.ResponseWriter
	if w.disabled || underline == nil {
		return underline
	}

	w.disabled = true

	h := underline.Header()
	h.Del(context.ContentEncodingHeaderKey)
	if h.Get(context.VaryHeaderKey) == context.AcceptEncodingHeaderKey {
		// Only the value this package set. Anything else is the handler's.
		h.Del(context.VaryHeaderKey)
	}

	return underline
}

// declaredLengthUntouched reports whether the response is still untouched,
// nothing was handed to the compressor and the response header carries a
// Content-Length. AcquireResponseWriter removes that header, so one that is
// there now was declared by a handler that ran after it.
func (w *ResponseWriter) declaredLengthUntouched() bool {
	underline := w.ResponseWriter
	if w.disabled || w.written > 0 || underline == nil {
		return false
	}

	return underline.Written() == context.NoWritten &&
		underline.Header().Get(context.ContentLengthHeaderKey) != ""
}

// abandon marks a writer whose response was cut short after compressed bytes
// had left it, so that ReleaseResponseWriter returns it to the pool without
// closing the compressor. It is Disable without the headers: the
// Content-Encoding stays, because the bytes that went out are compressed, and
// no footer is written, because the stream is not complete.
//
// Call it right before ReleaseResponseWriter and nowhere else. A marked writer
// passes writes straight through, like a disabled one.
func (w *ResponseWriter) abandon() {
	w.disabled = true
}

// checkMediaType disables compression the first time it runs if the handler has
// chosen a streaming Content-Type.
//
// This is the safety net for a handler that writes text/event-stream by hand
// instead of going through the sse package, which disables buffering explicitly.
// It runs on the first write or flush because that is the earliest point at
// which the Content-Type is known and the latest at which stepping aside is
// still free.
func (w *ResponseWriter) checkMediaType() {
	if w.mediaTypeChecked {
		return
	}
	w.mediaTypeChecked = true

	if w.ResponseWriter == nil {
		return
	}

	mediaType := context.TrimHeaderValue(w.ResponseWriter.Header().Get(context.ContentTypeHeaderKey))
	if _, isStreaming := streamingMediaTypes[mediaType]; isStreaming {
		w.Disable()
	}
}

// Write writes the compressed data to the underline response writer.
func (w *ResponseWriter) Write(p []byte) (int, error) {
	w.checkMediaType()

	if w.disabled {
		return w.ResponseWriter.Write(p)
	}

	n, err := w.Writer.Write(p)
	w.written += n
	return n, err
}

// Written reports how many body bytes the handlers have written through this
// writer, counted before compression.
//
// The count is kept here and not read from the writer underneath because a
// compressor buffers. Deflate, brotli, snappy, s2 and zstd hand nothing over
// until their window fills or they are closed, so after a handler wrote a
// short body the writer underneath still reports NoWritten. Code that asks
// "did a body already go out?" (rest/errors.Handler, Context.StopWithJSON,
// the router's default error handler) then answered no and wrote a second
// body after the first.
//
// Before the first write it reports what the writer underneath does, so
// NoWritten and StatusCodeWritten keep their meaning. So does a disabled
// writer, whose writes pass straight through.
func (w *ResponseWriter) Written() int {
	if w.disabled || w.written <= 0 {
		return w.ResponseWriter.Written()
	}

	return w.written
}

// Flush implements the http.Flusher interface.
func (w *ResponseWriter) Flush() {
	_ = w.FlushError()
}

// FlushError drains the compressor and then the transport, and reports the first
// failure, implementing context.ResponseWriterFlushError.
//
// The two stages are both required and their order matters: bytes sitting in the
// compressor's window have not reached the connection yet, and flushing the
// connection without draining the compressor first sends nothing.
func (w *ResponseWriter) FlushError() error {
	w.checkMediaType()

	if !w.disabled {
		if err := w.Writer.Flush(); err != nil {
			return err
		}
	}

	return context.FlushResponseWriter(w.ResponseWriter)
}

func setCompressHeaders(h http.Header, encoding string) {
	context.SetCompressHeaders(h, encoding)
}
