package compress_test

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"
)

// gzipBomb returns a gzip payload that decompresses to "size" zero bytes.
// Zeros compress to roughly nothing, which is the whole point of the attack.
func gzipBomb(t *testing.T, size int) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestCompressLimitedRejectsDecompressionBomb(t *testing.T) {
	const limit = 64 << 10 // 64KB

	app := iris.New()
	app.Use(compress.Limited(limit))
	app.Post("/", func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusRequestEntityTooLarge, err)
			return
		}
		ctx.Writef("%d", len(body))
	})

	e := httptest.New(t, app)

	// Well under the limit: read in full.
	small := gzipBomb(t, 1024)
	e.POST("/").WithHeader("Content-Encoding", "gzip").WithBytes(small).
		Expect().Status(httptest.StatusOK).Body().IsEqual("1024")

	// Far over the limit: the read must fail rather than allocate 8MB.
	bomb := gzipBomb(t, 8<<20)
	e.POST("/").WithHeader("Content-Encoding", "gzip").WithBytes(bomb).
		Expect().Status(httptest.StatusRequestEntityTooLarge)
}

func TestCompressSafeRejectsDecompressionBomb(t *testing.T) {
	app := iris.New()
	app.Use(compress.Safe)
	app.Post("/", func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusRequestEntityTooLarge, err)
			return
		}
		ctx.Writef("%d", len(body))
	})

	e := httptest.New(t, app)

	// DefaultMaxDecompressedSize is 32MB; 64MB of zeros must be refused.
	bomb := gzipBomb(t, 64<<20)
	e.POST("/").WithHeader("Content-Encoding", "gzip").WithBytes(bomb).
		Expect().Status(httptest.StatusRequestEntityTooLarge)
}

// The unbounded Handler is kept for backwards compatibility. This pins that it
// still decompresses without a ceiling, so the behaviour is known and
// deliberate rather than accidentally re-broken.
func TestCompressHandlerRemainsUnbounded(t *testing.T) {
	app := iris.New()
	app.Use(compress.Handler)
	app.Post("/", func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusRequestEntityTooLarge, err)
			return
		}
		ctx.Writef("%d", len(body))
	})

	e := httptest.New(t, app)

	const size = 2 << 20
	e.POST("/").WithHeader("Content-Encoding", "gzip").WithBytes(gzipBomb(t, size)).
		Expect().Status(httptest.StatusOK).Body().IsEqual("2097152")
}

// ContextOptions.BodyMaxMemory bounds the DECOMPRESSED stream, because the
// compression middleware replaced r.Body before the limit is applied.
func TestCompressBodyMaxMemoryBoundsDecompressedStream(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.BodyMaxMemory = 64 << 10
	}))
	app.Use(compress.Handler)
	app.Post("/", func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusRequestEntityTooLarge, err)
			return
		}
		ctx.Writef("%d", len(body))
	})

	e := httptest.New(t, app)

	e.POST("/").WithHeader("Content-Encoding", "gzip").WithBytes(gzipBomb(t, 8<<20)).
		Expect().Status(httptest.StatusRequestEntityTooLarge)
}
