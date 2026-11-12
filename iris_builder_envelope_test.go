package iris_test

import (
	"bytes"
	stdContext "context"
	"encoding/json"
	stdErrors "errors"
	"io"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/middleware/modrevision"
	"github.com/kataras/iris/v14/rest/errors"
)

// envelope is the rest/errors JSON body as a client reads it.
type envelope struct {
	ErrorCode struct {
		CanonicalName string `json:"canonical_name"`
		Status        int    `json:"status"`
	} `json:"http_error_code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

// singleEnvelope decodes body as exactly one rest/errors document and fails
// the test on anything else: plain text, an empty body or two documents.
//
// The message is not checked here. A failure nobody rendered always carries
// one, the status text, and bareEnvelope checks it; an error the application
// rendered itself carries whatever the application gave it.
func singleEnvelope(t *testing.T, body []byte, status int, canonicalName string) envelope {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(body))

	var doc envelope
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("expected the rest/errors envelope, got %q: %v", body, err)
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		t.Fatalf("expected one JSON document, got more: %s", body)
	}

	if doc.ErrorCode.CanonicalName != canonicalName || doc.ErrorCode.Status != status {
		t.Fatalf("expected %d %s, got %d %s: %s", status, canonicalName,
			doc.ErrorCode.Status, doc.ErrorCode.CanonicalName, body)
	}

	return doc
}

// bareEnvelope is singleEnvelope for a failure that reached the client as a
// status and nothing else before: the envelope written for it has a message.
func bareEnvelope(t *testing.T, body []byte, status int, canonicalName string) envelope {
	t.Helper()

	doc := singleEnvelope(t, body, status, canonicalName)
	if doc.Message == "" {
		t.Fatalf("expected the envelope to carry a message: %s", body)
	}

	return doc
}

// startBuilder runs the first builder step: Prefix when prefix is set,
// NoPrefix otherwise.
func startBuilder(prefix string, configurators ...iris.Configurator) iris.CorsBuilder {
	b := iris.NewBuilder(configurators...)
	if prefix == "" {
		return b.NoPrefix()
	}

	return b.Prefix(prefix)
}

// An application built with iris.NewBuilder answers its own errors with the
// rest/errors envelope. A bare failure status has to answer the same way,
// whoever set it: a route, a middleware given to RouterMiddlewares, or the
// router itself for an unknown path and a method it does not serve.
func TestBuilderEnvelopeCoversRouteRouterMiddleware404And405(t *testing.T) {
	const privateText = "token signature mismatch for key 7f3a"

	for _, prefix := range []string{"/api", ""} {
		name := "Prefix"
		if prefix == "" {
			name = "NoPrefix"
		}

		t.Run(name, func(t *testing.T) {
			withMethodNotAllowed := iris.Defaults(func(opts *iris.Options) {
				opts.Router.FireMethodNotAllowed = true
			})

			app := startBuilder(prefix, withMethodNotAllowed).
				AllowOrigin("none").
				Compression(false).
				LogRequests(false).
				NoHealth().
				Services().
				RouterMiddlewares(func(ctx iris.Context) {
					if ctx.Header("X-Limited") != "" {
						ctx.StopWithStatus(iris.StatusTooManyRequests)
						return
					}

					ctx.Next()
				}).
				Middlewares().
				API("/", func(r iris.Router) {
					r.Get("/ping", func(ctx iris.Context) {
						ctx.WriteString("pong")
					})
					r.Get("/forbidden", func(ctx iris.Context) {
						ctx.StopWithStatus(iris.StatusForbidden)
					})
					r.Get("/private", func(ctx iris.Context) {
						ctx.StopWithError(iris.StatusUnauthorized, iris.PrivateError(stdErrors.New(privateText)))
					})
				}).
				Build()

			e := httptest.New(t, app)

			e.GET(prefix + "/ping").Expect().Status(httptest.StatusOK).Body().IsEqual("pong")

			// A matched route which sets a bare failure status.
			body := e.GET(prefix + "/forbidden").Expect().Status(httptest.StatusForbidden).Body().Raw()
			bareEnvelope(t, []byte(body), httptest.StatusForbidden, "PERMISSION_DENIED")

			// A matched route which stops with a private error: the status
			// renders, the error's own text stays on the server.
			body = e.GET(prefix + "/private").Expect().Status(httptest.StatusUnauthorized).Body().Raw()
			private := bareEnvelope(t, []byte(body), httptest.StatusUnauthorized, "UNAUTHENTICATED")
			if strings.Contains(body, privateText) || strings.Contains(body, "7f3a") {
				t.Fatalf("the private error text reached the client: %s", body)
			}
			// A 401 is not a server fault: the status text, and no detail.
			if private.Message != "Unauthorized" || private.Detail != "" {
				t.Fatalf("expected the message %q and no detail for a private 401, got: %s", "Unauthorized", body)
			}

			// A RouterMiddlewares handler which rejects before routing.
			body = e.GET(prefix+"/ping").WithHeader("X-Limited", "1").Expect().
				Status(httptest.StatusTooManyRequests).Body().Raw()
			bareEnvelope(t, []byte(body), httptest.StatusTooManyRequests, "RESOURCE_EXHAUSTED")

			// An unknown path.
			body = e.GET(prefix + "/unknown").Expect().Status(httptest.StatusNotFound).Body().Raw()
			bareEnvelope(t, []byte(body), httptest.StatusNotFound, "NOT_FOUND")

			if prefix != "" {
				// And one outside the prefix: the envelope is application-wide.
				body = e.GET("/unknown").Expect().Status(httptest.StatusNotFound).Body().Raw()
				bareEnvelope(t, []byte(body), httptest.StatusNotFound, "NOT_FOUND")
			}

			// A known path, the wrong method.
			res := e.POST(prefix + "/ping").Expect().Status(httptest.StatusMethodNotAllowed)
			res.Header("Allow").IsEqual("GET")
			bareEnvelope(t, []byte(res.Body().Raw()), httptest.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		})
	}
}

// The envelope is written through the compression writer, so it reaches the
// client compressed under the Content-Encoding the response announces, and
// the access log, which sits outside compression, records the status.
func TestBuilderEnvelopeUnderCompressionAndAccessLog(t *testing.T) {
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
			r.Get("/forbidden", func(ctx iris.Context) {
				ctx.StopWithStatus(iris.StatusForbidden)
			})
		}).
		Build()

	e := httptest.New(t, app)

	res := e.GET("/forbidden").WithHeader("Accept-Encoding", "gzip").Expect().
		Status(httptest.StatusForbidden).HasContentEncoding("gzip").Raw()

	bareEnvelope(t, []byte(gunzipString(t, res.Body)), httptest.StatusForbidden, "PERMISSION_DENIED")

	logged := strings.TrimSpace(buf.String())
	if lines := strings.Split(logged, "\n"); len(lines) != 1 || logged == "" {
		t.Fatalf("expected the access log to record one line, got %d: %q", len(lines), logged)
	}

	// The default line is Time|Latency|Code|Method|Path|...
	if !strings.Contains(logged, "|403|GET|/forbidden|") {
		t.Fatalf("expected the access log line to carry status 403 for GET /forbidden, got: %q", logged)
	}
}

// Every encoding, not only gzip: a compressor that has not flushed yet must
// not make the builder's handler believe nothing was written and append a
// second document to the one the route rendered, or skip a bare status.
func TestBuilderEnvelopeOnceUnderEveryEncoding(t *testing.T) {
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
			r.Get("/explicit", func(ctx iris.Context) {
				errors.NotFound.Detail(ctx, "plan not found", "no plan has this id")
			})
			r.Get("/bare", func(ctx iris.Context) {
				ctx.StopWithStatus(iris.StatusForbidden)
			})
		}).
		Build()

	e := httptest.New(t, app)

	decompress := func(t *testing.T, body io.Reader, encoding string) []byte {
		t.Helper()

		r, err := compress.NewReader(body, encoding)
		if err != nil {
			t.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}

		return b
	}

	for _, encoding := range compress.AllEncodings {
		t.Run(encoding, func(t *testing.T) {
			res := e.GET("/explicit").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusNotFound).HasContentEncoding(encoding).Raw()
			doc := singleEnvelope(t, decompress(t, res.Body, encoding), httptest.StatusNotFound, "NOT_FOUND")
			if doc.Message != "plan not found" {
				t.Fatalf("expected the route's own error, got: %+v", doc)
			}

			res = e.GET("/bare").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusForbidden).HasContentEncoding(encoding).Raw()
			bareEnvelope(t, decompress(t, res.Body, encoding), httptest.StatusForbidden, "PERMISSION_DENIED")

			res = e.GET("/unknown").WithHeader("Accept-Encoding", encoding).Expect().
				Status(httptest.StatusNotFound).HasContentEncoding(encoding).Raw()
			bareEnvelope(t, decompress(t, res.Body, encoding), httptest.StatusNotFound, "NOT_FOUND")
		})
	}
}

// An application that registers its own 404 or 405 handlers keeps them: the
// builder installs the rest/errors ones only where none exist when the
// application is built.
func TestBuilderCustomNoRouteStillRuns(t *testing.T) {
	custom := iris.ConfiguratorFunc(func(app *iris.Application) {
		app.HandleNoRoute(func(ctx iris.Context) {
			ctx.WriteString("custom not found")
		})
		app.HandleNoMethod(func(ctx iris.Context) {
			ctx.WriteString("custom method not allowed")
		})
	})

	withMethodNotAllowed := iris.Defaults(func(opts *iris.Options) {
		opts.Router.FireMethodNotAllowed = true
	})

	api := func(r iris.Router) {
		r.Get("/ping", func(ctx iris.Context) {
			ctx.WriteString("pong")
		})
	}

	check := func(t *testing.T, app *iris.Application) {
		t.Helper()

		e := httptest.New(t, app)
		e.GET("/unknown").Expect().Status(httptest.StatusNotFound).
			Body().IsEqual("custom not found")
		e.POST("/ping").Expect().Status(httptest.StatusMethodNotAllowed).
			Body().IsEqual("custom method not allowed")
	}

	t.Run("NewBuilder configurator", func(t *testing.T) {
		app := iris.NewBuilder(withMethodNotAllowed, custom).
			NoPrefix().
			AllowOrigin("none").
			Compression(false).
			LogRequests(false).
			NoHealth().
			Services().
			RouterMiddlewares().
			Middlewares().
			API("/", api).
			Build()

		check(t, app)
	})

	t.Run("Services configurator", func(t *testing.T) {
		app := iris.NewBuilder(withMethodNotAllowed).
			NoPrefix().
			AllowOrigin("none").
			Compression(false).
			LogRequests(false).
			NoHealth().
			Services(custom).
			RouterMiddlewares().
			Middlewares().
			API("/", api).
			Build()

		check(t, app)
	})

	t.Run("Default", func(t *testing.T) {
		// Default hands back an application that is not built yet, so its own
		// HandleNoRoute is a registration point too.
		app := iris.Default()
		app.EditOptions(func(opts *iris.Options) {
			opts.Router.FireMethodNotAllowed = true
		})
		custom(app)
		app.Get("/ping", func(ctx iris.Context) {
			ctx.WriteString("pong")
		})

		check(t, app)
	})
}

// /health is a plain text probe. It answers 200, so the envelope has nothing
// to say about it: the body is what the handler writes on a bare application.
func TestBuilderHealthUnchanged(t *testing.T) {
	bare := iris.New()
	bare.Get("/health", modrevision.New(modrevision.Options{
		ServerName:     "Iris Server",
		Env:            "test",
		Developer:      "the-developer",
		CloudDetection: true,
	}))
	expected := httptest.New(t, bare).GET("/health").Expect().
		Status(httptest.StatusOK).Body().Raw()

	if !strings.HasPrefix(expected, "Iris Server (test)") || !strings.HasSuffix(expected, "OK") {
		t.Fatalf("unexpected health body on a bare application: %q", expected)
	}

	for _, prefix := range []string{"/api", ""} {
		name := "Prefix"
		if prefix == "" {
			name = "NoPrefix"
		}

		t.Run(name, func(t *testing.T) {
			app := startBuilder(prefix).
				AllowOrigin("none").
				Compression(false).
				LogRequests(false).
				Health(true, "test", "the-developer").
				Services().
				RouterMiddlewares().
				Middlewares().
				API("/", func(r iris.Router) {
					r.Get("/ping", func(ctx iris.Context) {
						ctx.WriteString("pong")
					})
				}).
				Build()

			res := httptest.New(t, app).GET("/health").Expect().Status(httptest.StatusOK)
			res.Body().IsEqual(expected)
			res.Header("X-Developer").IsEqual("the-developer")
		})
	}
}

// A response that already has a body is left alone, whatever its status: the
// envelope is for a failure nobody rendered.
func TestBuilderWrittenBodyUntouched(t *testing.T) {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			// Body first, the failure status after it.
			r.Get("/body-then-status", func(ctx iris.Context) {
				ctx.WriteString("partial report")
				ctx.StopWithStatus(iris.StatusInternalServerError)
			})
			// The failure status with its own body.
			r.Get("/status-with-body", func(ctx iris.Context) {
				ctx.StopWithText(iris.StatusInternalServerError, "my own failure body")
			})
		}).
		Build()

	e := httptest.New(t, app)

	// The body went out under the 200 it started with: a status set after the
	// first byte cannot be sent any more.
	e.GET("/body-then-status").Expect().Status(httptest.StatusOK).
		Body().IsEqual("partial report")

	e.GET("/status-with-body").Expect().Status(httptest.StatusInternalServerError).
		Body().IsEqual("my own failure body")
}

type envelopeMemo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// envelopeMemoRepository fails every call: Get with a rest/errors code, the
// rest with an error the pipeline knows nothing about.
type envelopeMemoRepository struct{}

var errEnvelopeStoreDown = stdErrors.New("connection refused: 10.0.0.7:5432")

func (envelopeMemoRepository) Create(stdContext.Context, envelopeMemo) (string, error) {
	return "", errEnvelopeStoreDown
}

func (envelopeMemoRepository) Get(stdContext.Context, string) (envelopeMemo, error) {
	return envelopeMemo{}, errors.NotFound
}

func (envelopeMemoRepository) Update(stdContext.Context, string, envelopeMemo, ...string) (bool, error) {
	return false, errEnvelopeStoreDown
}

func (envelopeMemoRepository) Delete(_ stdContext.Context, id string) (bool, error) {
	if id == "gone" {
		return false, nil // nothing to delete: the resource answers 304.
	}

	return false, errEnvelopeStoreDown
}

func (envelopeMemoRepository) List(stdContext.Context, iris.ListFilter, int, int) ([]envelopeMemo, int64, error) {
	return nil, 0, errEnvelopeStoreDown
}

// mvc.UseRepository installs rest/errors.Handler on its own resource and the
// builder installs it application-wide. Two handlers, one failure: the client
// must still read a single document.
func TestBuilderMVCResourceEnvelopeOnce(t *testing.T) {
	for _, compression := range []bool{false, true} {
		name := "plain"
		if compression {
			name = "compressed"
		}

		t.Run(name, func(t *testing.T) {
			app := iris.NewBuilder().
				Prefix("/api").
				AllowOrigin("none").
				Compression(compression).
				LogRequests(false).
				NoHealth().
				Services().
				RouterMiddlewares().
				Middlewares().
				API("/memos", mvc.UseRepository(envelopeMemoRepository{})).
				Build()

			e := httptest.New(t, app)

			// read sends the request and returns the body as the handlers
			// wrote it, after checking the status.
			read := func(method, path string, status int) []byte {
				t.Helper()

				req := e.Request(method, path)
				if !compression {
					return []byte(req.Expect().Status(status).Body().Raw())
				}

				// Brotli on purpose: it hands nothing to the writer under it
				// until it is closed, the case a byte count read from that
				// writer gets wrong.
				res := req.WithHeader("Accept-Encoding", "br").Expect().
					Status(status).HasContentEncoding("br").Raw()

				r, err := compress.NewReader(res.Body, "br")
				if err != nil {
					t.Fatal(err)
				}

				body, err := io.ReadAll(r)
				if err != nil {
					t.Fatal(err)
				}

				return body
			}

			// The repository answers with a rest/errors code.
			body := read("GET", "/api/memos/1", httptest.StatusNotFound)
			singleEnvelope(t, body, httptest.StatusNotFound, "NOT_FOUND")

			// The repository answers with an error nothing maps.
			body = read("DELETE", "/api/memos/1", httptest.StatusInternalServerError)
			singleEnvelope(t, body, httptest.StatusInternalServerError, "INTERNAL")
			if bytes.Contains(body, []byte("10.0.0.7")) {
				t.Fatalf("the repository's error text reached the client: %s", body)
			}

			// A path the resource does not serve.
			body = read("GET", "/api/memos/1/nothing/here", httptest.StatusNotFound)
			bareEnvelope(t, body, httptest.StatusNotFound, "NOT_FOUND")

			// Nothing to delete: 304 is not a failure and keeps its empty body.
			if body = read("DELETE", "/api/memos/gone", httptest.StatusNotModified); len(body) != 0 {
				t.Fatalf("expected an empty 304, got: %s", body)
			}
		})
	}
}
