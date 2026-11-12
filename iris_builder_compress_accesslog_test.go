package iris_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/servertiming"
)

// gunzipString decompresses a gzip response body read by the test client.
func gunzipString(t *testing.T, body io.Reader) string {
	t.Helper()

	zr, err := gzip.NewReader(body)
	if err != nil {
		t.Fatalf("the response body is not gzip: %v", err)
	}
	defer zr.Close()

	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("reading the gzip response body: %v", err)
	}

	return string(b)
}

// The builder registers the access log outside compression. The access log
// reads the status code from the Context's response writer after the chain
// returns, so the compression middleware must not leave its own writer, already
// closed and back in its pool, on the Context when it is done.
//
// The client has to send Accept-Encoding: without it no compression writer is
// acquired and nothing is swapped.
func TestBuilderCompressionWithAccessLog(t *testing.T) {
	var buf bytes.Buffer
	ac := accesslog.New(&buf)

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(true).
		LogRequestsWith(ac).
		NoHealth().
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

	res := e.GET("/ping").WithHeader("Accept-Encoding", "gzip").Expect().
		Status(httptest.StatusOK).HasContentEncoding("gzip").Raw()

	if got := gunzipString(t, res.Body); got != "pong" {
		t.Fatalf("expected the body to gunzip to %q, got %q", "pong", got)
	}

	logged := strings.TrimSpace(buf.String())
	if lines := strings.Split(logged, "\n"); len(lines) != 1 || logged == "" {
		t.Fatalf("expected the access log to record one line, got %d: %q", len(lines), logged)
	}

	// The default line is Time|Latency|Code|Method|Path|...
	if !strings.Contains(logged, "|200|GET|/ping|") {
		t.Fatalf("expected the access log line to carry status 200 for GET /ping, got: %q", logged)
	}
}

// The same combination under load, for the race detector. The compression
// writer and the response recorder are pooled, so a writer left on a Context
// after its release is shared with whichever request picks it up next. Every
// layer that swaps the writer is in the chain: the access log records the
// response body (a recorder outside compression), compression, and a route
// middleware with a recorder of its own inside it.
func TestBuilderCompressionWithAccessLogConcurrent(t *testing.T) {
	const (
		clients  = 8
		requests = 40
	)

	var buf bytes.Buffer
	ac := accesslog.New(&buf)
	ac.ResponseBody = true

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(true).
		LogRequestsWith(ac).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/ping", servertiming.Handler, func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
			r.Get("/forbidden", servertiming.Handler, func(ctx iris.Context) {
				ctx.StopWithStatus(iris.StatusForbidden)
			})
		}).
		Build()
	app.Logger().SetLevel("disable")

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	get := func(path string) (int, string, error) {
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			return 0, "", err
		}
		// Set by hand, so the transport leaves the body compressed.
		req.Header.Set("Accept-Encoding", "gzip")

		res, err := srv.Client().Do(req)
		if err != nil {
			return 0, "", err
		}
		defer res.Body.Close()

		zr, err := gzip.NewReader(res.Body)
		if err != nil {
			return res.StatusCode, "", err
		}

		body, err := io.ReadAll(zr)
		return res.StatusCode, string(body), err
	}

	var (
		wg       sync.WaitGroup
		failures = make(chan string, clients*requests)
	)

	for range clients {
		wg.Go(func() {
			for i := range requests {
				if i%2 == 0 {
					status, body, err := get("/ping")
					if err != nil || status != http.StatusOK || body != "pong" {
						failures <- "GET /ping: " + http.StatusText(status) + " " + body
					}

					continue
				}

				status, body, err := get("/forbidden")
				if err != nil || status != http.StatusForbidden || !strings.Contains(body, `"PERMISSION_DENIED"`) {
					failures <- "GET /forbidden: " + http.StatusText(status) + " " + body
				}
			}
		})
	}

	wg.Wait()
	close(failures)

	for failure := range failures {
		t.Error(failure)
	}

	// Close waits for the lines still being written.
	if err := ac.Close(); err != nil {
		t.Fatal(err)
	}

	logged := buf.String()
	if got, expected := strings.Count(logged, "|200|GET|/ping|"), clients*requests/2; got != expected {
		t.Fatalf("expected %d access log lines with status 200 for /ping, got %d", expected, got)
	}

	if got, expected := strings.Count(logged, "|403|GET|/forbidden|"), clients*requests/2; got != expected {
		t.Fatalf("expected %d access log lines with status 403 for /forbidden, got %d", expected, got)
	}
}

// A panic under compression is answered by the recovery handler, which sits
// outside compression and writes plain JSON. The Content-Encoding the
// compression middleware announced before the handler ran must be gone by
// then, or the client is told to gunzip a body that is not gzip.
func TestBuilderPanicUnderCompression(t *testing.T) {
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
			r.Get("/boom", func(ctx iris.Context) {
				panic("boom")
			})
			r.Get("/half", func(ctx iris.Context) {
				ctx.WriteString("half an answer")
				panic("boom")
			})
			r.Get("/ping", func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
		}).
		Build()

	e := httptest.New(t, app)

	// A panic before the first byte under gzip, and a panic after a short write
	// under brotli, which still holds those bytes when the panic comes: they
	// are dropped with the compressor and the recovery answer goes out clean.
	//
	// The same short write under gzip is not here because nothing can save it:
	// gzip sends its 10-byte header on the first write, so the status line and
	// the Content-Encoding are on the wire before the panic, as they would be
	// after any partial body.
	for _, request := range []struct{ path, encoding string }{
		{"/boom", "gzip"},
		{"/boom", "br"},
		{"/half", "br"},
	} {
		res := e.GET(request.path).WithHeader("Accept-Encoding", request.encoding).Expect().
			Status(httptest.StatusInternalServerError)

		header := res.Raw().Header
		if encoding := header.Get("Content-Encoding"); encoding != "" {
			t.Fatalf("%s (%s): the recovery answer is not compressed but announces Content-Encoding: %s",
				request.path, request.encoding, encoding)
		}

		if vary := header.Get("Vary"); vary != "" {
			t.Fatalf("%s (%s): expected the Vary the compression middleware set to be taken back, got %q",
				request.path, request.encoding, vary)
		}

		body := res.Body().Raw()
		if strings.Contains(body, "half an answer") {
			t.Fatalf("%s (%s): the abandoned body went out with the recovery answer: %q",
				request.path, request.encoding, body)
		}

		bareEnvelope(t, []byte(body), httptest.StatusInternalServerError, "INTERNAL")
	}

	// The writer the panic interrupted went back to its pool: the next request
	// that gets it must compress as usual.
	for range 4 {
		res := e.GET("/ping").WithHeader("Accept-Encoding", "gzip").Expect().
			Status(httptest.StatusOK).HasContentEncoding("gzip").Raw()

		if got := gunzipString(t, res.Body); got != "pong" {
			t.Fatalf("expected the body to gunzip to %q, got %q", "pong", got)
		}
	}
}

// The other side of the panic: compressed bytes that left the compressor
// before it. gzip sends its 10-byte header on the first write, and under an
// access log that records the response body those bytes sit in the recorder,
// so nothing is on the wire yet and the header map is still what the client
// will read. Taking Content-Encoding back at that point labels gzip bytes as
// plain text: the client read a 200 whose body was ten bytes of binary. Left
// alone, it reads a gzip stream that ends early, which it can detect, as it
// would without the recorder.
func TestBuilderPanicAfterCompressedBytesKeepsTheEncoding(t *testing.T) {
	var buf bytes.Buffer
	ac := accesslog.New(&buf)
	ac.ResponseBody = true

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(true).
		LogRequestsWith(ac).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/half", func(ctx iris.Context) {
				ctx.WriteString("half an answer")
				panic("boom")
			})
			r.Get("/ping", func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
		}).
		Build()
	app.Logger().SetLevel("disable")

	e := httptest.New(t, app)

	res := e.GET("/half").WithHeader("Accept-Encoding", "gzip").Expect().Raw()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if len(body) == 0 {
		t.Fatal("expected the bytes gzip wrote before the panic to have reached the client")
	}

	if encoding := res.Header.Get("Content-Encoding"); encoding != "gzip" {
		t.Fatalf("%d gzip bytes reached the client (%q) with status %d, but the response says Content-Encoding: %q",
			len(body), body, res.StatusCode, encoding)
	}

	// No footer was written after the panic: the stream ends early, and that
	// is how the client learns the answer is not complete.
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err == nil {
		_, err = io.ReadAll(zr)
	}

	if err == nil {
		t.Fatalf("expected a gzip stream cut short by the panic, got a complete one: %q", body)
	}

	// The writer the panic interrupted went back to its pool without being
	// closed: the next request that gets it must compress as usual.
	for range 4 {
		res := e.GET("/ping").WithHeader("Accept-Encoding", "gzip").Expect().
			Status(httptest.StatusOK).HasContentEncoding("gzip").Raw()

		if got := gunzipString(t, res.Body); got != "pong" {
			t.Fatalf("expected the body to gunzip to %q, got %q", "pong", got)
		}
	}
}

// The access log sits outside compression, so what its recorder holds for a
// compressed response is the compressed bytes. They are not the response body
// a person reading the log wants, and a Redact function cannot match anything
// in them, so the body is left out of that line. Everything else is logged.
func TestBuilderAccessLogOmitsACompressedBody(t *testing.T) {
	const (
		answer     = "the secret is swordfish"
		gzipHeader = "\x1f\x8b"
	)

	var (
		buf      bytes.Buffer
		redacted []string
	)

	ac := accesslog.New(&buf)
	ac.ResponseBody = true
	ac.Redact = func(ctx iris.Context, body string) string {
		redacted = append(redacted, body)
		return strings.ReplaceAll(body, "swordfish", "[redacted]")
	}

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(true).
		LogRequestsWith(ac).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/secret", func(ctx iris.Context) {
				ctx.WriteString(answer)
			})
		}).
		Build()

	e := httptest.New(t, app)

	// fields splits the one logged line of the default format. With no path
	// parameter and no query the request values column is not written:
	// Time|Latency|Code|Method|Path|IP|Bytes Received|Bytes Sent|Request|Response|
	const (
		codeField, methodField, pathField = 2, 3, 4
		bytesSentField, responseField     = 7, 9
	)

	fields := func(t *testing.T) []string {
		t.Helper()

		line := strings.TrimSuffix(buf.String(), "\n")
		buf.Reset()

		parts := strings.Split(line, "|")
		if len(parts) != 11 {
			t.Fatalf("expected one access log line of 10 fields, got %d: %q", len(parts)-1, line)
		}

		return parts
	}

	// Compressed: the body is left out, the rest of the line is there.
	raw, err := io.ReadAll(e.GET("/secret").WithHeader("Accept-Encoding", "gzip").Expect().
		Status(httptest.StatusOK).HasContentEncoding("gzip").Raw().Body)
	if err != nil {
		t.Fatal(err)
	}

	line := fields(t)
	if code, method, path := line[codeField], line[methodField], line[pathField]; code != "200" || method != "GET" || path != "/secret" {
		t.Fatalf("unexpected status, method or path in the access log line: %q", line)
	}

	if got, expected := line[bytesSentField], fmt.Sprintf("%d B", len(raw)); got != expected {
		t.Fatalf("expected bytes sent to be the %s that went out, got %s", expected, got)
	}

	if response := line[responseField]; response != "" {
		t.Fatalf("expected no response body for a compressed response, got %q", response)
	}

	for _, body := range redacted {
		if strings.HasPrefix(body, gzipHeader) {
			t.Fatalf("Redact was called with compressed bytes: %q", body)
		}
	}

	// Not compressed: the body is logged, through Redact, as before.
	redacted = redacted[:0]
	e.GET("/secret").Expect().Status(httptest.StatusOK).Body().IsEqual(answer)

	line = fields(t)
	if got, expected := line[bytesSentField], fmt.Sprintf("%d B", len(answer)); got != expected {
		t.Fatalf("expected bytes sent to be %s, got %s", expected, got)
	}

	if response := line[responseField]; response != "the secret is [redacted]" {
		t.Fatalf("expected the redacted response body, got %q", response)
	}

	if len(redacted) != 1 || redacted[0] != answer {
		t.Fatalf("expected Redact to receive the response body once, got %q", redacted)
	}
}

// The access log that records the response body wraps everything inside it in
// a response recorder, the builder's errors handler included. A recorder that
// holds nothing reports the state of the writer under it, so the handler sees
// an untouched response and renders the bare failure into the recorder, where
// the access log reads it.
func TestBuilderEnvelopeUnderARecordingAccessLog(t *testing.T) {
	var buf bytes.Buffer
	ac := accesslog.New(&buf)
	ac.ResponseBody = true

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
			r.Get("/forbidden", func(ctx iris.Context) {
				ctx.StopWithStatus(iris.StatusForbidden)
			})
		}).
		Build()

	body := httptest.New(t, app).GET("/forbidden").Expect().
		Status(httptest.StatusForbidden).Body().Raw()
	bareEnvelope(t, []byte(body), httptest.StatusForbidden, "PERMISSION_DENIED")

	logged := buf.String()
	if !strings.Contains(logged, "|403|GET|/forbidden|") || !strings.Contains(logged, `"PERMISSION_DENIED"`) {
		t.Fatalf("expected the access log to record the 403 and the envelope it was answered with, got: %q", logged)
	}
}
