package iris_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

// A standard handler that declares a Content-Length and sets a failure status
// has chosen its body, an empty one included. It is what
// httputil.ReverseProxy relays for an upstream's empty 401, 404 or 502.
//
// Iris defers WriteHeader, so to the builder's errors handler that response
// looks like one nobody touched. The envelope it wrote went into a declared
// length of zero: net/http refused the bytes, and the client read a 401
// labelled application/json with no body. Under compression the client was
// also told the nothing it received was gzip.
//
// A standard handler that sets the status and declares nothing still gets the
// envelope.
func TestBuilderEnvelopeLeavesADeclaredContentLengthAlone(t *testing.T) {
	for _, test := range []struct {
		name        string
		compression bool
	}{
		{"Plain", false},
		{"Gzip", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var collected []string

			opts := errors.NewOptions().SetCollector(func(ctx iris.Context, err *errors.Error) {
				collected = append(collected, ctx.Path())
			})

			app := iris.NewBuilder().
				NoPrefix().
				AllowOrigin("none").
				Compression(test.compression).
				LogRequests(false).
				NoHealth().
				Errors(opts).
				Services().
				RouterMiddlewares().
				Middlewares().
				API("/", func(r iris.Router) {
					r.Get("/declared", iris.FromStd(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Length", "0")
						w.WriteHeader(http.StatusUnauthorized)
					}))
					r.Get("/bare", iris.FromStd(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusUnauthorized)
					}))
				}).
				Build()

			srv := nethttptest.NewServer(app)
			defer srv.Close()

			get := func(path string) (*http.Response, []byte) {
				t.Helper()

				req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
				if err != nil {
					t.Fatal(err)
				}

				if test.compression {
					// Set by hand, so the transport leaves the response as it came.
					req.Header.Set("Accept-Encoding", "gzip")
				}

				res, err := srv.Client().Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()

				body, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("%s: reading the body: %v", path, err)
				}

				return res, body
			}

			// The handler that declared its length: its answer, as it wrote it.
			res, body := get("/declared")

			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("/declared: expected the handler's 401, got %d", res.StatusCode)
			}

			if len(body) != 0 {
				t.Errorf("/declared: expected the empty body the handler declared, got %q", body)
			}

			if contentType := res.Header.Get("Content-Type"); contentType != "" {
				t.Errorf("/declared: nothing was written for the handler, but the response says Content-Type: %s", contentType)
			}

			if encoding := res.Header.Get("Content-Encoding"); encoding != "" {
				t.Errorf("/declared: an empty body is not compressed, but the response says Content-Encoding: %s", encoding)
			}

			if length := res.Header.Get("Content-Length"); length != "0" {
				t.Errorf("/declared: expected the Content-Length: 0 the handler declared, got %q", length)
			}

			if len(collected) != 0 {
				t.Errorf("/declared: the collector was called for a document that was never sent: %q", collected)
			}

			// The handler that only set the status: the envelope, as before.
			collected = collected[:0]
			res, body = get("/bare")

			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("/bare: expected 401, got %d", res.StatusCode)
			}

			if test.compression {
				if encoding := res.Header.Get("Content-Encoding"); encoding != "gzip" {
					t.Fatalf("/bare: expected the envelope to be gzip, got Content-Encoding %q", encoding)
				}

				zr, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Fatalf("/bare: the body is not gzip: %v", err)
				}

				if body, err = io.ReadAll(zr); err != nil {
					t.Fatalf("/bare: reading the gzip body: %v", err)
				}
			}

			bareEnvelope(t, body, http.StatusUnauthorized, "UNAUTHENTICATED")

			if len(collected) != 1 {
				t.Errorf("/bare: expected the collector to receive the envelope once, got %q", collected)
			}
		})
	}
}
