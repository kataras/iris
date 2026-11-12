package compress_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
)

// Written is how a handler asks "did a body already go out?". A compressor
// buffers, so the count cannot come from the writer under it: all but gzip
// hand nothing over until their window fills or they are closed, and the
// writer underneath still reports that nothing was written.
func TestResponseWriterWrittenCountsBufferedBytes(t *testing.T) {
	for _, encoding := range compress.AllEncodings {
		t.Run(encoding, func(t *testing.T) {
			var beforeWrite, afterWrite int

			app := iris.New()
			app.Use(compress.Safe)
			app.Get("/", func(ctx iris.Context) {
				beforeWrite = ctx.ResponseWriter().Written()
				ctx.WriteString("hello")
				afterWrite = ctx.ResponseWriter().Written()
			})

			e := httptest.New(t, app)
			e.GET("/").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusOK).HasContentEncoding(encoding)

			if beforeWrite > 0 {
				t.Fatalf("expected no written bytes before the first write, got %d", beforeWrite)
			}

			if afterWrite != len("hello") {
				t.Fatalf("expected Written to report the %d bytes the handler wrote, got %d", len("hello"), afterWrite)
			}
		})
	}
}

// rest/errors.Handler renders a failure only when nothing was written. Under an
// encoding whose compressor had not flushed yet it used to see an untouched
// writer and append a second error document to the one the route rendered.
func TestErrorsHandlerRendersOnceUnderEveryEncoding(t *testing.T) {
	for _, encoding := range compress.AllEncodings {
		t.Run(encoding, func(t *testing.T) {
			app := iris.New()
			app.Use(compress.Safe)
			app.Get("/", errors.Handler, func(ctx iris.Context) {
				errors.NotFound.Detail(ctx, "plan not found", "no plan has this id")
			})

			e := httptest.New(t, app)
			res := e.GET("/").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusNotFound).HasContentEncoding(encoding).Raw()

			r, err := compress.NewReader(res.Body, encoding)
			if err != nil {
				t.Fatal(err)
			}

			body, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}

			dec := json.NewDecoder(bytes.NewReader(body))

			var doc map[string]any
			if err := dec.Decode(&doc); err != nil {
				t.Fatalf("the body is not a JSON object: %v: %s", err, body)
			}

			if doc["message"] != "plan not found" {
				t.Fatalf("expected the route's own error, got: %s", body)
			}

			var extra any
			if err := dec.Decode(&extra); err != io.EOF {
				t.Fatalf("expected one JSON document, got more: %s", body)
			}
		})
	}
}
