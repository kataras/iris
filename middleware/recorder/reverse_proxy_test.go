package recorder_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// The setup of #1809: recorder.Response in front of a reverse proxy mounted
// with iris.FromStd, and an upstream that answers chunked, with no
// Content-Length. httputil.ReverseProxy drops the upstream's Transfer-Encoding
// header, which is hop-by-hop, and flushes after every write of such an
// answer. So the recorder is flushed while it holds the upstream's status code
// and headers, with nothing on it that says chunked. The client has to get the
// answer of the upstream: its status code, its headers and its whole body.
func TestRecorderResponseInFrontOfAReverseProxy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		upstream    http.HandlerFunc
		status      int
		contentType string
		body        string
	}{
		{
			// The handler of the issue, with a status code and headers of its
			// own. It sets Transfer-Encoding and frames the body by hand, and
			// net/http chunks the answer once more, so a client reads that
			// framing as the body, from the upstream directly as well.
			name: "the upstream of the issue",
			upstream: func(w http.ResponseWriter, r *http.Request) {
				chunkedWriter := httputil.NewChunkedWriter(w)
				w.Header().Set("Transfer-Encoding", "chunked")
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Upstream", "yes")
				w.WriteHeader(http.StatusAccepted)

				chunkedWriter.Write([]byte("hello"))
				chunkedWriter.Close()
			},
			status:      http.StatusAccepted,
			contentType: "text/plain; charset=utf-8",
			body:        "5\r\nhello\r\n0\r\n",
		},
		{
			name: "a flush between writes",
			upstream: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Upstream", "yes")
				w.WriteHeader(http.StatusNotFound)

				io.WriteString(w, `{"hello":`)
				w.(http.Flusher).Flush()
				io.WriteString(w, `"world"}`)
			},
			status:      http.StatusNotFound,
			contentType: "application/json",
			body:        `{"hello":"world"}`,
		},
		{
			// The upstream sends its status code and headers ahead of the body.
			// The proxy arms a flush on a timer before it reads the body, so
			// the recorder is often flushed while its body is still empty. Not
			// in every run: that flush races the first write.
			name: "the head flushed before the body",
			upstream: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.Header().Set("X-Upstream", "yes")
				w.WriteHeader(http.StatusCreated)
				w.(http.Flusher).Flush()

				io.WriteString(w, "<a>")
				w.(http.Flusher).Flush()
				io.WriteString(w, "</a>")
				w.(http.Flusher).Flush()
			},
			status:      http.StatusCreated,
			contentType: "application/xml",
			body:        "<a></a>",
		},
		{
			name: "one write and a flush",
			upstream: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Upstream", "yes")
				w.WriteHeader(http.StatusInternalServerError)

				io.WriteString(w, "boom")
				w.(http.Flusher).Flush()
			},
			status:      http.StatusInternalServerError,
			contentType: "text/plain; charset=utf-8",
			body:        "boom",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(tc.upstream)
			defer upstream.Close()

			u, err := url.Parse(upstream.URL)
			if err != nil {
				t.Fatal(err)
			}

			app := iris.New()
			app.Use(recorder.Response)
			app.Get("/", iris.FromStd(httputil.NewSingleHostReverseProxy(u)))

			resp, body := get(t, app, "/")

			if resp.StatusCode != tc.status {
				t.Errorf("expected the upstream's %d, got %d", tc.status, resp.StatusCode)
			}

			if got := resp.Header.Get("Content-Type"); got != tc.contentType {
				t.Errorf("expected the upstream's Content-Type %q, got %q", tc.contentType, got)
			}

			if got := resp.Header.Get("X-Upstream"); got != "yes" {
				t.Errorf("expected the upstream's header, got %q", got)
			}

			if body != tc.body {
				t.Errorf("expected the upstream's body %q, got %q", tc.body, body)
			}
		})
	}
}
