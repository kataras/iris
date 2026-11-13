package context_test

import (
	"crypto/tls"
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// AbsoluteURI writes "scheme://host" whether the scheme comes from the request
// URL (an absolute-form request line, or code that sets URL.Scheme) or is
// worked out from the connection. It used to drop the colon in the first case:
// "https//example.com:8443/b". See #1733.
func TestAbsoluteURI(t *testing.T) {
	app := iris.New()
	app.Get("/a", func(ctx iris.Context) {
		ctx.WriteString(ctx.AbsoluteURI(ctx.Request().Header.Get("X-Ref")))
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		target string
		tls    bool
		ref    string
		want   string
	}{
		{"https request URL", "https://example.com:8443/a", false, "/b", "https://example.com:8443/b"},
		{"http request URL", "http://example.com/a", false, "/b", "http://example.com/b"},
		{"request URL with a scheme, user info", "https://example.com/a", false, "@user:pass/b", "https://user:pass@example.com/b"},
		{"origin-form", "/a", false, "/b", "http://example.com/b"},
		{"origin-form over TLS", "/a", true, "/b", "https://example.com/b"},
		{"origin-form, user info", "/a", false, "@user:pass/b", "http://user:pass@example.com/b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := nethttptest.NewRequest(http.MethodGet, tt.target, nil)
			r.Header.Set("X-Ref", tt.ref)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}

			w := nethttptest.NewRecorder()
			app.ServeHTTP(w, r)

			if got := w.Body.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// FullRequestURI goes through AbsoluteURI. The test client with a base URL
// sends a request URL that carries its scheme.
func TestFullRequestURI(t *testing.T) {
	app := iris.New()
	app.Get("/full", func(ctx iris.Context) {
		ctx.WriteString(ctx.FullRequestURI())
	})

	e := httptest.New(t, app, httptest.URL("https://example.com"))
	e.GET("/full").Expect().Status(httptest.StatusOK).Body().IsEqual("https://example.com/full")
}
