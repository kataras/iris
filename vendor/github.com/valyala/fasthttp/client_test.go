package fasthttp

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp/fasthttputil"
)

func TestPipelineClientDoSerial(t *testing.T) {
	testPipelineClientDoConcurrent(t, 1, 0)
}

func TestPipelineClientDoConcurrent(t *testing.T) {
	testPipelineClientDoConcurrent(t, 10, 0)
}

func TestPipelineClientDoBatchDelayConcurrent(t *testing.T) {
	testPipelineClientDoConcurrent(t, 10, 5*time.Millisecond)
}

func testPipelineClientDoConcurrent(t *testing.T, concurrency int, maxBatchDelay time.Duration) {
	ln := fasthttputil.NewInmemoryListener()

	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.WriteString("OK")
		},
	}

	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	c := &PipelineClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		MaxIdleConnDuration: 23 * time.Millisecond,
		MaxPendingRequests:  6,
		MaxBatchDelay:       maxBatchDelay,
		Logger:              &customLogger{},
	}

	clientStopCh := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testPipelineClientDo(t, c)
			clientStopCh <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-clientStopCh:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout")
		}
	}

	if c.PendingRequests() != 0 {
		t.Fatalf("unexpected number of pending requests: %d. Expecting zero", c.PendingRequests())
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	select {
	case <-serverStopCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func testPipelineClientDo(t *testing.T, c *PipelineClient) {
	var err error
	req := AcquireRequest()
	req.SetRequestURI("http://foobar/baz")
	resp := AcquireResponse()
	for i := 0; i < 10; i++ {
		if i&1 == 0 {
			err = c.DoTimeout(req, resp, time.Second)
		} else {
			err = c.Do(req, resp)
		}
		if err != nil {
			if err == ErrPipelineOverflow {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			t.Fatalf("unexpected error on iteration %d: %s", i, err)
		}
		if resp.StatusCode() != StatusOK {
			t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), StatusOK)
		}
		body := string(resp.Body())
		if body != "OK" {
			t.Fatalf("unexpected body: %q. Expecting %q", body, "OK")
		}

		// sleep for a while, so the connection to the host may expire.
		if i%5 == 0 {
			time.Sleep(30 * time.Millisecond)
		}
	}
	ReleaseRequest(req)
	ReleaseResponse(resp)
}

func TestClientDoTimeoutDisableNormalizing(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()

	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.Response.Header.Set("foo-BAR", "baz")
		},
		DisableHeaderNamesNormalizing: true,
	}

	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		DisableHeaderNamesNormalizing: true,
	}

	var req Request
	req.SetRequestURI("http://aaaai.com/bsdf?sddfsd")
	var resp Response
	for i := 0; i < 5; i++ {
		if err := c.DoTimeout(&req, &resp, time.Second); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		hv := resp.Header.Peek("foo-BAR")
		if string(hv) != "baz" {
			t.Fatalf("unexpected header value: %q. Expecting %q", hv, "baz")
		}
		hv = resp.Header.Peek("Foo-Bar")
		if len(hv) > 0 {
			t.Fatalf("unexpected non-empty header value %q", hv)
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	select {
	case <-serverStopCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestHostClientMaxConnDuration(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()

	connectionCloseCount := uint32(0)
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.WriteString("abcd")
			if ctx.Request.ConnectionClose() {
				atomic.AddUint32(&connectionCloseCount, 1)
			}
		},
	}
	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	c := &HostClient{
		Addr: "foobar",
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		MaxConnDuration: 10 * time.Millisecond,
	}

	for i := 0; i < 5; i++ {
		statusCode, body, err := c.Get(nil, "http://aaaa.com/bbb/cc")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code %d. Expecting %d", statusCode, StatusOK)
		}
		if string(body) != "abcd" {
			t.Fatalf("unexpected body %q. Expecting %q", body, "abcd")
		}
		time.Sleep(c.MaxConnDuration)
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	select {
	case <-serverStopCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}

	if connectionCloseCount == 0 {
		t.Fatalf("expecting at least one 'Connection: close' request header")
	}
}

func TestHostClientMultipleAddrs(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()

	s := &Server{
		Handler: func(ctx *RequestCtx) {
			ctx.Write(ctx.Host())
			ctx.SetConnectionClose()
		},
	}
	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	dialsCount := make(map[string]int)
	c := &HostClient{
		Addr: "foo,bar,baz",
		Dial: func(addr string) (net.Conn, error) {
			dialsCount[addr]++
			return ln.Dial()
		},
	}

	for i := 0; i < 9; i++ {
		statusCode, body, err := c.Get(nil, "http://foobar/baz/aaa?bbb=ddd")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code %d. Expecting %d", statusCode, StatusOK)
		}
		if string(body) != "foobar" {
			t.Fatalf("unexpected body %q. Expecting %q", body, "foobar")
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	select {
	case <-serverStopCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}

	if len(dialsCount) != 3 {
		t.Fatalf("unexpected dialsCount size %d. Expecting 3", len(dialsCount))
	}
	for _, k := range []string{"foo", "bar", "baz"} {
		if dialsCount[k] != 3 {
			t.Fatalf("unexpected dialsCount for %q. Expecting 3", k)
		}
	}
}

func TestClientFollowRedirects(t *testing.T) {
	addr := "127.0.0.1:55234"
	s := &Server{
		Handler: func(ctx *RequestCtx) {
			switch string(ctx.Path()) {
			case "/foo":
				u := ctx.URI()
				u.Update("/xy?z=wer")
				ctx.Redirect(u.String(), StatusFound)
			case "/xy":
				u := ctx.URI()
				u.Update("/bar")
				ctx.Redirect(u.String(), StatusFound)
			default:
				ctx.Success("text/plain", ctx.Path())
			}
		},
	}
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	uri := fmt.Sprintf("http://%s/foo", addr)
	for i := 0; i < 10; i++ {
		statusCode, body, err := GetTimeout(nil, uri, time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code: %d", statusCode)
		}
		if string(body) != "/bar" {
			t.Fatalf("unexpected response %q. Expecting %q", body, "/bar")
		}
	}

	uri = fmt.Sprintf("http://%s/aaab/sss", addr)
	for i := 0; i < 10; i++ {
		statusCode, body, err := Get(nil, uri)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code: %d", statusCode)
		}
		if string(body) != "/aaab/sss" {
			t.Fatalf("unexpected response %q. Expecting %q", body, "/aaab/sss")
		}
	}
}

func TestClientGetTimeoutSuccess(t *testing.T) {
	addr := "127.0.0.1:56889"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	testClientGetTimeoutSuccess(t, &defaultClient, addr, 100)
}

func TestClientGetTimeoutSuccessConcurrent(t *testing.T) {
	addr := "127.0.0.1:56989"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testClientGetTimeoutSuccess(t, &defaultClient, addr, 100)
		}()
	}
	wg.Wait()
}

func TestClientDoTimeoutSuccess(t *testing.T) {
	addr := "127.0.0.1:63897"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	testClientDoTimeoutSuccess(t, &defaultClient, addr, 100)
}

func TestClientDoTimeoutSuccessConcurrent(t *testing.T) {
	addr := "127.0.0.1:63898"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testClientDoTimeoutSuccess(t, &defaultClient, addr, 100)
		}()
	}
	wg.Wait()
}

func TestClientGetTimeoutError(t *testing.T) {
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return &readTimeoutConn{t: time.Second}, nil
		},
	}

	testClientGetTimeoutError(t, c, 100)
}

func TestClientGetTimeoutErrorConcurrent(t *testing.T) {
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return &readTimeoutConn{t: time.Second}, nil
		},
		MaxConnsPerHost: 1000,
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testClientGetTimeoutError(t, c, 100)
		}()
	}
	wg.Wait()
}

func TestClientDoTimeoutError(t *testing.T) {
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return &readTimeoutConn{t: time.Second}, nil
		},
	}

	testClientDoTimeoutError(t, c, 100)
}

func TestClientDoTimeoutErrorConcurrent(t *testing.T) {
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			return &readTimeoutConn{t: time.Second}, nil
		},
		MaxConnsPerHost: 1000,
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testClientDoTimeoutError(t, c, 100)
		}()
	}
	wg.Wait()
}

func testClientDoTimeoutError(t *testing.T, c *Client, n int) {
	var req Request
	var resp Response
	req.SetRequestURI("http://foobar.com/baz")
	for i := 0; i < n; i++ {
		err := c.DoTimeout(&req, &resp, time.Millisecond)
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err != ErrTimeout {
			t.Fatalf("unexpected error: %s. Expecting %s", err, ErrTimeout)
		}
	}
}

func testClientGetTimeoutError(t *testing.T, c *Client, n int) {
	buf := make([]byte, 10)
	for i := 0; i < n; i++ {
		statusCode, body, err := c.GetTimeout(buf, "http://foobar.com/baz", time.Millisecond)
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err != ErrTimeout {
			t.Fatalf("unexpected error: %s. Expecting %s", err, ErrTimeout)
		}
		if statusCode != 0 {
			t.Fatalf("unexpected statusCode=%d. Expecting %d", statusCode, 0)
		}
		if body == nil {
			t.Fatalf("body must be non-nil")
		}
	}
}

type readTimeoutConn struct {
	net.Conn
	t time.Duration
}

func (r *readTimeoutConn) Read(p []byte) (int, error) {
	time.Sleep(r.t)
	return 0, io.EOF
}

func (r *readTimeoutConn) Write(p []byte) (int, error) {
	return len(p), nil
}

func (r *readTimeoutConn) Close() error {
	return nil
}

func TestClientIdempotentRequest(t *testing.T) {
	dialsCount := 0
	c := &Client{
		Dial: func(addr string) (net.Conn, error) {
			switch dialsCount {
			case 0:
				dialsCount++
				return &readErrorConn{}, nil
			case 1:
				dialsCount++
				return &singleReadConn{
					s: "HTTP/1.1 345 OK\r\nContent-Type: foobar\r\nContent-Length: 7\r\n\r\n0123456",
				}, nil
			default:
				t.Fatalf("unexpected number of dials: %d", dialsCount)
			}
			panic("unreachable")
		},
	}

	statusCode, body, err := c.Get(nil, "http://foobar/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if statusCode != 345 {
		t.Fatalf("unexpected status code: %d. Expecting 345", statusCode)
	}
	if string(body) != "0123456" {
		t.Fatalf("unexpected body: %q. Expecting %q", body, "0123456")
	}

	var args Args

	dialsCount = 0
	statusCode, body, err = c.Post(nil, "http://foobar/a/b", &args)
	if err == nil {
		t.Fatalf("expecting error")
	}

	dialsCount = 0
	statusCode, body, err = c.Post(nil, "http://foobar/a/b", nil)
	if err == nil {
		t.Fatalf("expecting error")
	}
}

type readErrorConn struct {
	net.Conn
}

func (r *readErrorConn) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("error")
}

func (r *readErrorConn) Write(p []byte) (int, error) {
	return len(p), nil
}

func (r *readErrorConn) Close() error {
	return nil
}

type singleReadConn struct {
	net.Conn
	s string
	n int
}

func (r *singleReadConn) Read(p []byte) (int, error) {
	if len(r.s) == r.n {
		return 0, io.EOF
	}
	n := copy(p, []byte(r.s[r.n:]))
	r.n += n
	return n, nil
}

func (r *singleReadConn) Write(p []byte) (int, error) {
	return len(p), nil
}

func (r *singleReadConn) Close() error {
	return nil
}

func TestClientHTTPSConcurrent(t *testing.T) {
	addrHTTP := "127.0.0.1:56793"
	sHTTP := startEchoServer(t, "tcp", addrHTTP)
	defer sHTTP.Stop()

	addrHTTPS := "127.0.0.1:56794"
	sHTTPS := startEchoServerTLS(t, "tcp", addrHTTPS)
	defer sHTTPS.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		addr := "http://" + addrHTTP
		if i&1 != 0 {
			addr = "https://" + addrHTTPS
		}
		go func() {
			defer wg.Done()
			testClientGet(t, &defaultClient, addr, 20)
			testClientPost(t, &defaultClient, addr, 10)
		}()
	}
	wg.Wait()
}

func TestClientManyServers(t *testing.T) {
	var addrs []string
	for i := 0; i < 10; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", 56904+i)
		s := startEchoServer(t, "tcp", addr)
		defer s.Stop()
		addrs = append(addrs, addr)
	}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		addr := "http://" + addrs[i]
		go func() {
			defer wg.Done()
			testClientGet(t, &defaultClient, addr, 20)
			testClientPost(t, &defaultClient, addr, 10)
		}()
	}
	wg.Wait()
}

func TestClientGet(t *testing.T) {
	addr := "127.0.0.1:56789"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	testClientGet(t, &defaultClient, addr, 100)
}

func TestClientPost(t *testing.T) {
	addr := "127.0.0.1:56798"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	testClientPost(t, &defaultClient, addr, 100)
}

func TestClientConcurrent(t *testing.T) {
	addr := "127.0.0.1:55780"
	s := startEchoServer(t, "tcp", addr)
	defer s.Stop()

	addr = "http://" + addr
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testClientGet(t, &defaultClient, addr, 30)
			testClientPost(t, &defaultClient, addr, 10)
		}()
	}
	wg.Wait()
}

func TestHostClientGet(t *testing.T) {
	addr := "./TestHostClientGet.unix"
	s := startEchoServer(t, "unix", addr)
	defer s.Stop()
	c := createEchoClient(t, "unix", addr)

	testHostClientGet(t, c, 100)
}

func TestHostClientPost(t *testing.T) {
	addr := "./TestHostClientPost.unix"
	s := startEchoServer(t, "unix", addr)
	defer s.Stop()
	c := createEchoClient(t, "unix", addr)

	testHostClientPost(t, c, 100)
}

func TestHostClientConcurrent(t *testing.T) {
	addr := "./TestHostClientConcurrent.unix"
	s := startEchoServer(t, "unix", addr)
	defer s.Stop()
	c := createEchoClient(t, "unix", addr)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testHostClientGet(t, c, 30)
			testHostClientPost(t, c, 10)
		}()
	}
	wg.Wait()
}

func testClientGet(t *testing.T, c clientGetter, addr string, n int) {
	var buf []byte
	for i := 0; i < n; i++ {
		uri := fmt.Sprintf("%s/foo/%d?bar=baz", addr, i)
		statusCode, body, err := c.Get(buf, uri)
		buf = body
		if err != nil {
			t.Fatalf("unexpected error when doing http request: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code: %d. Expecting %d", statusCode, StatusOK)
		}
		resultURI := string(body)
		if strings.HasPrefix(uri, "https") {
			resultURI = uri[:5] + resultURI[4:]
		}
		if resultURI != uri {
			t.Fatalf("unexpected uri %q. Expecting %q", resultURI, uri)
		}
	}
}

func testClientDoTimeoutSuccess(t *testing.T, c *Client, addr string, n int) {
	var req Request
	var resp Response

	for i := 0; i < n; i++ {
		uri := fmt.Sprintf("%s/foo/%d?bar=baz", addr, i)
		req.SetRequestURI(uri)
		if err := c.DoTimeout(&req, &resp, time.Second); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if resp.StatusCode() != StatusOK {
			t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), StatusOK)
		}
		resultURI := string(resp.Body())
		if strings.HasPrefix(uri, "https") {
			resultURI = uri[:5] + resultURI[4:]
		}
		if resultURI != uri {
			t.Fatalf("unexpected uri %q. Expecting %q", resultURI, uri)
		}
	}
}

func testClientGetTimeoutSuccess(t *testing.T, c *Client, addr string, n int) {
	var buf []byte
	for i := 0; i < n; i++ {
		uri := fmt.Sprintf("%s/foo/%d?bar=baz", addr, i)
		statusCode, body, err := c.GetTimeout(buf, uri, time.Second)
		buf = body
		if err != nil {
			t.Fatalf("unexpected error when doing http request: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code: %d. Expecting %d", statusCode, StatusOK)
		}
		resultURI := string(body)
		if strings.HasPrefix(uri, "https") {
			resultURI = uri[:5] + resultURI[4:]
		}
		if resultURI != uri {
			t.Fatalf("unexpected uri %q. Expecting %q", resultURI, uri)
		}
	}
}

func testClientPost(t *testing.T, c clientPoster, addr string, n int) {
	var buf []byte
	var args Args
	for i := 0; i < n; i++ {
		uri := fmt.Sprintf("%s/foo/%d?bar=baz", addr, i)
		args.Set("xx", fmt.Sprintf("yy%d", i))
		args.Set("zzz", fmt.Sprintf("qwe_%d", i))
		argsS := args.String()
		statusCode, body, err := c.Post(buf, uri, &args)
		buf = body
		if err != nil {
			t.Fatalf("unexpected error when doing http request: %s", err)
		}
		if statusCode != StatusOK {
			t.Fatalf("unexpected status code: %d. Expecting %d", statusCode, StatusOK)
		}
		s := string(body)
		if s != argsS {
			t.Fatalf("unexpected response %q. Expecting %q", s, argsS)
		}
	}
}

func testHostClientGet(t *testing.T, c *HostClient, n int) {
	testClientGet(t, c, "http://google.com", n)
}

func testHostClientPost(t *testing.T, c *HostClient, n int) {
	testClientPost(t, c, "http://post-host.com", n)
}

type clientPoster interface {
	Post(dst []byte, uri string, postArgs *Args) (int, []byte, error)
}

type clientGetter interface {
	Get(dst []byte, uri string) (int, []byte, error)
}

func createEchoClient(t *testing.T, network, addr string) *HostClient {
	return &HostClient{
		Addr: addr,
		Dial: func(addr string) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
}

type testEchoServer struct {
	s  *Server
	ln net.Listener
	ch chan struct{}
	t  *testing.T
}

func (s *testEchoServer) Stop() {
	s.ln.Close()
	select {
	case <-s.ch:
	case <-time.After(time.Second):
		s.t.Fatalf("timeout when waiting for server close")
	}
}

func startEchoServerTLS(t *testing.T, network, addr string) *testEchoServer {
	return startEchoServerExt(t, network, addr, true)
}

func startEchoServer(t *testing.T, network, addr string) *testEchoServer {
	return startEchoServerExt(t, network, addr, false)
}

func startEchoServerExt(t *testing.T, network, addr string, isTLS bool) *testEchoServer {
	if network == "unix" {
		os.Remove(addr)
	}
	var ln net.Listener
	var err error
	if isTLS {
		certFile := "./ssl-cert-snakeoil.pem"
		keyFile := "./ssl-cert-snakeoil.key"
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			t.Fatalf("Cannot load TLS certificate: %s", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		ln, err = tls.Listen(network, addr, tlsConfig)
	} else {
		ln, err = net.Listen(network, addr)
	}
	if err != nil {
		t.Fatalf("cannot listen %q: %s", addr, err)
	}

	s := &Server{
		Handler: func(ctx *RequestCtx) {
			if ctx.IsGet() {
				ctx.Success("text/plain", ctx.URI().FullURI())
			} else if ctx.IsPost() {
				ctx.PostArgs().WriteTo(ctx)
			}
		},
	}
	ch := make(chan struct{})
	go func() {
		err := s.Serve(ln)
		if err != nil {
			t.Fatalf("unexpected error returned from Serve(): %s", err)
		}
		close(ch)
	}()
	return &testEchoServer{
		s:  s,
		ln: ln,
		ch: ch,
		t:  t,
	}
}
