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
//   - ContextOptions.BodyMaxMemory, which caps ctx.Body, ctx.ReadJSON,
//     ctx.ReadJSONStream and ctx.ReadMultipartRelated application-wide. It
//     does not cap a form or an upload;
//   - the bodylimit middleware, registered AFTER this one, which stops every
//     reader of the decompressed body at its limit. ctx.ReadForm reports no
//     error for a form cut there and binds nothing of it.
//
// This handler is kept unbounded for backwards compatibility. It logs a warning
// once, on the first compressed request body, unless
// ContextOptions.BodyMaxMemory is set. It cannot see a bodylimit middleware
// registered after it, so that setup gets the warning too.
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
