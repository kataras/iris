package iris_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
)

// The CORS rejection is an Errors-mapped error like any other, so the options
// registered through Errors must already be in the context when it is written:
// the collector, the renderer and the id generator apply to it too.
func TestBuilderErrorsOptionsReachTheCorsRejection(t *testing.T) {
	var collected []string

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("https://a.com").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions().
			SetCollector(func(ctx iris.Context, err *errors.Error) {
				collected = append(collected, string(err.ErrorCode.CanonicalName))
			})).
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/ping", func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
		}).
		Build()

	e := httptest.New(t, app)
	e.GET("/ping").WithHeader("Origin", "https://a.com").Expect().
		Status(httptest.StatusOK).Body().IsEqual("pong")

	if len(collected) != 0 {
		t.Fatalf("an allowed origin must not report an error, got %v", collected)
	}

	e.GET("/ping").WithHeader("Origin", "https://evil.com").Expect().
		Status(httptest.StatusBadRequest)

	if len(collected) != 1 || collected[0] != "FAILED_PRECONDITION" {
		t.Fatalf("expected the CORS rejection to reach the Errors collector, got %v", collected)
	}
}

// Under NoPrefix the API middlewares belong to the API's routes, not to the
// 404 and 405 chains: a rejecting middleware must not turn every unknown path
// into its own status code.
func TestBuilderMiddlewaresDoNotRunOnNoRoute(t *testing.T) {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares(func(ctx iris.Context) {
			ctx.StopWithStatus(httptest.StatusUnauthorized)
		}).
		API("/", func(r iris.Router) {
			r.Get("/ping", func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
		}).
		Build()

	e := httptest.New(t, app)
	e.GET("/ping").Expect().Status(httptest.StatusUnauthorized)
	e.GET("/unknown").Expect().Status(httptest.StatusNotFound)
}

type unmarshalable struct {
	Ch chan int `json:"ch"`
}

// A child Router copies the context options when it is created, so the
// rest/errors context error handler has to be in place before Prefix runs:
// otherwise a prefixed API falls back to the framework's default handler,
// which leaves the client a bare 500 with no body, while a NoPrefix one
// answers with the rest/errors envelope.
func TestBuilderPrefixedRouterUsesTheErrorsContextErrorHandler(t *testing.T) {
	boom := func(r iris.Router) {
		r.Get("/boom", func(ctx iris.Context) {
			ctx.JSON(unmarshalable{})
		})
	}

	prefixed := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", boom).
		Build()

	e := httptest.New(t, prefixed)
	e.GET("/api/boom").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().Value("http_error_code").Object().
		Value("canonical_name").IsEqual("INTERNAL")
}

// gzipBytes returns "n" zero bytes, gzip compressed.
func gzipBytes(t *testing.T, n int64) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := io.CopyN(zw, zeroReader{}, n); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type zeroReader struct{}

func (zeroReader) Read(b []byte) (int, error) {
	clear(b)
	return len(b), nil
}

// Compression(true) must install the bounded decompression handler: a gzip
// bomb expands to gigabytes through the unbounded one.
func TestBuilderCompressionBoundsTheRequestBody(t *testing.T) {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(true).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Post("/upload", func(ctx iris.Context) {
				body, err := ctx.Body()
				if err != nil {
					ctx.StopWithStatus(httptest.StatusRequestEntityTooLarge)
					return
				}
				ctx.Writef("%d", len(body))
			})
		}).
		Build()

	e := httptest.New(t, app)

	e.POST("/upload").WithHeader("Content-Encoding", "gzip").
		WithBytes(gzipBytes(t, 1024)).Expect().
		Status(httptest.StatusOK).Body().IsEqual("1024")

	e.POST("/upload").WithHeader("Content-Encoding", "gzip").
		WithBytes(gzipBytes(t, compress.DefaultMaxDecompressedSize+1)).Expect().
		Status(httptest.StatusRequestEntityTooLarge)
}

// The access log is registered before recovery, so it is still on the stack
// when recovery turns a panic into a 500 and returns normally. Registered the
// other way round, accesslog.Handler never resumes after ctx.Next and a
// panicking handler is logged nowhere.
func TestBuilderAccessLogRecordsAPanickingHandler(t *testing.T) {
	var buf bytes.Buffer
	ac := accesslog.New(&buf)

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequestsWith(ac).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/boom", func(ctx iris.Context) {
				panic("boom")
			})
		}).
		Build()

	e := httptest.New(t, app)
	e.GET("/boom").Expect().Status(httptest.StatusInternalServerError)

	if got := buf.String(); !strings.Contains(got, "/boom") {
		t.Fatalf("expected the access log to record the panicking request, got: %q", got)
	}
}
