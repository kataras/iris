// Package recorder keeps a copy of the request body, the response body, or
// both, so that a later handler can read them again.
//
// It has no settings, so following the plain-handler rule its entry points are
// the `Request` and `Response` functions themselves: register the one whose
// side of the exchange you need to replay.
package recorder

import (
	"bytes"
	"io"
	"net/http"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recorder.Request*", "iris.recorder.request")
}

// Request enables multiple request body reading by replacing
// the request's Body with a buffered one.
// Remember to use this cautiously, as it reads the entire request body into memory,
// which might not be suitable for large request bodies or high-traffic servers.
//
// It should be used as a middleware.
// Usage:
//
//	app.Use(recorder.Request)
//	app.Post("/path", handler)
//
// See `ReadRequestBody` to read the request body
// or `RecordRequestBody` to get the buffer itself.
func Request(ctx *context.Context) {
	RecordRequestBody(ctx)
	// Call the next handler.
	ctx.Next()
}

// ReadRequestBody enables multiple request body reading by replacing
// the request's Body with a buffered one and returns the request body.
// Usage:
//
//	body := ReadRequestBody(ctx)
func ReadRequestBody(ctx *context.Context) []byte {
	return RecordRequestBody(ctx).Bytes()
}

// RecordRequestBody records the request body.
// It should be used when you want to read the request body multiple times.
// Usage:
// buffer := RecordRequestBody(ctx)
func RecordRequestBody(ctx *context.Context) *BufferCloser {
	return acquireRequestBodyRecorder(ctx)
}

// IsRequestBodyRecorded reports whether the request body is buffered.
func IsRequestBodyRecorded(ctx *context.Context) bool {
	_, ok := getRequestBodyRecorder(ctx.Request())
	return ok
}

// BufferCloser is an io.ReadCloser over an in-memory copy of the request body,
// with a no-op Close method. It allows reading from the same reader multiple
// times by buffering the data: each pass that drains it rewinds for the next one.
type BufferCloser struct {
	reader *bytes.Reader
	// cache  []byte
}

// newBufferCloser creates a new BufferCloser that wraps the provided reader.
// It reads the entire content of the reader into a buffer, allowing repeated reads.
func newBufferCloser(requestReader io.Reader) *BufferCloser {
	contents, _ := io.ReadAll(requestReader)
	bc := &BufferCloser{
		reader: bytes.NewReader(bytes.Clone(contents)),
		// cache:  contents,
	}
	return bc
}

// Read reads the next len(p) bytes from the buffer or until the buffer is drained.
// Each full pass ends with io.EOF, so read loops like io.ReadAll terminate;
// the buffer rewinds as soon as a pass drains it, allowing the body to be
// read again. EOF surfaces together with the final bytes, because decoder
// consumers (json.Decoder) never issue the extra read that would return
// (0, io.EOF) on its own.
func (r *BufferCloser) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err == io.EOF || (err == nil && r.reader.Len() == 0) {
		// The pass is complete: surface EOF so the caller's read loop stops,
		// and rewind so the next consumer reads the full content again.
		r.reader.Seek(0, io.SeekStart)
		err = io.EOF
	}

	return n, err
}

// Close implements the io.Closer interface. It does nothing.
func (r *BufferCloser) Close() error { return nil }

// Bytes returns the content of the buffer.
func (r *BufferCloser) Bytes() []byte {
	contents, _ := io.ReadAll(r)
	return contents
}

func acquireRequestBodyRecorder(ctx *context.Context) *BufferCloser {
	r := ctx.Request()

	if buffer, ok := getRequestBodyRecorder(r); ok {
		return buffer
	}

	// Set the new body which can be read multiple times.
	buffer := newBufferCloser(r.Body) // can be pool-ed.
	r.Body = buffer
	// Return the body.
	return buffer
}

func getRequestBodyRecorder(r *http.Request) (*BufferCloser, bool) {
	buf, ok := r.Body.(*BufferCloser)
	return buf, ok
	// return buf.Bytes(), ok
}
