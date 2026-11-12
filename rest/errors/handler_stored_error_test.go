package errors_test

import (
	"bufio"
	"encoding/json"
	stdErrors "errors"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// storedErrorApp is an application with errors.Handler installed and an
// Options value that records what its logger and its collector receive. Any
// "outer" handler is registered ahead of both, so it sees the chain return.
func storedErrorApp(t *testing.T, opts *errors.Options, outer ...iris.Handler) (app *iris.Application, logged, collected *[]string) {
	t.Helper()

	logged, collected = new([]string), new([]string)

	opts.SetLogger(func(ctx iris.Context, err error) {
		*logged = append(*logged, err.Error())
	})
	opts.SetCollector(func(ctx iris.Context, err *errors.Error) {
		*collected = append(*collected, err.Message)
	})

	app = iris.New()
	app.UseRouter(outer...)
	app.UseRouter(opts.Handler(), errors.Handler)

	return app, logged, collected
}

// An error a handler stored on the Context with a status of 500 or above is
// the server's own trouble: a store that is down, a driver that failed. Its
// text never reaches the client. It is logged, and the client reads the
// status with its canonical code and the generic message, exactly what
// errors.HandleError does with an error no mapping recognises. The status the
// handler chose stays: a 503 is still a 503.
func TestHandlerUnmappedServerErrorTextStaysOnTheServer(t *testing.T) {
	const storeText = "dial tcp 10.0.0.5:6379: refused"

	app, logged, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/unavailable", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusServiceUnavailable, stdErrors.New(storeText))
	})
	app.Get("/internal", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusInternalServerError, stdErrors.New(storeText))
	})

	e := httptest.New(t, app)

	for path, expected := range map[string]struct {
		status        int
		canonicalName string
	}{
		"/unavailable": {iris.StatusServiceUnavailable, "UNAVAILABLE"},
		"/internal":    {iris.StatusInternalServerError, "INTERNAL"},
	} {
		*logged = (*logged)[:0]

		res := e.GET(path).Expect().Status(expected.status)
		if body := res.Body().Raw(); strings.Contains(body, storeText) || strings.Contains(body, "10.0.0.5") {
			t.Fatalf("%s: the stored error text reached the client: %s", path, body)
		}

		doc := res.JSON().Object()
		doc.Value("http_error_code").Object().
			HasValue("canonical_name", expected.canonicalName).
			HasValue("status", expected.status)
		doc.HasValue("message", "server error")

		if len(*logged) != 1 || (*logged)[0] != storeText {
			t.Fatalf("%s: expected the configured logger to receive the stored error once, got %q", path, *logged)
		}
	}
}

// Below 500 a stored error that is not private is a message written for the
// client, and it still renders as one.
func TestHandlerStoredClientErrorRendersItsText(t *testing.T) {
	app, logged, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/name", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusBadRequest, stdErrors.New("name is required"))
	})

	e := httptest.New(t, app)

	doc := e.GET("/name").Expect().Status(iris.StatusBadRequest).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "INVALID_ARGUMENT").
		HasValue("status", iris.StatusBadRequest)
	doc.HasValue("message", "name is required")

	if len(*logged) != 0 {
		t.Fatalf("a client error is not logged, got %q", *logged)
	}
}

// A stored error that a mapping recognises renders through that mapping,
// whatever status the handler set next to it.
func TestHandlerStoredMappedErrorKeepsItsMapping(t *testing.T) {
	errPlanNotFound := stdErrors.New("plan not found")

	app, _, _ := storedErrorApp(t, errors.NewOptions().MapErrors(errors.NotFound, errPlanNotFound))

	app.Get("/plan", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusInternalServerError, errPlanNotFound)
	})

	e := httptest.New(t, app)

	doc := e.GET("/plan").Expect().Status(iris.StatusNotFound).JSON().Object()
	doc.Value("http_error_code").Object().HasValue("canonical_name", "NOT_FOUND")
	doc.HasValue("message", "plan not found")
}

// A JSON error renders as the client's fault or as the server's, and for a
// stored one the status next to it says which. A body that does not parse,
// stored with a 400, is the client's: it used to answer 500 "unable to parse
// response body", because the handler passed "the error is public" where the
// question is "is this the server's fault". A value the server could not
// encode, stored with a 500, is still the server's.
func TestHandlerStoredJSONErrorFollowsTheStatus(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions())

	app.Post("/decode", func(ctx iris.Context) {
		var v map[string]any
		if err := ctx.ReadJSON(&v); err != nil {
			ctx.StopWithPlainError(iris.StatusBadRequest, err)
			return
		}

		ctx.WriteString("ok")
	})
	app.Get("/encode", func(ctx iris.Context) {
		_, err := json.Marshal(math.Inf(1)) // JSON has no infinity.
		if err == nil {
			t.Error("expected encoding +Inf as JSON to fail")
		}

		ctx.StopWithPlainError(iris.StatusInternalServerError, err)
	})

	e := httptest.New(t, app)

	// The client sent half a document.
	doc := e.POST("/decode").WithHeader("Content-Type", "application/json").WithBytes([]byte(`{"a":`)).
		Expect().Status(iris.StatusBadRequest).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "INVALID_ARGUMENT").
		HasValue("status", iris.StatusBadRequest)
	doc.HasValue("message", "unable to parse request body")
	doc.HasValue("detail", "malformed JSON")

	// The server could not encode its own value.
	doc = e.GET("/encode").Expect().Status(iris.StatusInternalServerError).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "INTERNAL").
		HasValue("status", iris.StatusInternalServerError)
	doc.HasValue("message", "unable to parse response body")
}

// A private error below 500 is not a server fault and must not read as one:
// the client reads the status text, with no detail. The private text stays
// off the wire and still reaches the collector, as it did before.
func TestHandlerPrivateClientErrorRendersTheStatusText(t *testing.T) {
	const privateText = "token signature mismatch for key 7f3a"

	app, _, collected := storedErrorApp(t, errors.NewOptions())

	app.Get("/private", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusUnauthorized, iris.PrivateError(stdErrors.New(privateText)))
	})

	e := httptest.New(t, app)

	res := e.GET("/private").Expect().Status(iris.StatusUnauthorized)
	if body := res.Body().Raw(); strings.Contains(body, privateText) || strings.Contains(body, "7f3a") {
		t.Fatalf("the private error text reached the client: %s", body)
	}

	doc := res.JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "UNAUTHENTICATED").
		HasValue("status", iris.StatusUnauthorized)
	doc.HasValue("message", "Unauthorized")
	doc.NotContainsKey("detail")

	if len(*collected) != 1 || (*collected)[0] != privateText {
		t.Fatalf("expected the collector to receive the private error text once, got %q", *collected)
	}
}

// A failure status renders whether or not the handler stopped the chain:
// ctx.NotFound() sets the status and returns, and so does the file server for
// a missing file.
func TestHandlerRendersABareStatusWithoutStop(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/missing", func(ctx iris.Context) {
		ctx.NotFound()
	})
	app.Get("/unchanged", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNotModified)
	})

	e := httptest.New(t, app)

	doc := e.GET("/missing").Expect().Status(iris.StatusNotFound).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "NOT_FOUND").
		HasValue("status", iris.StatusNotFound)
	doc.HasValue("message", "Not Found")

	// Nothing changes below 400.
	e.GET("/unchanged").Expect().Status(iris.StatusNotModified).Body().IsEmpty()
}

// Once the status line has gone out there is no response left to render: a
// handler that flushed its headers and then failed must not get a JSON
// document appended to the 200 the client already received.
func TestHandlerLeavesACommittedResponseAlone(t *testing.T) {
	app, _, collected := storedErrorApp(t, errors.NewOptions())

	app.Get("/flushed", func(ctx iris.Context) {
		ctx.SetContentType("text/plain")
		ctx.ResponseWriter().Flush() // the status line and the headers are on the wire.
		ctx.StopWithStatus(iris.StatusInternalServerError)
	})

	srv := serve(t, app)

	res, err := http.Get(srv.URL + "/flushed")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected the 200 that was already sent, got %d", res.StatusCode)
	}

	if len(body) != 0 {
		t.Fatalf("expected nothing after the flushed headers, got %q", body)
	}

	if len(*collected) != 0 {
		t.Fatalf("expected no error document to be built for a committed response, got %q", *collected)
	}
}

// A hijacked connection belongs to the handler that took it. Whatever status
// is left on the Context afterwards, there is no HTTP response to write.
func TestHandlerLeavesAHijackedConnectionAlone(t *testing.T) {
	const raw = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nhi"

	// The handler closes the connection before the chain has returned, so the
	// client is done reading while errors.Handler has yet to run. "served" is
	// closed once the whole chain is back.
	served := make(chan struct{})

	app, _, collected := storedErrorApp(t, errors.NewOptions(), func(ctx iris.Context) {
		ctx.Next()
		close(served)
	})

	app.Get("/hijacked", func(ctx iris.Context) {
		conn, rw, err := ctx.ResponseWriter().Hijack()
		if err != nil {
			ctx.StopWithStatus(iris.StatusNotImplemented)
			return
		}
		defer conn.Close()

		rw.WriteString(raw)
		rw.Flush()

		ctx.StopWithStatus(iris.StatusInternalServerError)
	})

	srv := serve(t, app)

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := io.WriteString(conn, "GET /hijacked HTTP/1.1\r\nHost: example.com\r\n\r\n"); err != nil {
		t.Fatal(err)
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK || string(body) != "hi" {
		t.Fatalf("expected the handler's own reply, got %d %q", res.StatusCode, body)
	}

	<-served

	if len(*collected) != 0 {
		t.Fatalf("expected no error document to be built on a hijacked connection, got %q", *collected)
	}
}
