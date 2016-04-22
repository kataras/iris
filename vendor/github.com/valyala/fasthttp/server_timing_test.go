package fasthttp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var defaultClientsCount = runtime.NumCPU()

func BenchmarkServerGet1ReqPerConn(b *testing.B) {
	benchmarkServerGet(b, defaultClientsCount, 1)
}

func BenchmarkServerGet2ReqPerConn(b *testing.B) {
	benchmarkServerGet(b, defaultClientsCount, 2)
}

func BenchmarkServerGet10ReqPerConn(b *testing.B) {
	benchmarkServerGet(b, defaultClientsCount, 10)
}

func BenchmarkServerGet10KReqPerConn(b *testing.B) {
	benchmarkServerGet(b, defaultClientsCount, 10000)
}

func BenchmarkNetHTTPServerGet1ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerGet(b, defaultClientsCount, 1)
}

func BenchmarkNetHTTPServerGet2ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerGet(b, defaultClientsCount, 2)
}

func BenchmarkNetHTTPServerGet10ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerGet(b, defaultClientsCount, 10)
}

func BenchmarkNetHTTPServerGet10KReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerGet(b, defaultClientsCount, 10000)
}

func BenchmarkServerPost1ReqPerConn(b *testing.B) {
	benchmarkServerPost(b, defaultClientsCount, 1)
}

func BenchmarkServerPost2ReqPerConn(b *testing.B) {
	benchmarkServerPost(b, defaultClientsCount, 2)
}

func BenchmarkServerPost10ReqPerConn(b *testing.B) {
	benchmarkServerPost(b, defaultClientsCount, 10)
}

func BenchmarkServerPost10KReqPerConn(b *testing.B) {
	benchmarkServerPost(b, defaultClientsCount, 10000)
}

func BenchmarkNetHTTPServerPost1ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerPost(b, defaultClientsCount, 1)
}

func BenchmarkNetHTTPServerPost2ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerPost(b, defaultClientsCount, 2)
}

func BenchmarkNetHTTPServerPost10ReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerPost(b, defaultClientsCount, 10)
}

func BenchmarkNetHTTPServerPost10KReqPerConn(b *testing.B) {
	benchmarkNetHTTPServerPost(b, defaultClientsCount, 10000)
}

func BenchmarkServerGet1ReqPerConn10KClients(b *testing.B) {
	benchmarkServerGet(b, 10000, 1)
}

func BenchmarkServerGet2ReqPerConn10KClients(b *testing.B) {
	benchmarkServerGet(b, 10000, 2)
}

func BenchmarkServerGet10ReqPerConn10KClients(b *testing.B) {
	benchmarkServerGet(b, 10000, 10)
}

func BenchmarkServerGet100ReqPerConn10KClients(b *testing.B) {
	benchmarkServerGet(b, 10000, 100)
}

func BenchmarkNetHTTPServerGet1ReqPerConn10KClients(b *testing.B) {
	benchmarkNetHTTPServerGet(b, 10000, 1)
}

func BenchmarkNetHTTPServerGet2ReqPerConn10KClients(b *testing.B) {
	benchmarkNetHTTPServerGet(b, 10000, 2)
}

func BenchmarkNetHTTPServerGet10ReqPerConn10KClients(b *testing.B) {
	benchmarkNetHTTPServerGet(b, 10000, 10)
}

func BenchmarkNetHTTPServerGet100ReqPerConn10KClients(b *testing.B) {
	benchmarkNetHTTPServerGet(b, 10000, 100)
}

func BenchmarkServerHijack(b *testing.B) {
	clientsCount := 1000
	requestsPerConn := 10000
	ch := make(chan struct{}, b.N)
	responseBody := []byte("123")
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.Hijack(func(c net.Conn) {
				// emulate server loop :)
				err := ServeConn(c, func(ctx *RequestCtx) {
					ctx.Success("foobar", responseBody)
					registerServedRequest(b, ch)
				})
				if err != nil {
					b.Fatalf("error when serving connection")
				}
			})
			ctx.Success("foobar", responseBody)
			registerServedRequest(b, ch)
		},
		Concurrency: 16 * clientsCount,
	}
	req := "GET /foo HTTP/1.1\r\nHost: google.com\r\n\r\n"
	benchmarkServer(b, s, clientsCount, requestsPerConn, req)
	verifyRequestsServed(b, ch)
}

func BenchmarkServerMaxConnsPerIP(b *testing.B) {
	clientsCount := 1000
	requestsPerConn := 10
	ch := make(chan struct{}, b.N)
	responseBody := []byte("123")
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.Success("foobar", responseBody)
			registerServedRequest(b, ch)
		},
		MaxConnsPerIP: clientsCount * 2,
		Concurrency:   16 * clientsCount,
	}
	req := "GET /foo HTTP/1.1\r\nHost: google.com\r\n\r\n"
	benchmarkServer(b, s, clientsCount, requestsPerConn, req)
	verifyRequestsServed(b, ch)
}

func BenchmarkServerTimeoutError(b *testing.B) {
	clientsCount := 10
	requestsPerConn := 1
	ch := make(chan struct{}, b.N)
	n := uint32(0)
	responseBody := []byte("123")
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			if atomic.AddUint32(&n, 1)&7 == 0 {
				ctx.TimeoutError("xxx")
				go func() {
					ctx.Success("foobar", responseBody)
				}()
			} else {
				ctx.Success("foobar", responseBody)
			}
			registerServedRequest(b, ch)
		},
		Concurrency: 16 * clientsCount,
	}
	req := "GET /foo HTTP/1.1\r\nHost: google.com\r\n\r\n"
	benchmarkServer(b, s, clientsCount, requestsPerConn, req)
	verifyRequestsServed(b, ch)
}

type fakeServerConn struct {
	net.TCPConn
	ln            *fakeListener
	requestsCount int
	pos           int
	closed        uint32
}

func (c *fakeServerConn) Read(b []byte) (int, error) {
	nn := 0
	reqLen := len(c.ln.request)
	for len(b) > 0 {
		if c.requestsCount == 0 {
			if nn == 0 {
				return 0, io.EOF
			}
			return nn, nil
		}
		pos := c.pos % reqLen
		n := copy(b, c.ln.request[pos:])
		b = b[n:]
		nn += n
		c.pos += n
		if n+pos == reqLen {
			c.requestsCount--
		}
	}
	return nn, nil
}

func (c *fakeServerConn) Write(b []byte) (int, error) {
	return len(b), nil
}

var fakeAddr = net.TCPAddr{
	IP:   []byte{1, 2, 3, 4},
	Port: 12345,
}

func (c *fakeServerConn) RemoteAddr() net.Addr {
	return &fakeAddr
}

func (c *fakeServerConn) Close() error {
	if atomic.AddUint32(&c.closed, 1) == 1 {
		c.ln.ch <- c
	}
	return nil
}

func (c *fakeServerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeServerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type fakeListener struct {
	lock            sync.Mutex
	requestsCount   int
	requestsPerConn int
	request         []byte
	ch              chan *fakeServerConn
	done            chan struct{}
	closed          bool
}

func (ln *fakeListener) Accept() (net.Conn, error) {
	ln.lock.Lock()
	if ln.requestsCount == 0 {
		ln.lock.Unlock()
		for len(ln.ch) < cap(ln.ch) {
			time.Sleep(10 * time.Millisecond)
		}
		ln.lock.Lock()
		if !ln.closed {
			close(ln.done)
			ln.closed = true
		}
		ln.lock.Unlock()
		return nil, io.EOF
	}
	requestsCount := ln.requestsPerConn
	if requestsCount > ln.requestsCount {
		requestsCount = ln.requestsCount
	}
	ln.requestsCount -= requestsCount
	ln.lock.Unlock()

	c := <-ln.ch
	c.requestsCount = requestsCount
	c.closed = 0
	c.pos = 0

	return c, nil
}

func (ln *fakeListener) Close() error {
	return nil
}

func (ln *fakeListener) Addr() net.Addr {
	return &fakeAddr
}

func newFakeListener(requestsCount, clientsCount, requestsPerConn int, request string) *fakeListener {
	ln := &fakeListener{
		requestsCount:   requestsCount,
		requestsPerConn: requestsPerConn,
		request:         []byte(request),
		ch:              make(chan *fakeServerConn, clientsCount),
		done:            make(chan struct{}),
	}
	for i := 0; i < clientsCount; i++ {
		ln.ch <- &fakeServerConn{
			ln: ln,
		}
	}
	return ln
}

var (
	fakeResponse = []byte("Hello, world!")
	getRequest   = "GET /foobar?baz HTTP/1.1\r\nHost: google.com\r\nUser-Agent: aaa/bbb/ccc/ddd/eee Firefox Chrome MSIE Opera\r\n" +
		"Referer: http://xxx.com/aaa?bbb=ccc\r\nCookie: foo=bar; baz=baraz; aa=aakslsdweriwereowriewroire\r\n\r\n"
	postRequest = fmt.Sprintf("POST /foobar?baz HTTP/1.1\r\nHost: google.com\r\nContent-Type: foo/bar\r\nContent-Length: %d\r\n"+
		"User-Agent: Opera Chrome MSIE Firefox and other/1.2.34\r\nReferer: http://google.com/aaaa/bbb/ccc\r\n"+
		"Cookie: foo=bar; baz=baraz; aa=aakslsdweriwereowriewroire\r\n\r\n%s",
		len(fakeResponse), fakeResponse)
)

func benchmarkServerGet(b *testing.B, clientsCount, requestsPerConn int) {
	ch := make(chan struct{}, b.N)
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			if !ctx.IsGet() {
				b.Fatalf("Unexpected request method: %s", ctx.Method())
			}
			ctx.Success("text/plain", fakeResponse)
			if requestsPerConn == 1 {
				ctx.SetConnectionClose()
			}
			registerServedRequest(b, ch)
		},
		Concurrency: 16 * clientsCount,
	}
	benchmarkServer(b, s, clientsCount, requestsPerConn, getRequest)
	verifyRequestsServed(b, ch)
}

func benchmarkNetHTTPServerGet(b *testing.B, clientsCount, requestsPerConn int) {
	ch := make(chan struct{}, b.N)
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "GET" {
				b.Fatalf("Unexpected request method: %s", req.Method)
			}
			h := w.Header()
			h.Set("Content-Type", "text/plain")
			if requestsPerConn == 1 {
				h.Set("Connection", "close")
			}
			w.Write(fakeResponse)
			registerServedRequest(b, ch)
		}),
	}
	benchmarkServer(b, s, clientsCount, requestsPerConn, getRequest)
	verifyRequestsServed(b, ch)
}

func benchmarkServerPost(b *testing.B, clientsCount, requestsPerConn int) {
	ch := make(chan struct{}, b.N)
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			if !ctx.IsPost() {
				b.Fatalf("Unexpected request method: %s", ctx.Method())
			}
			body := ctx.Request.Body()
			if !bytes.Equal(body, fakeResponse) {
				b.Fatalf("Unexpected body %q. Expected %q", body, fakeResponse)
			}
			ctx.Success("text/plain", body)
			if requestsPerConn == 1 {
				ctx.SetConnectionClose()
			}
			registerServedRequest(b, ch)
		},
		Concurrency: 16 * clientsCount,
	}
	benchmarkServer(b, s, clientsCount, requestsPerConn, postRequest)
	verifyRequestsServed(b, ch)
}

func benchmarkNetHTTPServerPost(b *testing.B, clientsCount, requestsPerConn int) {
	ch := make(chan struct{}, b.N)
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "POST" {
				b.Fatalf("Unexpected request method: %s", req.Method)
			}
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				b.Fatalf("Unexpected error: %s", err)
			}
			req.Body.Close()
			if !bytes.Equal(body, fakeResponse) {
				b.Fatalf("Unexpected body %q. Expected %q", body, fakeResponse)
			}
			h := w.Header()
			h.Set("Content-Type", "text/plain")
			if requestsPerConn == 1 {
				h.Set("Connection", "close")
			}
			w.Write(body)
			registerServedRequest(b, ch)
		}),
	}
	benchmarkServer(b, s, clientsCount, requestsPerConn, postRequest)
	verifyRequestsServed(b, ch)
}

func registerServedRequest(b *testing.B, ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
		b.Fatalf("More than %d requests served", cap(ch))
	}
}

func verifyRequestsServed(b *testing.B, ch <-chan struct{}) {
	requestsServed := 0
	for len(ch) > 0 {
		<-ch
		requestsServed++
	}
	requestsSent := b.N
	for requestsServed < requestsSent {
		select {
		case <-ch:
			requestsServed++
		case <-time.After(100 * time.Millisecond):
			b.Fatalf("Unexpected number of requests served %d. Expected %d", requestsServed, requestsSent)
		}
	}
}

type realServer interface {
	Serve(ln net.Listener) error
}

func benchmarkServer(b *testing.B, s realServer, clientsCount, requestsPerConn int, request string) {
	ln := newFakeListener(b.N, clientsCount, requestsPerConn, request)
	ch := make(chan struct{})
	go func() {
		s.Serve(ln)
		ch <- struct{}{}
	}()

	<-ln.done

	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		b.Fatalf("Server.Serve() didn't stop")
	}
}
