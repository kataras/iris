package iris_test

import (
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/sse"

	"github.com/iris-contrib/httpexpect/v2"
	"github.com/kataras/neffos"
)

// edgeBuilderApp is a builder application with the given routes under /, and
// nothing optional switched on.
func edgeBuilderApp(routes func(r iris.Router)) *iris.Application {
	return iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", routes).
		Build()
}

// A failure status is rendered whether or not the handler stopped the chain.
// ctx.NotFound() only sets the status, and a route that calls it and returns
// used to answer an empty body.
func TestBuilderEnvelopeCoversNotFoundWithoutStop(t *testing.T) {
	app := edgeBuilderApp(func(r iris.Router) {
		r.Get("/plans/{id}", func(ctx iris.Context) {
			ctx.NotFound()
		})
	})

	body := httptest.New(t, app).GET("/plans/42").Expect().
		Status(httptest.StatusNotFound).Body().Raw()

	doc := bareEnvelope(t, []byte(body), httptest.StatusNotFound, "NOT_FOUND")
	if doc.Message != "Not Found" {
		t.Fatalf("expected the status text as the message, got: %s", body)
	}
}

// The file server sets 404 for a missing file and returns, without stopping
// the chain. Under the builder that is the envelope; a file that exists is
// served byte for byte, and the answers that carry no body keep none.
func TestBuilderFileServerMissingFile(t *testing.T) {
	const contents = "body { color: #36c; }\n"

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "site.css"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/static", fileserver.New(root, fileserver.Options{})).
		Build()

	e := httptest.New(t, app)

	// An existing file, byte for byte.
	existing := e.GET("/static/site.css").Expect().Status(httptest.StatusOK)
	existing.Body().IsEqual(contents)
	lastModified := existing.Header("Last-Modified").NotEmpty().Raw()

	// A missing one.
	body := e.GET("/static/missing.css").Expect().Status(httptest.StatusNotFound).Body().Raw()
	bareEnvelope(t, []byte(body), httptest.StatusNotFound, "NOT_FOUND")

	// A fresh copy: 304 is not a failure and stays empty.
	e.GET("/static/site.css").WithHeader("If-Modified-Since", lastModified).Expect().
		Status(httptest.StatusNotModified).Body().IsEmpty()

	// HEAD for a missing file over a real connection: the status and no body,
	// which is all a HEAD answer may carry.
	srv := nethttptest.NewServer(app)
	defer srv.Close()

	res, err := srv.Client().Head(srv.URL + "/static/missing.css")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	headBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusNotFound || len(headBody) != 0 {
		t.Fatalf("expected an empty 404 for HEAD, got %d %q", res.StatusCode, headBody)
	}

	// And the connection is still usable for the next request.
	res, err = srv.Client().Get(srv.URL + "/static/site.css")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if got, _ := io.ReadAll(res.Body); string(got) != contents {
		t.Fatalf("expected the file after the HEAD request, got %q", got)
	}
}

// Nothing below 400 is touched: a redirect and a 304 answer what they answer
// on an application built with iris.New().
func TestBuilderEnvelopeLeavesRedirectsAndNotModifiedAlone(t *testing.T) {
	routes := func(r iris.Router) {
		r.Get("/old", func(ctx iris.Context) {
			ctx.Redirect("/new", iris.StatusFound)
		})
		r.Get("/cached", func(ctx iris.Context) {
			ctx.StopWithStatus(iris.StatusNotModified)
		})
		r.Get("/empty", func(ctx iris.Context) {
			ctx.StatusCode(iris.StatusNoContent)
		})
	}

	bare := iris.New()
	routes(bare)

	expect := httptest.New(t, bare)
	got := httptest.New(t, edgeBuilderApp(routes))

	for _, path := range []string{"/old", "/cached", "/empty"} {
		expected := expect.GET(path).WithRedirectPolicy(httpexpect.DontFollowRedirects).Expect().Raw()
		expectedBody, _ := io.ReadAll(expected.Body)

		actual := got.GET(path).WithRedirectPolicy(httpexpect.DontFollowRedirects).Expect().Raw()
		actualBody, _ := io.ReadAll(actual.Body)

		if actual.StatusCode != expected.StatusCode || string(actualBody) != string(expectedBody) {
			t.Fatalf("%s: expected %d %q as on a bare application, got %d %q", path,
				expected.StatusCode, expectedBody, actual.StatusCode, actualBody)
		}

		if location := actual.Header.Get("Location"); location != expected.Header.Get("Location") {
			t.Fatalf("%s: expected Location %q, got %q", path, expected.Header.Get("Location"), location)
		}
	}
}

// A stream that fails after its first event has a body already: the events.
// No error document may follow them.
func TestBuilderEnvelopeLeavesAStartedStreamAlone(t *testing.T) {
	app := edgeBuilderApp(func(r iris.Router) {
		r.Get("/events", func(ctx iris.Context) {
			stream, err := sse.Open(ctx)
			if err != nil {
				ctx.StopWithStatus(iris.StatusNotImplemented)
				return
			}
			defer stream.Close(nil)

			if err := stream.Send(sse.Event{Data: []byte("first")}); err != nil {
				return
			}

			ctx.StopWithStatus(iris.StatusInternalServerError)
		})
	})
	app.Logger().SetLevel("disable")

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	client := srv.Client()
	client.Timeout = 10 * time.Second

	res, err := client.Get(srv.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected the 200 the stream started with, got %d", res.StatusCode)
	}

	if !strings.Contains(string(body), "data: first") {
		t.Fatalf("expected the first event in the stream, got %q", body)
	}

	if strings.Contains(string(body), "http_error_code") {
		t.Fatalf("an error document was appended to a stream: %q", body)
	}
}

type edgeWebsocketController struct{}

func (c *edgeWebsocketController) Namespace() string { return "default" }

func (c *edgeWebsocketController) OnNamespaceConnected(*neffos.NSConn, neffos.Message) error {
	return nil
}

// A websocket endpoint refuses a request that is not a handshake with its own
// 400 and its own text. That answer has a body, so it is left as it is.
func TestBuilderRefusedWebsocketHandshakeUnchanged(t *testing.T) {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/ws", mvc.Websocket(&edgeWebsocketController{})).
		Build()

	res := httptest.New(t, app).GET("/ws").Expect().Status(httptest.StatusBadRequest)
	res.Body().IsEqual("Bad Request\n")
	res.Header("Sec-Websocket-Version").IsEqual("13")
}

// iris.Default is built on the builder and hands back an application that is
// not built yet. The 404 and 405 handlers are installed when it is.
func TestDefaultAnswersTheEnvelope(t *testing.T) {
	app := iris.Default()
	app.Get("/forbidden", func(ctx iris.Context) {
		ctx.StopWithStatus(iris.StatusForbidden)
	})

	e := httptest.New(t, app)

	body := e.GET("/forbidden").Expect().Status(httptest.StatusForbidden).Body().Raw()
	bareEnvelope(t, []byte(body), httptest.StatusForbidden, "PERMISSION_DENIED")

	body = e.GET("/unknown").Expect().Status(httptest.StatusNotFound).Body().Raw()
	bareEnvelope(t, []byte(body), httptest.StatusNotFound, "NOT_FOUND")
}

// The two failures the builder's handler does not render. Both are answered
// before it runs, and the documentation of NewBuilder names them; this test
// keeps that sentence true.
func TestBuilderEnvelopeNamedGaps(t *testing.T) {
	// A UseRouter handler registered from a Configurator sits ahead of every
	// built-in the builder registers, the errors handler included.
	early := iris.ConfiguratorFunc(func(app *iris.Application) {
		app.UseRouter(func(ctx iris.Context) {
			if ctx.Header("X-Blocked") != "" {
				ctx.StopWithStatus(iris.StatusForbidden)
				return
			}

			ctx.Next()
		})
	})

	app := iris.NewBuilder(early).
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
				ctx.WriteString("ok")
			})
		}).
		Build()

	e := httptest.New(t, app)

	e.POST("/upload").WithHeader("X-Blocked", "1").WithText("x").Expect().
		Status(httptest.StatusForbidden).Body().IsEmpty()

	// The compression middleware's own 415: a request body that is not what
	// its Content-Encoding says.
	e.POST("/upload").WithHeader("Content-Encoding", "gzip").WithText("not gzip at all").Expect().
		Status(httptest.StatusUnsupportedMediaType).Body().IsEmpty()
}
