package errors_test

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

// A hijacked connection belongs to the handler that took it, and a handler
// that panics afterwards (a websocket read loop is the usual one) reaches the
// render through RecoveryHandler, or through the recovery the builder
// installs. Hijack leaves Written at StatusCodeWritten, which the "a body
// already went out" guard lets through, so the 500 document used to be written
// to a connection that carries no HTTP response any more: net/http refuses
// those bytes and reports each attempt in the server's error log. The panic is
// logged and nothing is rendered.
func TestRenderOnAHijackedConnectionIsSuppressed(t *testing.T) {
	const raw = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nhi"

	var logged, collected []string

	opts := errors.NewOptions()
	opts.SetLogger(func(ctx iris.Context, err error) {
		logged = append(logged, err.Error())
	})
	opts.SetCollector(func(ctx iris.Context, err *errors.Error) {
		collected = append(collected, err.Message)
	})

	app := iris.New()
	app.UseRouter(opts.Handler(), errors.RecoveryHandler)
	app.Get("/hijacked", func(ctx iris.Context) {
		conn, rw, err := ctx.ResponseWriter().Hijack()
		if err != nil {
			ctx.StopWithStatus(iris.StatusNotImplemented)
			return
		}
		defer conn.Close()

		rw.WriteString(raw)
		rw.Flush()

		panic("after the hijack")
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	// The handler replies before it panics, so the client is done reading
	// while RecoveryHandler has yet to run. "served" is closed once the whole
	// chain is back.
	served := make(chan struct{})
	serverLog := new(bytes.Buffer)

	srv := nethttptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.ServeHTTP(w, r)
		close(served)
	}))
	srv.Config.ErrorLog = log.New(serverLog, "", 0)
	srv.Start()
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
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

	if got := serverLog.String(); got != "" {
		t.Fatalf("expected nothing written to the hijacked connection, the server logged: %s", got)
	}

	if len(collected) != 0 {
		t.Fatalf("expected no error document to be built on a hijacked connection, got %q", collected)
	}

	if len(logged) == 0 {
		t.Fatal("expected the panic to be logged")
	}
}
