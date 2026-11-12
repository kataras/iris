package compress

import (
	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/compress.*", "iris.compression")
}

// Handler retruns a new Handler which is responsible to decompress the request's body
// based on the "Content-Encoding" header and compress the response's body based on
// the client's "Accept-Encoding" header.
//
// SECURITY WARNING: this handler places NO ceiling on the decompressed size of a
// request body.
//
// A ~1MB "Content-Encoding: gzip" body of zeros expands to roughly 1GB, and any
// handler that calls ctx.Body or ctx.ReadJSON will allocate all of it. A handful
// of concurrent requests exhausts memory. Use one of:
//
//   - compress.Safe, or compress.Limited(n), which cap the decompressed stream;
//   - ContextOptions.BodyMaxMemory, which caps every request body application-wide;
//   - the bodylimit middleware, registered AFTER this one.
//
// This handler is kept unbounded for backwards compatibility and logs a warning
// once at first use when nothing else bounds the body.
//
// Usage:
//
//	app.Use(compress.Handler)
//
// OR
//
//	app.UseRouter(compress.Handler)
//
// Prefer:
//
//	app.Use(compress.Safe)
func Handler(ctx *context.Context) {
	handle(ctx, 0)
}
