package recorder_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// get serves app on a real connection and returns the response to GET path
// with its body read. A flush is about what is on the wire before the handler
// returns, so an in-process tester is not enough here.
func get(t *testing.T, app *iris.Application, path string) (*http.Response, string) {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	defer srv.Close()

	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return resp, string(body)
}

// A flush sends the status line and the headers. A recorder keeps the body
// until the handler chain returns, but the status code and the headers it
// holds have to reach the writer under it before the flush does: after it that
// writer has sent its own 200 and its own header map, and what the release
// copies over arrives too late.
func TestRecorderFlushSendsTheRecordedStatusAndHeaders(t *testing.T) {
	for _, tc := range []struct {
		name    string
		chunked bool // the handler sets Transfer-Encoding: chunked itself.
	}{
		{"buffered", false},
		{"chunked", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := iris.New()
			app.Use(recorder.Response)
			app.Get("/", func(ctx iris.Context) {
				w := ctx.ResponseWriter()
				if tc.chunked {
					w.Header().Set("Transfer-Encoding", "chunked")
				}
				w.Header().Set("Content-Type", "text/xml")
				w.Header().Set("X-Plan", "starter")
				w.WriteHeader(http.StatusCreated)

				w.Write([]byte("<a>"))
				w.Flush()
				w.Write([]byte("</a>"))
			})

			resp, body := get(t, app, "/")

			if resp.StatusCode != http.StatusCreated {
				t.Errorf("expected the 201 the handler set, got %d", resp.StatusCode)
			}

			if got := resp.Header.Get("Content-Type"); got != "text/xml" {
				t.Errorf("expected the Content-Type the handler set, got %q", got)
			}

			if got := resp.Header.Get("X-Plan"); got != "starter" {
				t.Errorf("expected the header the handler set, got %q", got)
			}

			if body != "<a></a>" {
				t.Errorf("expected the whole body, got %q", body)
			}
		})
	}
}

// lockedBuffer is an access log destination the test reads while the server
// may still be writing to it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// The same through httputil.ReverseProxy under an access log that records the
// response body, which is where it was reported (#1740). The proxy flushes
// after every write when the upstream sends no Content-Length, so the client
// received a 200 without any of the upstream's headers for every such answer.
// The access log still has to end up with the body.
func TestRecorderFlushFromAReverseProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusNotFound)

		io.WriteString(w, "<error>")
		w.(http.Flusher).Flush() // from here on the response has no Content-Length.
		io.WriteString(w, "</error>")
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
		},
	}

	logs := new(lockedBuffer)
	ac := accesslog.New(logs)
	ac.ResponseBody = true // this is what makes the access log record the response.

	app := iris.New()
	app.UseRouter(ac.Handler)
	app.Get("/proxy/{p:path}", func(ctx iris.Context) {
		proxy.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
	})

	resp, body := get(t, app, "/proxy/wmts")
	ac.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected the upstream's 404, got %d", resp.StatusCode)
	}

	if got := resp.Header.Get("Content-Type"); got != "application/xml" {
		t.Errorf("expected the upstream's Content-Type, got %q", got)
	}

	if got := resp.Header.Get("X-Upstream"); got != "yes" {
		t.Errorf("expected the upstream's header, got %q", got)
	}

	if body != "<error></error>" {
		t.Errorf("expected the upstream's body, got %q", body)
	}

	if got := logs.String(); !strings.Contains(got, "<error></error>") {
		t.Errorf("expected the access log to hold the response body, got %q", got)
	}
}
