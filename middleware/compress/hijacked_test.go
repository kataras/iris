package compress_test

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
	"github.com/kataras/iris/v14/middleware/compress"
)

// A handler that hijacks the connection owns it from then on. A websocket
// upgrade is the usual one, and on an application with compression on it runs
// under this middleware for every client that sends Accept-Encoding, which is
// every browser. The compressor was acquired before the handler ran and has
// nothing to say on that connection. Closing it on release used to write its
// header and footer through the response writer: net/http refuses those bytes
// and reports each attempt in the server's error log, once per upgrade.
func TestHijackedConnectionIsNotWrittenOnRelease(t *testing.T) {
	const raw = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nhi"

	for _, encoding := range compress.AllEncodings {
		t.Run(encoding, func(t *testing.T) {
			app := iris.New()
			app.UseRouter(compress.Safe)
			app.Get("/hijacked", func(ctx iris.Context) {
				conn, rw, err := ctx.ResponseWriter().Hijack()
				if err != nil {
					ctx.StopWithStatus(iris.StatusNotImplemented)
					return
				}
				defer conn.Close()

				rw.WriteString(raw)
				rw.Flush()
			})

			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			// The handler closes the connection before the chain has returned,
			// so the client is done reading while the middleware has yet to
			// release its writer. "served" is closed once the chain is back.
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

			request := "GET /hijacked HTTP/1.1\r\nHost: example.com\r\nAccept-Encoding: " + encoding + "\r\n\r\n"
			if _, err := io.WriteString(conn, request); err != nil {
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
		})
	}
}
