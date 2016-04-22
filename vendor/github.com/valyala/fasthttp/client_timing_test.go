package fasthttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp/fasthttputil"
)

type fakeClientConn struct {
	net.Conn
	s  []byte
	n  int
	ch chan struct{}
}

func (c *fakeClientConn) Write(b []byte) (int, error) {
	c.ch <- struct{}{}
	return len(b), nil
}

func (c *fakeClientConn) Read(b []byte) (int, error) {
	if c.n == 0 {
		// wait for request :)
		<-c.ch
	}
	n := 0
	for len(b) > 0 {
		if c.n == len(c.s) {
			c.n = 0
			return n, nil
		}
		n = copy(b, c.s[c.n:])
		c.n += n
		b = b[n:]
	}
	return n, nil
}

func (c *fakeClientConn) Close() error {
	releaseFakeServerConn(c)
	return nil
}

func releaseFakeServerConn(c *fakeClientConn) {
	c.n = 0
	fakeClientConnPool.Put(c)
}

func acquireFakeServerConn(s []byte) *fakeClientConn {
	v := fakeClientConnPool.Get()
	if v == nil {
		c := &fakeClientConn{
			s:  s,
			ch: make(chan struct{}, 1),
		}
		return c
	}
	return v.(*fakeClientConn)
}

var fakeClientConnPool sync.Pool

func BenchmarkClientGetTimeoutFastServer(b *testing.B) {
	body := []byte("123456789099")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
	}

	nn := uint32(0)
	b.RunParallel(func(pb *testing.PB) {
		url := fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1))
		var statusCode int
		var bodyBuf []byte
		var err error
		for pb.Next() {
			statusCode, bodyBuf, err = c.GetTimeout(bodyBuf[:0], url, time.Second)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if statusCode != StatusOK {
				b.Fatalf("unexpected status code: %d", statusCode)
			}
			if !bytes.Equal(bodyBuf, body) {
				b.Fatalf("unexpected response body: %q. Expected %q", bodyBuf, body)
			}
		}
	})
}

func BenchmarkClientDoFastServer(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
		MaxConnsPerHost: runtime.GOMAXPROCS(-1),
	}

	nn := uint32(0)
	b.RunParallel(func(pb *testing.PB) {
		var req Request
		var resp Response
		req.Header.SetRequestURI(fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1)))
		for pb.Next() {
			if err := c.Do(&req, &resp); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.Header.StatusCode() != StatusOK {
				b.Fatalf("unexpected status code: %d", resp.Header.StatusCode())
			}
			if !bytes.Equal(resp.Body(), body) {
				b.Fatalf("unexpected response body: %q. Expected %q", resp.Body(), body)
			}
		}
	})
}

func BenchmarkNetHTTPClientDoFastServer(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return acquireFakeServerConn(s), nil
			},
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(-1),
		},
	}

	nn := uint32(0)
	b.RunParallel(func(pb *testing.PB) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1)), nil)
		if err != nil {
			b.Fatalf("unexpected error: %s", err)
		}
		for pb.Next() {
			resp, err := c.Do(req)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.StatusCode)
			}
			respBody, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %s", err)
			}
			if !bytes.Equal(respBody, body) {
				b.Fatalf("unexpected response body: %q. Expected %q", respBody, body)
			}
		}
	})
}

func fasthttpEchoHandler(ctx *RequestCtx) {
	ctx.Success("text/plain", ctx.RequestURI())
}

func nethttpEchoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(r.RequestURI))
}

func BenchmarkClientGetEndToEnd1TCP(b *testing.B) {
	benchmarkClientGetEndToEndTCP(b, 1)
}

func BenchmarkClientGetEndToEnd10TCP(b *testing.B) {
	benchmarkClientGetEndToEndTCP(b, 10)
}

func BenchmarkClientGetEndToEnd100TCP(b *testing.B) {
	benchmarkClientGetEndToEndTCP(b, 100)
}

func benchmarkClientGetEndToEndTCP(b *testing.B, parallelism int) {
	addr := "127.0.0.1:8543"

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		b.Fatalf("cannot listen %q: %s", addr, err)
	}

	ch := make(chan struct{})
	go func() {
		if err := Serve(ln, fasthttpEchoHandler); err != nil {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://" + addr + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var buf []byte
		for pb.Next() {
			statusCode, body, err := c.Get(buf, url)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if statusCode != StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", statusCode, StatusOK)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
			buf = body
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkNetHTTPClientGetEndToEnd1TCP(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndTCP(b, 1)
}

func BenchmarkNetHTTPClientGetEndToEnd10TCP(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndTCP(b, 10)
}

func BenchmarkNetHTTPClientGetEndToEnd100TCP(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndTCP(b, 100)
}

func benchmarkNetHTTPClientGetEndToEndTCP(b *testing.B, parallelism int) {
	addr := "127.0.0.1:8542"

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		b.Fatalf("cannot listen %q: %s", addr, err)
	}

	ch := make(chan struct{})
	go func() {
		if err := http.Serve(ln, http.HandlerFunc(nethttpEchoHandler)); err != nil && !strings.Contains(
			err.Error(), "use of closed network connection") {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: parallelism * runtime.GOMAXPROCS(-1),
		},
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://" + addr + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Get(url)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode, http.StatusOK)
			}
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %s", err)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkClientGetEndToEnd1Inmemory(b *testing.B) {
	benchmarkClientGetEndToEndInmemory(b, 1)
}

func BenchmarkClientGetEndToEnd10Inmemory(b *testing.B) {
	benchmarkClientGetEndToEndInmemory(b, 10)
}

func BenchmarkClientGetEndToEnd100Inmemory(b *testing.B) {
	benchmarkClientGetEndToEndInmemory(b, 100)
}

func BenchmarkClientGetEndToEnd1000Inmemory(b *testing.B) {
	benchmarkClientGetEndToEndInmemory(b, 1000)
}

func BenchmarkClientGetEndToEnd10KInmemory(b *testing.B) {
	benchmarkClientGetEndToEndInmemory(b, 10000)
}

func benchmarkClientGetEndToEndInmemory(b *testing.B, parallelism int) {
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := Serve(ln, fasthttpEchoHandler); err != nil {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
		Dial:            func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var buf []byte
		for pb.Next() {
			statusCode, body, err := c.Get(buf, url)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if statusCode != StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", statusCode, StatusOK)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
			buf = body
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkNetHTTPClientGetEndToEnd1Inmemory(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndInmemory(b, 1)
}

func BenchmarkNetHTTPClientGetEndToEnd10Inmemory(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndInmemory(b, 10)
}

func BenchmarkNetHTTPClientGetEndToEnd100Inmemory(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndInmemory(b, 100)
}

func BenchmarkNetHTTPClientGetEndToEnd1000Inmemory(b *testing.B) {
	benchmarkNetHTTPClientGetEndToEndInmemory(b, 1000)
}

func benchmarkNetHTTPClientGetEndToEndInmemory(b *testing.B, parallelism int) {
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := http.Serve(ln, http.HandlerFunc(nethttpEchoHandler)); err != nil && !strings.Contains(
			err.Error(), "use of closed network connection") {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &http.Client{
		Transport: &http.Transport{
			Dial:                func(_, _ string) (net.Conn, error) { return ln.Dial() },
			MaxIdleConnsPerHost: parallelism * runtime.GOMAXPROCS(-1),
		},
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Get(url)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode, http.StatusOK)
			}
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %s", err)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkClientEndToEndBigResponse1Inmemory(b *testing.B) {
	benchmarkClientEndToEndBigResponseInmemory(b, 1)
}

func BenchmarkClientEndToEndBigResponse10Inmemory(b *testing.B) {
	benchmarkClientEndToEndBigResponseInmemory(b, 10)
}

func benchmarkClientEndToEndBigResponseInmemory(b *testing.B, parallelism int) {
	bigResponse := createFixedBody(1024 * 1024)
	h := func(ctx *RequestCtx) {
		ctx.SetContentType("text/plain")
		ctx.Write(bigResponse)
	}

	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := Serve(ln, h); err != nil {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
		Dial:            func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var req Request
		req.SetRequestURI(url)
		var resp Response
		for pb.Next() {
			if err := c.DoTimeout(&req, &resp, 5*time.Second); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode() != StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), StatusOK)
			}
			body := resp.Body()
			if !bytes.Equal(bigResponse, body) {
				b.Fatalf("unexpected response %q. Expecting %q", body, bigResponse)
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkNetHTTPClientEndToEndBigResponse1Inmemory(b *testing.B) {
	benchmarkNetHTTPClientEndToEndBigResponseInmemory(b, 1)
}

func BenchmarkNetHTTPClientEndToEndBigResponse10Inmemory(b *testing.B) {
	benchmarkNetHTTPClientEndToEndBigResponseInmemory(b, 10)
}

func benchmarkNetHTTPClientEndToEndBigResponseInmemory(b *testing.B, parallelism int) {
	bigResponse := createFixedBody(1024 * 1024)
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(bigResponse)
	}
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := http.Serve(ln, http.HandlerFunc(h)); err != nil && !strings.Contains(
			err.Error(), "use of closed network connection") {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	c := &http.Client{
		Transport: &http.Transport{
			Dial:                func(_, _ string) (net.Conn, error) { return ln.Dial() },
			MaxIdleConnsPerHost: parallelism * runtime.GOMAXPROCS(-1),
		},
		Timeout: 5 * time.Second,
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			b.Fatalf("unexpected error: %s", err)
		}
		for pb.Next() {
			resp, err := c.Do(req)
			if err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode, http.StatusOK)
			}
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %s", err)
			}
			if !bytes.Equal(bigResponse, body) {
				b.Fatalf("unexpected response %q. Expecting %q", body, bigResponse)
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func BenchmarkPipelineClient1(b *testing.B) {
	benchmarkPipelineClient(b, 1)
}

func BenchmarkPipelineClient10(b *testing.B) {
	benchmarkPipelineClient(b, 10)
}

func BenchmarkPipelineClient100(b *testing.B) {
	benchmarkPipelineClient(b, 100)
}

func BenchmarkPipelineClient1000(b *testing.B) {
	benchmarkPipelineClient(b, 1000)
}

func benchmarkPipelineClient(b *testing.B, parallelism int) {
	h := func(ctx *RequestCtx) {
		ctx.WriteString("foobar")
	}
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := Serve(ln, h); err != nil {
			b.Fatalf("error when serving requests: %s", err)
		}
		close(ch)
	}()

	var clients []*PipelineClient
	for i := 0; i < runtime.GOMAXPROCS(-1); i++ {
		c := &PipelineClient{
			Dial:               func(addr string) (net.Conn, error) { return ln.Dial() },
			ReadBufferSize:     1024 * 1024,
			WriteBufferSize:    1024 * 1024,
			MaxPendingRequests: parallelism,
		}
		clients = append(clients, c)
	}

	clientID := uint32(0)
	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		n := atomic.AddUint32(&clientID, 1)
		c := clients[n%uint32(len(clients))]
		var req Request
		req.SetRequestURI(url)
		var resp Response
		for pb.Next() {
			if err := c.Do(&req, &resp); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			if resp.StatusCode() != StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), StatusOK)
			}
			body := resp.Body()
			if string(body) != "foobar" {
				b.Fatalf("unexpected response %q. Expecting %q", body, "foobar")
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}
