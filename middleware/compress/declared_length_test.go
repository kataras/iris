package compress_test

import (
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"
)

// A handler that declares its own Content-Length and gives the compressor
// nothing has chosen its body. A standard handler relaying an upstream's empty
// 401 does exactly that with "Content-Length: 0". Closing the compressor
// writes an empty compressed stream, which does not fit a declared length of
// zero: over a real connection net/http refuses it and the client reads
// nothing at all under a Content-Encoding header. The middleware steps aside
// for that response instead.
func TestDeclaredContentLengthWithNoBodyIsNotCompressed(t *testing.T) {
	for _, encoding := range compress.AllEncodings {
		t.Run(encoding, func(t *testing.T) {
			app := iris.New()
			app.UseRouter(compress.Safe)
			app.Get("/declared", iris.FromStd(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusUnauthorized)
			}))
			app.Get("/bare", func(ctx iris.Context) {
				ctx.StatusCode(iris.StatusUnauthorized)
			})

			e := httptest.New(t, app)

			res := e.GET("/declared").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusUnauthorized)
			res.Body().IsEmpty()

			header := res.Raw().Header
			if got := header.Get("Content-Encoding"); got != "" {
				t.Fatalf("an empty body with a declared length is not compressed, but the response says Content-Encoding: %s", got)
			}

			if got := header.Get("Vary"); got != "" {
				t.Fatalf("expected the Vary the middleware set to be taken back with the encoding, got %q", got)
			}

			if got := header.Get("Content-Length"); got != "0" {
				t.Fatalf("expected the Content-Length: 0 the handler declared, got %q", got)
			}

			// Nothing declared: the response is the middleware's to encode, as before.
			e.GET("/bare").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusUnauthorized).HasContentEncoding(encoding)
		})
	}
}
