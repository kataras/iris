package fasthttputil

import (
	"net"
	"testing"

	"github.com/valyala/fasthttp"
)

// BenchmarkPlainStreaming measures end-to-end plaintext streaming performance
// for fasthttp client and server.
//
// It issues http requests over a small number of keep-alive connections.
func BenchmarkPlainStreaming(b *testing.B) {
	benchmark(b, streamingHandler, false)
}

// BenchmarkPlainHandshake measures end-to-end plaintext handshake performance
// for fasthttp client and server.
//
// It re-establishes new connection per each http request.
func BenchmarkPlainHandshake(b *testing.B) {
	benchmark(b, handshakeHandler, false)
}

// BenchmarkTLSStreaming measures end-to-end TLS streaming performance
// for fasthttp client and server.
//
// It issues http requests over a small number of TLS keep-alive connections.
func BenchmarkTLSStreaming(b *testing.B) {
	benchmark(b, streamingHandler, true)
}

// BenchmarkTLSHandshake measures end-to-end TLS handshake performance
// for fasthttp client and server.
//
// It re-establishes new TLS connection per each http request.
func BenchmarkTLSHandshake(b *testing.B) {
	benchmark(b, handshakeHandler, true)
}

func benchmark(b *testing.B, h fasthttp.RequestHandler, isTLS bool) {
	ln := NewInmemoryListener()
	serverStopCh := startServer(b, ln, h, isTLS)
	c := newClient(ln, isTLS)
	b.RunParallel(func(pb *testing.PB) {
		runRequests(b, pb, c)
	})
	ln.Close()
	<-serverStopCh
}

func streamingHandler(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("foobar")
}

func handshakeHandler(ctx *fasthttp.RequestCtx) {
	streamingHandler(ctx)

	// Explicitly close connection after each response.
	ctx.SetConnectionClose()
}

func startServer(b *testing.B, ln *InmemoryListener, h fasthttp.RequestHandler, isTLS bool) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		var err error
		if isTLS {
			err = fasthttp.ServeTLS(ln, certFile, keyFile, h)
		} else {
			err = fasthttp.Serve(ln, h)
		}
		if err != nil {
			b.Fatalf("unexpected error in server: %s", err)
		}
		close(ch)
	}()
	return ch
}

const (
	certFile = "./ssl-cert-snakeoil.pem"
	keyFile  = "./ssl-cert-snakeoil.key"
)

func newClient(ln *InmemoryListener, isTLS bool) *fasthttp.HostClient {
	return &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		IsTLS: isTLS,
	}
}

func runRequests(b *testing.B, pb *testing.PB, c *fasthttp.HostClient) {
	var req fasthttp.Request
	req.SetRequestURI("http://foo.bar/baz")
	var resp fasthttp.Response
	for pb.Next() {
		if err := c.Do(&req, &resp); err != nil {
			b.Fatalf("unexpected error: %s", err)
		}
		if resp.StatusCode() != fasthttp.StatusOK {
			b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
		}
	}
}
