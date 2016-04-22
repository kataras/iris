package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestResponseHeaderAdd(t *testing.T) {
	m := make(map[string]struct{})
	var h ResponseHeader
	h.Add("aaa", "bbb")
	m["bbb"] = struct{}{}
	for i := 0; i < 10; i++ {
		v := fmt.Sprintf("%d", i)
		h.Add("Foo-Bar", v)
		m[v] = struct{}{}
	}
	if h.Len() != 12 {
		t.Fatalf("unexpected header len %d. Expecting 12", h.Len())
	}

	h.VisitAll(func(k, v []byte) {
		switch string(k) {
		case "Aaa", "Foo-Bar":
			if _, ok := m[string(v)]; !ok {
				t.Fatalf("unexpected value found %q. key %q", v, k)
			}
			delete(m, string(v))
		case "Content-Type":
		default:
			t.Fatalf("unexpected key found: %q", k)
		}
	})
	if len(m) > 0 {
		t.Fatalf("%d headers are missed", len(m))
	}

	s := h.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	var h1 ResponseHeader
	if err := h1.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	h.VisitAll(func(k, v []byte) {
		switch string(k) {
		case "Aaa", "Foo-Bar":
			m[string(v)] = struct{}{}
		case "Content-Type":
		default:
			t.Fatalf("unexpected key found: %q", k)
		}
	})
	if len(m) != 11 {
		t.Fatalf("unexpected number of headers: %d. Expecting 11", len(m))
	}
}

func TestRequestHeaderAdd(t *testing.T) {
	m := make(map[string]struct{})
	var h RequestHeader
	h.Add("aaa", "bbb")
	m["bbb"] = struct{}{}
	for i := 0; i < 10; i++ {
		v := fmt.Sprintf("%d", i)
		h.Add("Foo-Bar", v)
		m[v] = struct{}{}
	}
	if h.Len() != 11 {
		t.Fatalf("unexpected header len %d. Expecting 11", h.Len())
	}

	h.VisitAll(func(k, v []byte) {
		switch string(k) {
		case "Aaa", "Foo-Bar":
			if _, ok := m[string(v)]; !ok {
				t.Fatalf("unexpected value found %q. key %q", v, k)
			}
			delete(m, string(v))
		default:
			t.Fatalf("unexpected key found: %q", k)
		}
	})
	if len(m) > 0 {
		t.Fatalf("%d headers are missed", len(m))
	}

	s := h.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	var h1 RequestHeader
	if err := h1.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	h.VisitAll(func(k, v []byte) {
		switch string(k) {
		case "Aaa", "Foo-Bar":
			m[string(v)] = struct{}{}
		case "User-Agent":
		default:
			t.Fatalf("unexpected key found: %q", k)
		}
	})
	if len(m) != 11 {
		t.Fatalf("unexpected number of headers: %d. Expecting 11", len(m))
	}
	s1 := h1.String()
	if s != s1 {
		t.Fatalf("unexpected headers %q. Expecting %q", s1, s)
	}
}

func TestHasHeaderValue(t *testing.T) {
	testHasHeaderValue(t, "foobar", "foobar", true)
	testHasHeaderValue(t, "foobar", "foo", false)
	testHasHeaderValue(t, "foobar", "bar", false)
	testHasHeaderValue(t, "keep-alive, Upgrade", "keep-alive", true)
	testHasHeaderValue(t, "keep-alive  ,    Upgrade", "Upgrade", true)
	testHasHeaderValue(t, "keep-alive, Upgrade", "Upgrade-foo", false)
	testHasHeaderValue(t, "keep-alive, Upgrade", "Upgr", false)
	testHasHeaderValue(t, "foo  ,   bar,  baz   ,", "foo", true)
	testHasHeaderValue(t, "foo  ,   bar,  baz   ,", "bar", true)
	testHasHeaderValue(t, "foo  ,   bar,  baz   ,", "baz", true)
	testHasHeaderValue(t, "foo  ,   bar,  baz   ,", "ba", false)
	testHasHeaderValue(t, "foo, ", "", true)
	testHasHeaderValue(t, "foo", "", false)
}

func testHasHeaderValue(t *testing.T, s, value string, has bool) {
	ok := hasHeaderValue([]byte(s), []byte(value))
	if ok != has {
		t.Fatalf("unexpected hasHeaderValue(%q, %q)=%v. Expecting %v", s, value, ok, has)
	}
}

func TestRequestHeaderDel(t *testing.T) {
	var h RequestHeader
	h.Set("Foo-Bar", "baz")
	h.Set("aaa", "bbb")
	h.Set("Connection", "keep-alive")
	h.Set("Content-Type", "aaa")
	h.Set("Host", "aaabbb")
	h.Set("User-Agent", "asdfas")
	h.Set("Content-Length", "1123")
	h.Set("Cookie", "foobar=baz")

	h.Del("foo-bar")
	h.Del("connection")
	h.DelBytes([]byte("content-type"))
	h.Del("Host")
	h.Del("user-agent")
	h.Del("content-length")
	h.Del("cookie")

	hv := h.Peek("aaa")
	if string(hv) != "bbb" {
		t.Fatalf("unexpected header value: %q. Expecting %q", hv, "bbb")
	}
	hv = h.Peek("Foo-Bar")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Connection")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Content-Type")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Host")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("User-Agent")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Content-Length")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Cookie")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}

	cv := h.Cookie("foobar")
	if len(cv) > 0 {
		t.Fatalf("unexpected cookie obtianed: %q", cv)
	}
	if h.ContentLength() != 0 {
		t.Fatalf("unexpected content-length: %d. Expecting 0", h.ContentLength())
	}
}

func TestResponseHeaderDel(t *testing.T) {
	var h ResponseHeader
	h.Set("Foo-Bar", "baz")
	h.Set("aaa", "bbb")
	h.Set("Connection", "keep-alive")
	h.Set("Content-Type", "aaa")
	h.Set("Server", "aaabbb")
	h.Set("Content-Length", "1123")

	var c Cookie
	c.SetKey("foo")
	c.SetValue("bar")
	h.SetCookie(&c)

	h.Del("foo-bar")
	h.Del("connection")
	h.DelBytes([]byte("content-type"))
	h.Del("Server")
	h.Del("content-length")
	h.Del("set-cookie")

	hv := h.Peek("aaa")
	if string(hv) != "bbb" {
		t.Fatalf("unexpected header value: %q. Expecting %q", hv, "bbb")
	}
	hv = h.Peek("Foo-Bar")
	if len(hv) > 0 {
		t.Fatalf("non-zero header value: %q", hv)
	}
	hv = h.Peek("Connection")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Content-Type")
	if string(hv) != string(defaultContentType) {
		t.Fatalf("unexpected content-type: %q. Expecting %q", hv, defaultContentType)
	}
	hv = h.Peek("Server")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}
	hv = h.Peek("Content-Length")
	if len(hv) > 0 {
		t.Fatalf("non-zero value: %q", hv)
	}

	if h.Cookie(&c) {
		t.Fatalf("unexpected cookie obtianed: %q", &c)
	}
	if h.ContentLength() != 0 {
		t.Fatalf("unexpected content-length: %d. Expecting 0", h.ContentLength())
	}
}

func TestAppendNormalizedHeaderKeyBytes(t *testing.T) {
	testAppendNormalizedHeaderKeyBytes(t, "", "")
	testAppendNormalizedHeaderKeyBytes(t, "Content-Type", "Content-Type")
	testAppendNormalizedHeaderKeyBytes(t, "foO-bAr-BAZ", "Foo-Bar-Baz")
}

func testAppendNormalizedHeaderKeyBytes(t *testing.T, key, expectedKey string) {
	buf := []byte("foobar")
	result := AppendNormalizedHeaderKeyBytes(buf, []byte(key))
	normalizedKey := result[len(buf):]
	if string(normalizedKey) != expectedKey {
		t.Fatalf("unexpected normalized key %q. Expecting %q", normalizedKey, expectedKey)
	}
}

func TestRequestHeaderHTTP10ConnectionClose(t *testing.T) {
	s := "GET / HTTP/1.0\r\nHost: foobar\r\n\r\n"
	var h RequestHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !h.connectionCloseFast() {
		t.Fatalf("expecting 'Connection: close' request header")
	}
	if !h.ConnectionClose() {
		t.Fatalf("expecting 'Connection: close' request header")
	}
}

func TestRequestHeaderHTTP10ConnectionKeepAlive(t *testing.T) {
	s := "GET / HTTP/1.0\r\nHost: foobar\r\nConnection: keep-alive\r\n\r\n"
	var h RequestHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if h.ConnectionClose() {
		t.Fatalf("unexpected 'Connection: close' request header")
	}
}

func TestBufferStartEnd(t *testing.T) {
	testBufferStartEnd(t, "", "", "")
	testBufferStartEnd(t, "foobar", "foobar", "")

	b := string(createFixedBody(199))
	testBufferStartEnd(t, b, b, "")
	for i := 0; i < 10; i++ {
		b += "foobar"
		testBufferStartEnd(t, b, b, "")
	}

	b = string(createFixedBody(400))
	testBufferStartEnd(t, b, b, "")
	for i := 0; i < 10; i++ {
		b += "sadfqwer"
		testBufferStartEnd(t, b, b[:200], b[len(b)-200:])
	}
}

func testBufferStartEnd(t *testing.T, buf, expectedStart, expectedEnd string) {
	start, end := bufferStartEnd([]byte(buf))
	if string(start) != expectedStart {
		t.Fatalf("unexpected start %q. Expecting %q. buf %q", start, expectedStart, buf)
	}
	if string(end) != expectedEnd {
		t.Fatalf("unexpected end %q. Expecting %q. buf %q", end, expectedEnd, buf)
	}
}

func TestResponseHeaderTrailingCRLFSuccess(t *testing.T) {
	trailingCRLF := "\r\n\r\n\r\n"
	s := "HTTP/1.1 200 OK\r\nContent-Type: aa\r\nContent-Length: 123\r\n\r\n" + trailingCRLF

	var r ResponseHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := r.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// try reading the trailing CRLF. It must return EOF
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error: %s. Expecting %s", err, io.EOF)
	}
}

func TestResponseHeaderTrailingCRLFError(t *testing.T) {
	trailingCRLF := "\r\nerror\r\n\r\n"
	s := "HTTP/1.1 200 OK\r\nContent-Type: aa\r\nContent-Length: 123\r\n\r\n" + trailingCRLF

	var r ResponseHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := r.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// try reading the trailing CRLF. It must return EOF
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err == io.EOF {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestRequestHeaderTrailingCRLFSuccess(t *testing.T) {
	trailingCRLF := "\r\n\r\n\r\n"
	s := "GET / HTTP/1.1\r\nHost: aaa.com\r\n\r\n" + trailingCRLF

	var r RequestHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := r.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// try reading the trailing CRLF. It must return EOF
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error: %s. Expecting %s", err, io.EOF)
	}
}

func TestRequestHeaderTrailingCRLFError(t *testing.T) {
	trailingCRLF := "\r\nerror\r\n\r\n"
	s := "GET / HTTP/1.1\r\nHost: aaa.com\r\n\r\n" + trailingCRLF

	var r RequestHeader
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := r.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// try reading the trailing CRLF. It must return EOF
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err == io.EOF {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestRequestHeaderReadEOF(t *testing.T) {
	var r RequestHeader

	br := bufio.NewReader(&bytes.Buffer{})
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error: %s. Expecting %s", err, io.EOF)
	}

	// incomplete request header mustn't return io.EOF
	br = bufio.NewReader(bytes.NewBufferString("GET "))
	err = r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err == io.EOF {
		t.Fatalf("expecting non-EOF error")
	}
}

func TestResponseHeaderReadEOF(t *testing.T) {
	var r ResponseHeader

	br := bufio.NewReader(&bytes.Buffer{})
	err := r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error: %s. Expecting %s", err, io.EOF)
	}

	// incomplete response header mustn't return io.EOF
	br = bufio.NewReader(bytes.NewBufferString("HTTP/1.1 "))
	err = r.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err == io.EOF {
		t.Fatalf("expecting non-EOF error")
	}
}

func TestResponseHeaderOldVersion(t *testing.T) {
	var h ResponseHeader

	s := "HTTP/1.0 200 OK\r\nContent-Length: 5\r\nContent-Type: aaa\r\n\r\n12345"
	s += "HTTP/1.0 200 OK\r\nContent-Length: 2\r\nContent-Type: ass\r\nConnection: keep-alive\r\n\r\n42"
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !h.ConnectionClose() {
		t.Fatalf("expecting 'Connection: close' for the response with old http protocol")
	}

	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if h.ConnectionClose() {
		t.Fatalf("unexpected 'Connection: close' for keep-alive response with old http protocol")
	}
}

func TestRequestHeaderSetByteRange(t *testing.T) {
	testRequestHeaderSetByteRange(t, 0, 10, "bytes=0-10")
	testRequestHeaderSetByteRange(t, 123, -1, "bytes=123-")
	testRequestHeaderSetByteRange(t, -234, 58349, "bytes=-234")
}

func testRequestHeaderSetByteRange(t *testing.T, startPos, endPos int, expectedV string) {
	var h RequestHeader
	h.SetByteRange(startPos, endPos)
	v := h.Peek("Range")
	if string(v) != expectedV {
		t.Fatalf("unexpected range: %q. Expecting %q. startPos=%d, endPos=%d", v, expectedV, startPos, endPos)
	}
}

func TestResponseHeaderSetContentRange(t *testing.T) {
	testResponseHeaderSetContentRange(t, 0, 0, 1, "bytes 0-0/1")
	testResponseHeaderSetContentRange(t, 123, 456, 789, "bytes 123-456/789")
}

func testResponseHeaderSetContentRange(t *testing.T, startPos, endPos, contentLength int, expectedV string) {
	var h ResponseHeader
	h.SetContentRange(startPos, endPos, contentLength)
	v := h.Peek("Content-Range")
	if string(v) != expectedV {
		t.Fatalf("unexpected content-range: %q. Expecting %q. startPos=%d, endPos=%d, contentLength=%d",
			v, expectedV, startPos, endPos, contentLength)
	}
}

func TestRequestHeaderHasAcceptEncoding(t *testing.T) {
	testRequestHeaderHasAcceptEncoding(t, "", "gzip", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip", "sdhc", false)
	testRequestHeaderHasAcceptEncoding(t, "deflate", "deflate", true)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "gzi", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "dhc", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "sdh", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "zip", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "flat", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "flate", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "def", false)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "gzip", true)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "deflate", true)
	testRequestHeaderHasAcceptEncoding(t, "gzip, deflate, sdhc", "sdhc", true)
}

func testRequestHeaderHasAcceptEncoding(t *testing.T, ae, v string, resultExpected bool) {
	var h RequestHeader
	h.Set("Accept-Encoding", ae)
	result := h.HasAcceptEncoding(v)
	if result != resultExpected {
		t.Fatalf("unexpected result in HasAcceptEncoding(%q, %q): %v. Expecting %v", ae, v, result, resultExpected)
	}
}

func TestRequestMultipartFormBoundary(t *testing.T) {
	testRequestMultipartFormBoundary(t, "POST / HTTP/1.1\r\nContent-Type: multipart/form-data; boundary=foobar\r\n\r\n", "foobar")

	// incorrect content-type
	testRequestMultipartFormBoundary(t, "POST / HTTP/1.1\r\nContent-Type: foo/bar\r\n\r\n", "")

	// empty boundary
	testRequestMultipartFormBoundary(t, "POST / HTTP/1.1\r\nContent-Type: multipart/form-data; boundary=\r\n\r\n", "")

	// missing boundary
	testRequestMultipartFormBoundary(t, "POST / HTTP/1.1\r\nContent-Type: multipart/form-data\r\n\r\n", "")

	// boundary after other content-type params
	testRequestMultipartFormBoundary(t, "POST / HTTP/1.1\r\nContent-Type: multipart/form-data;   foo=bar;   boundary=--aaabb  \r\n\r\n", "--aaabb")

	var h RequestHeader
	h.SetMultipartFormBoundary("foobarbaz")
	b := h.MultipartFormBoundary()
	if string(b) != "foobarbaz" {
		t.Fatalf("unexpected boundary %q. Expecting %q", b, "foobarbaz")
	}
}

func testRequestMultipartFormBoundary(t *testing.T, s, boundary string) {
	var h RequestHeader
	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. s=%q, boundary=%q", err, s, boundary)
	}

	b := h.MultipartFormBoundary()
	if string(b) != boundary {
		t.Fatalf("unexpected boundary %q. Expecting %q. s=%q", b, boundary, s)
	}
}

func TestResponseHeaderConnectionUpgrade(t *testing.T) {
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.1 200 OK\r\nContent-Length: 10\r\nConnection: Upgrade, HTTP2-Settings\r\n\r\n",
		true, true)
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.1 200 OK\r\nContent-Length: 10\r\nConnection: keep-alive, Upgrade\r\n\r\n",
		true, true)

	// non-http/1.1 protocol has 'connection: close' by default, which also disables 'connection: upgrade'
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.0 200 OK\r\nContent-Length: 10\r\nConnection: Upgrade, HTTP2-Settings\r\n\r\n",
		false, false)

	// explicit keep-alive for non-http/1.1, so 'connection: upgrade' works
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.0 200 OK\r\nContent-Length: 10\r\nConnection: Upgrade, keep-alive\r\n\r\n",
		true, true)

	// implicit keep-alive for http/1.1
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\n", false, true)

	// no content-length, so 'connection: close' is assumed
	testResponseHeaderConnectionUpgrade(t, "HTTP/1.1 200 OK\r\n\r\n", false, false)
}

func testResponseHeaderConnectionUpgrade(t *testing.T, s string, isUpgrade, isKeepAlive bool) {
	var h ResponseHeader

	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. Response header %q", err, s)
	}
	upgrade := h.ConnectionUpgrade()
	if upgrade != isUpgrade {
		t.Fatalf("unexpected 'connection: upgrade' when parsing response header: %v. Expecting %v. header %q. v=%q",
			upgrade, isUpgrade, s, h.Peek("Connection"))
	}
	keepAlive := !h.ConnectionClose()
	if keepAlive != isKeepAlive {
		t.Fatalf("unexpected 'connection: keep-alive' when parsing response header: %v. Expecting %v. header %q. v=%q",
			keepAlive, isKeepAlive, s, &h)
	}
}

func TestRequestHeaderConnectionUpgrade(t *testing.T) {
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.1\r\nConnection: Upgrade, HTTP2-Settings\r\nHost: foobar.com\r\n\r\n",
		true, true)
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.1\r\nConnection: keep-alive,Upgrade\r\nHost: foobar.com\r\n\r\n",
		true, true)

	// non-http/1.1 has 'connection: close' by default, which resets 'connection: upgrade'
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.0\r\nConnection: Upgrade, HTTP2-Settings\r\nHost: foobar.com\r\n\r\n",
		false, false)

	// explicit 'connection: keep-alive' in non-http/1.1
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.0\r\nConnection: foo, Upgrade, keep-alive\r\nHost: foobar.com\r\n\r\n",
		true, true)

	// no upgrade
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.1\r\nConnection: Upgradess, foobar\r\nHost: foobar.com\r\n\r\n",
		false, true)
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.1\r\nHost: foobar.com\r\n\r\n",
		false, true)

	// explicit connection close
	testRequestHeaderConnectionUpgrade(t, "GET /foobar HTTP/1.1\r\nConnection: close\r\nHost: foobar.com\r\n\r\n",
		false, false)
}

func testRequestHeaderConnectionUpgrade(t *testing.T, s string, isUpgrade, isKeepAlive bool) {
	var h RequestHeader

	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. Request header %q", err, s)
	}
	upgrade := h.ConnectionUpgrade()
	if upgrade != isUpgrade {
		t.Fatalf("unexpected 'connection: upgrade' when parsing request header: %v. Expecting %v. header %q",
			upgrade, isUpgrade, s)
	}
	keepAlive := !h.ConnectionClose()
	if keepAlive != isKeepAlive {
		t.Fatalf("unexpected 'connection: keep-alive' when parsing request header: %v. Expecting %v. header %q",
			keepAlive, isKeepAlive, s)
	}
}

func TestRequestHeaderProxyWithCookie(t *testing.T) {
	// Proxy request header (read it, then write it without touching any headers).
	var h RequestHeader
	r := bytes.NewBufferString("GET /foo HTTP/1.1\r\nFoo: bar\r\nHost: aaa.com\r\nCookie: foo=bar; bazzz=aaaaaaa; x=y\r\nCookie: aqqqqq=123\r\n\r\n")
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	if err := h.Write(bw); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var h1 RequestHeader
	br.Reset(w)
	if err := h1.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if string(h1.RequestURI()) != "/foo" {
		t.Fatalf("unexpected requestURI: %q. Expecting %q", h1.RequestURI(), "/foo")
	}
	if string(h1.Host()) != "aaa.com" {
		t.Fatalf("unexpected host: %q. Expecting %q", h1.Host(), "aaa.com")
	}
	if string(h1.Peek("Foo")) != "bar" {
		t.Fatalf("unexpected Foo: %q. Expecting %q", h1.Peek("Foo"), "bar")
	}
	if string(h1.Cookie("foo")) != "bar" {
		t.Fatalf("unexpected coookie foo=%q. Expecting %q", h1.Cookie("foo"), "bar")
	}
	if string(h1.Cookie("bazzz")) != "aaaaaaa" {
		t.Fatalf("unexpected cookie bazzz=%q. Expecting %q", h1.Cookie("bazzz"), "aaaaaaa")
	}
	if string(h1.Cookie("x")) != "y" {
		t.Fatalf("unexpected cookie x=%q. Expecting %q", h1.Cookie("x"), "y")
	}
	if string(h1.Cookie("aqqqqq")) != "123" {
		t.Fatalf("unexpected cookie aqqqqq=%q. Expecting %q", h1.Cookie("aqqqqq"), "123")
	}
}

func TestPeekRawHeader(t *testing.T) {
	// empty header
	testPeekRawHeader(t, "", "Foo-Bar", "")

	// different case
	testPeekRawHeader(t, "Content-Length: 3443\r\n", "content-length", "")

	// no trailing crlf
	testPeekRawHeader(t, "Content-Length: 234", "Content-Length", "")

	// single header
	testPeekRawHeader(t, "Content-Length: 12345\r\n", "Content-Length", "12345")

	// multiple headers
	testPeekRawHeader(t, "Host: foobar\r\nContent-Length: 434\r\nFoo: bar\r\n\r\n", "Content-Length", "434")

	// lf without cr
	testPeekRawHeader(t, "Foo: bar\nConnection: close\nAaa: bbb\ncc: ddd\n", "Connection", "close")
}

func testPeekRawHeader(t *testing.T, rawHeaders, key string, expectedValue string) {
	v := peekRawHeader([]byte(rawHeaders), []byte(key))
	if string(v) != expectedValue {
		t.Fatalf("unexpected raw headers value %q. Expected %q. key %q, rawHeaders %q", v, expectedValue, key, rawHeaders)
	}
}

func TestResponseHeaderFirstByteReadEOF(t *testing.T) {
	var h ResponseHeader

	r := &errorReader{fmt.Errorf("non-eof error")}
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error %s. Expecting %s", err, io.EOF)
	}
}

func TestRequestHeaderFirstByteReadEOF(t *testing.T) {
	var h RequestHeader

	r := &errorReader{fmt.Errorf("non-eof error")}
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err == nil {
		t.Fatalf("expecting error")
	}
	if err != io.EOF {
		t.Fatalf("unexpected error %s. Expecting %s", err, io.EOF)
	}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestRequestHeaderEmptyMethod(t *testing.T) {
	var h RequestHeader

	if !h.IsGet() {
		t.Fatalf("empty method must be equivalent to GET")
	}
	if h.IsPost() {
		t.Fatalf("empty method cannot be POST")
	}
	if h.IsHead() {
		t.Fatalf("empty method cannot be HEAD")
	}
}

func TestResponseHeaderHTTPVer(t *testing.T) {
	// non-http/1.1
	testResponseHeaderHTTPVer(t, "HTTP/1.0 200 OK\r\nContent-Type: aaa\r\nContent-Length: 123\r\n\r\n", true)
	testResponseHeaderHTTPVer(t, "HTTP/0.9 200 OK\r\nContent-Type: aaa\r\nContent-Length: 123\r\n\r\n", true)
	testResponseHeaderHTTPVer(t, "foobar 200 OK\r\nContent-Type: aaa\r\nContent-Length: 123\r\n\r\n", true)

	// http/1.1
	testResponseHeaderHTTPVer(t, "HTTP/1.1 200 OK\r\nContent-Type: aaa\r\nContent-Length: 123\r\n\r\n", false)
}

func TestRequestHeaderHTTPVer(t *testing.T) {
	// non-http/1.1
	testRequestHeaderHTTPVer(t, "GET / HTTP/1.0\r\nHost: aa.com\r\n\r\n", true)
	testRequestHeaderHTTPVer(t, "GET / HTTP/0.9\r\nHost: aa.com\r\n\r\n", true)
	testRequestHeaderHTTPVer(t, "GET / foobar\r\nHost: aa.com\r\n\r\n", true)

	// empty http version
	testRequestHeaderHTTPVer(t, "GET /\r\nHost: aaa.com\r\n\r\n", true)
	testRequestHeaderHTTPVer(t, "GET / \r\nHost: aaa.com\r\n\r\n", true)

	// http/1.1
	testRequestHeaderHTTPVer(t, "GET / HTTP/1.1\r\nHost: a.com\r\n\r\n", false)
}

func testResponseHeaderHTTPVer(t *testing.T, s string, connectionClose bool) {
	var h ResponseHeader

	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. response=%q", err, s)
	}
	if h.ConnectionClose() != connectionClose {
		t.Fatalf("unexpected connectionClose %v. Expecting %v. response=%q", h.ConnectionClose(), connectionClose, s)
	}
}

func testRequestHeaderHTTPVer(t *testing.T, s string, connectionClose bool) {
	var h RequestHeader

	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. request=%q", err, s)
	}
	if h.ConnectionClose() != connectionClose {
		t.Fatalf("unexpected connectionClose %v. Expecting %v. request=%q", h.ConnectionClose(), connectionClose, s)
	}
}

func TestResponseHeaderCopyTo(t *testing.T) {
	var h ResponseHeader

	h.Set("Set-Cookie", "foo=bar")
	h.Set("Content-Type", "foobar")
	h.Set("AAA-BBB", "aaaa")

	var h1 ResponseHeader
	h.CopyTo(&h1)
	if !bytes.Equal(h1.Peek("Set-cookie"), h.Peek("Set-Cookie")) {
		t.Fatalf("unexpected cookie %q. Expected %q", h1.Peek("set-cookie"), h.Peek("set-cookie"))
	}
	if !bytes.Equal(h1.Peek("Content-Type"), h.Peek("Content-Type")) {
		t.Fatalf("unexpected content-type %q. Expected %q", h1.Peek("content-type"), h.Peek("content-type"))
	}
	if !bytes.Equal(h1.Peek("aaa-bbb"), h.Peek("AAA-BBB")) {
		t.Fatalf("unexpected aaa-bbb %q. Expected %q", h1.Peek("aaa-bbb"), h.Peek("aaa-bbb"))
	}
}

func TestRequestHeaderCopyTo(t *testing.T) {
	var h RequestHeader

	h.Set("Cookie", "aa=bb; cc=dd")
	h.Set("Content-Type", "foobar")
	h.Set("Host", "aaaa")
	h.Set("aaaxxx", "123")

	var h1 RequestHeader
	h.CopyTo(&h1)
	if !bytes.Equal(h1.Peek("cookie"), h.Peek("Cookie")) {
		t.Fatalf("unexpected cookie after copying: %q. Expected %q", h1.Peek("cookie"), h.Peek("cookie"))
	}
	if !bytes.Equal(h1.Peek("content-type"), h.Peek("Content-Type")) {
		t.Fatalf("unexpected content-type %q. Expected %q", h1.Peek("content-type"), h.Peek("content-type"))
	}
	if !bytes.Equal(h1.Peek("host"), h.Peek("host")) {
		t.Fatalf("unexpected host %q. Expected %q", h1.Peek("host"), h.Peek("host"))
	}
	if !bytes.Equal(h1.Peek("aaaxxx"), h.Peek("aaaxxx")) {
		t.Fatalf("unexpected aaaxxx %q. Expected %q", h1.Peek("aaaxxx"), h.Peek("aaaxxx"))
	}
}

func TestRequestHeaderConnectionClose(t *testing.T) {
	var h RequestHeader

	h.Set("Connection", "close")
	h.Set("Host", "foobar")
	if !h.ConnectionClose() {
		t.Fatalf("connection: close not set")
	}

	var w bytes.Buffer
	bw := bufio.NewWriter(&w)
	if err := h.Write(bw); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var h1 RequestHeader
	br := bufio.NewReader(&w)
	if err := h1.Read(br); err != nil {
		t.Fatalf("error when reading request header: %s", err)
	}

	if !h1.ConnectionClose() {
		t.Fatalf("unexpected connection: close value: %v", h1.ConnectionClose())
	}
	if string(h1.Peek("Connection")) != "close" {
		t.Fatalf("unexpected connection value: %q. Expecting %q", h.Peek("Connection"), "close")
	}
}

func TestRequestHeaderSetCookie(t *testing.T) {
	var h RequestHeader

	h.Set("Cookie", "foo=bar; baz=aaa")
	h.Set("cOOkie", "xx=yyy")

	if string(h.Cookie("foo")) != "bar" {
		t.Fatalf("Unexpected cookie %q. Expecting %q", h.Cookie("foo"), "bar")
	}
	if string(h.Cookie("baz")) != "aaa" {
		t.Fatalf("Unexpected cookie %q. Expecting %q", h.Cookie("baz"), "aaa")
	}
	if string(h.Cookie("xx")) != "yyy" {
		t.Fatalf("unexpected cookie %q. Expecting %q", h.Cookie("xx"), "yyy")
	}
}

func TestResponseHeaderSetCookie(t *testing.T) {
	var h ResponseHeader

	h.Set("set-cookie", "foo=bar; path=/aa/bb; domain=aaa.com")
	h.Set("Set-Cookie", "aaaaa=bxx")

	var c Cookie
	c.SetKey("foo")
	if !h.Cookie(&c) {
		t.Fatalf("cannot obtain %q cookie", c.Key())
	}
	if string(c.Value()) != "bar" {
		t.Fatalf("unexpected cookie value %q. Expected %q", c.Value(), "bar")
	}
	if string(c.Path()) != "/aa/bb" {
		t.Fatalf("unexpected cookie path %q. Expected %q", c.Path(), "/aa/bb")
	}
	if string(c.Domain()) != "aaa.com" {
		t.Fatalf("unexpected cookie domain %q. Expected %q", c.Domain(), "aaa.com")
	}

	c.SetKey("aaaaa")
	if !h.Cookie(&c) {
		t.Fatalf("cannot obtain %q cookie", c.Key())
	}
	if string(c.Value()) != "bxx" {
		t.Fatalf("unexpected cookie value %q. Expecting %q", c.Value(), "bxx")
	}
}

func TestResponseHeaderVisitAll(t *testing.T) {
	var h ResponseHeader

	r := bytes.NewBufferString("HTTP/1.1 200 OK\r\nContent-Type: aa\r\nContent-Length: 123\r\nSet-Cookie: aa=bb; path=/foo/bar\r\nSet-Cookie: ccc\r\n\r\n")
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("Unepxected error: %s", err)
	}

	if h.Len() != 4 {
		t.Fatalf("Unexpected number of headers: %d. Expected 4", h.Len())
	}
	contentLengthCount := 0
	contentTypeCount := 0
	cookieCount := 0
	h.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		switch k {
		case "Content-Length":
			if v != string(h.Peek(k)) {
				t.Fatalf("unexpected content-length: %q. Expecting %q", v, h.Peek(k))
			}
			contentLengthCount++
		case "Content-Type":
			if v != string(h.Peek(k)) {
				t.Fatalf("Unexpected content-type: %q. Expected %q", v, h.Peek(k))
			}
			contentTypeCount++
		case "Set-Cookie":
			if cookieCount == 0 && v != "aa=bb; path=/foo/bar" {
				t.Fatalf("unexpected cookie header: %q. Expected %q", v, "aa=bb; path=/foo/bar")
			}
			if cookieCount == 1 && v != "ccc" {
				t.Fatalf("unexpected cookie header: %q. Expected %q", v, "ccc")
			}
			cookieCount++
		default:
			t.Fatalf("unexpected header %q=%q", k, v)
		}
	})
	if contentLengthCount != 1 {
		t.Fatalf("unexpected number of content-length headers: %d. Expected 1", contentLengthCount)
	}
	if contentTypeCount != 1 {
		t.Fatalf("unexpected number of content-type headers: %d. Expected 1", contentTypeCount)
	}
	if cookieCount != 2 {
		t.Fatalf("unexpected number of cookie header: %d. Expected 2", cookieCount)
	}
}

func TestRequestHeaderVisitAll(t *testing.T) {
	var h RequestHeader

	r := bytes.NewBufferString("GET / HTTP/1.1\r\nHost: aa.com\r\nXX: YYY\r\nXX: ZZ\r\nCookie: a=b; c=d\r\n\r\n")
	br := bufio.NewReader(r)
	if err := h.Read(br); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if h.Len() != 4 {
		t.Fatalf("Unexpected number of header: %d. Expected 4", h.Len())
	}
	hostCount := 0
	xxCount := 0
	cookieCount := 0
	h.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		switch k {
		case "Host":
			if v != string(h.Peek(k)) {
				t.Fatalf("Unexpected host value %q. Expected %q", v, h.Peek(k))
			}
			hostCount++
		case "Xx":
			if xxCount == 0 && v != "YYY" {
				t.Fatalf("Unexpected value %q. Expected %q", v, "YYY")
			}
			if xxCount == 1 && v != "ZZ" {
				t.Fatalf("Unexpected value %q. Expected %q", v, "ZZ")
			}
			xxCount++
		case "Cookie":
			if v != "a=b; c=d" {
				t.Fatalf("Unexpected cookie %q. Expected %q", v, "a=b; c=d")
			}
			cookieCount++
		default:
			t.Fatalf("Unepxected header %q=%q", k, v)
		}
	})
	if hostCount != 1 {
		t.Fatalf("Unepxected number of host headers detected %d. Expected 1", hostCount)
	}
	if xxCount != 2 {
		t.Fatalf("Unexpected number of xx headers detected %d. Expected 2", xxCount)
	}
	if cookieCount != 1 {
		t.Fatalf("Unexpected number of cookie headers %d. Expected 1", cookieCount)
	}
}

func TestResponseHeaderCookie(t *testing.T) {
	var h ResponseHeader
	var c Cookie

	c.SetKey("foobar")
	c.SetValue("aaa")
	h.SetCookie(&c)

	c.SetKey("йцук")
	c.SetDomain("foobar.com")
	h.SetCookie(&c)

	c.Reset()
	c.SetKey("foobar")
	if !h.Cookie(&c) {
		t.Fatalf("Cannot find cookie %q", c.Key())
	}

	var expectedC1 Cookie
	expectedC1.SetKey("foobar")
	expectedC1.SetValue("aaa")
	if !equalCookie(&expectedC1, &c) {
		t.Fatalf("unexpected cookie\n%#v\nExpected\n%#v\n", &c, &expectedC1)
	}

	c.SetKey("йцук")
	if !h.Cookie(&c) {
		t.Fatalf("cannot find cookie %q", c.Key())
	}

	var expectedC2 Cookie
	expectedC2.SetKey("йцук")
	expectedC2.SetValue("aaa")
	expectedC2.SetDomain("foobar.com")
	if !equalCookie(&expectedC2, &c) {
		t.Fatalf("unexpected cookie\n%v\nExpected\n%v\n", &c, &expectedC2)
	}

	h.VisitAllCookie(func(key, value []byte) {
		var cc Cookie
		cc.ParseBytes(value)
		if !bytes.Equal(key, cc.Key()) {
			t.Fatalf("Unexpected cookie key %q. Expected %q", key, cc.Key())
		}
		switch {
		case bytes.Equal(key, []byte("foobar")):
			if !equalCookie(&expectedC1, &cc) {
				t.Fatalf("unexpected cookie\n%v\nExpected\n%v\n", &cc, &expectedC1)
			}
		case bytes.Equal(key, []byte("йцук")):
			if !equalCookie(&expectedC2, &cc) {
				t.Fatalf("unexpected cookie\n%v\nExpected\n%v\n", &cc, &expectedC2)
			}
		default:
			t.Fatalf("unexpected cookie key %q", key)
		}
	})

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	if err := h.Write(bw); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var h1 ResponseHeader
	br := bufio.NewReader(w)
	if err := h1.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	c.SetKey("foobar")
	if !h1.Cookie(&c) {
		t.Fatalf("Cannot find cookie %q", c.Key())
	}
	if !equalCookie(&expectedC1, &c) {
		t.Fatalf("unexpected cookie\n%v\nExpected\n%v\n", &c, &expectedC1)
	}

	c.SetKey("йцук")
	if !h1.Cookie(&c) {
		t.Fatalf("cannot find cookie %q", c.Key())
	}
	if !equalCookie(&expectedC2, &c) {
		t.Fatalf("unexpected cookie\n%v\nExpected\n%v\n", &c, &expectedC2)
	}
}

func equalCookie(c1, c2 *Cookie) bool {
	if !bytes.Equal(c1.Key(), c2.Key()) {
		return false
	}
	if !bytes.Equal(c1.Value(), c2.Value()) {
		return false
	}
	if !c1.Expire().Equal(c2.Expire()) {
		return false
	}
	if !bytes.Equal(c1.Domain(), c2.Domain()) {
		return false
	}
	if !bytes.Equal(c1.Path(), c2.Path()) {
		return false
	}
	return true
}

func TestRequestHeaderCookie(t *testing.T) {
	var h RequestHeader
	h.SetRequestURI("/foobar")
	h.Set("Host", "foobar.com")

	h.SetCookie("foo", "bar")
	h.SetCookie("привет", "мир")

	if string(h.Cookie("foo")) != "bar" {
		t.Fatalf("Unexpected cookie value %q. Exepcted %q", h.Cookie("foo"), "bar")
	}
	if string(h.Cookie("привет")) != "мир" {
		t.Fatalf("Unexpected cookie value %q. Expected %q", h.Cookie("привет"), "мир")
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	if err := h.Write(bw); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	var h1 RequestHeader
	br := bufio.NewReader(w)
	if err := h1.Read(br); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if !bytes.Equal(h1.Cookie("foo"), h.Cookie("foo")) {
		t.Fatalf("Unexpected cookie value %q. Exepcted %q", h1.Cookie("foo"), h.Cookie("foo"))
	}
	if !bytes.Equal(h1.Cookie("привет"), h.Cookie("привет")) {
		t.Fatalf("Unexpected cookie value %q. Expected %q", h1.Cookie("привет"), h.Cookie("привет"))
	}
}

func TestRequestHeaderSetGet(t *testing.T) {
	h := &RequestHeader{}
	h.SetRequestURI("/aa/bbb")
	h.SetMethod("POST")
	h.Set("foo", "bar")
	h.Set("host", "12345")
	h.Set("content-type", "aaa/bbb")
	h.Set("content-length", "1234")
	h.Set("user-agent", "aaabbb")
	h.Set("referer", "axcv")
	h.Set("baz", "xxxxx")
	h.Set("transfer-encoding", "chunked")
	h.Set("connection", "close")

	expectRequestHeaderGet(t, h, "Foo", "bar")
	expectRequestHeaderGet(t, h, "Host", "12345")
	expectRequestHeaderGet(t, h, "Content-Type", "aaa/bbb")
	expectRequestHeaderGet(t, h, "Content-Length", "1234")
	expectRequestHeaderGet(t, h, "USER-AGent", "aaabbb")
	expectRequestHeaderGet(t, h, "Referer", "axcv")
	expectRequestHeaderGet(t, h, "baz", "xxxxx")
	expectRequestHeaderGet(t, h, "Transfer-Encoding", "")
	expectRequestHeaderGet(t, h, "connecTION", "close")
	if !h.ConnectionClose() {
		t.Fatalf("unset connection: close")
	}

	if h.ContentLength() != 1234 {
		t.Fatalf("Unexpected content-length %d. Expected %d", h.ContentLength(), 1234)
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	err := h.Write(bw)
	if err != nil {
		t.Fatalf("Unexpected error when writing request header: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Unexpected error when flushing request header: %s", err)
	}

	var h1 RequestHeader
	br := bufio.NewReader(w)
	if err = h1.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading request header: %s", err)
	}

	if h1.ContentLength() != h.ContentLength() {
		t.Fatalf("Unexpected Content-Length %d. Expected %d", h1.ContentLength(), h.ContentLength())
	}

	expectRequestHeaderGet(t, &h1, "Foo", "bar")
	expectRequestHeaderGet(t, &h1, "HOST", "12345")
	expectRequestHeaderGet(t, &h1, "Content-Type", "aaa/bbb")
	expectRequestHeaderGet(t, &h1, "Content-Length", "1234")
	expectRequestHeaderGet(t, &h1, "USER-AGent", "aaabbb")
	expectRequestHeaderGet(t, &h1, "Referer", "axcv")
	expectRequestHeaderGet(t, &h1, "baz", "xxxxx")
	expectRequestHeaderGet(t, &h1, "Transfer-Encoding", "")
	expectRequestHeaderGet(t, &h1, "Connection", "close")
	if !h1.ConnectionClose() {
		t.Fatalf("unset connection: close")
	}
}

func TestResponseHeaderSetGet(t *testing.T) {
	h := &ResponseHeader{}
	h.Set("foo", "bar")
	h.Set("content-type", "aaa/bbb")
	h.Set("connection", "close")
	h.Set("content-length", "1234")
	h.Set("Server", "aaaa")
	h.Set("baz", "xxxxx")
	h.Set("Transfer-Encoding", "chunked")

	expectResponseHeaderGet(t, h, "Foo", "bar")
	expectResponseHeaderGet(t, h, "Content-Type", "aaa/bbb")
	expectResponseHeaderGet(t, h, "Connection", "close")
	expectResponseHeaderGet(t, h, "Content-Length", "1234")
	expectResponseHeaderGet(t, h, "seRVer", "aaaa")
	expectResponseHeaderGet(t, h, "baz", "xxxxx")
	expectResponseHeaderGet(t, h, "Transfer-Encoding", "")

	if h.ContentLength() != 1234 {
		t.Fatalf("Unexpected content-length %d. Expected %d", h.ContentLength(), 1234)
	}
	if !h.ConnectionClose() {
		t.Fatalf("Unexpected Connection: close value %v. Expected %v", h.ConnectionClose(), true)
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	err := h.Write(bw)
	if err != nil {
		t.Fatalf("Unexpected error when writing response header: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Unexpected error when flushing response header: %s", err)
	}

	var h1 ResponseHeader
	br := bufio.NewReader(w)
	if err = h1.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response header: %s", err)
	}

	if h1.ContentLength() != h.ContentLength() {
		t.Fatalf("Unexpected Content-Length %d. Expected %d", h1.ContentLength(), h.ContentLength())
	}
	if h1.ConnectionClose() != h.ConnectionClose() {
		t.Fatalf("unexpected connection: close %v. Expected %v", h1.ConnectionClose(), h.ConnectionClose())
	}

	expectResponseHeaderGet(t, &h1, "Foo", "bar")
	expectResponseHeaderGet(t, &h1, "Content-Type", "aaa/bbb")
	expectResponseHeaderGet(t, &h1, "Connection", "close")
	expectResponseHeaderGet(t, &h1, "seRVer", "aaaa")
	expectResponseHeaderGet(t, &h1, "baz", "xxxxx")
}

func expectRequestHeaderGet(t *testing.T, h *RequestHeader, key, expectedValue string) {
	if string(h.Peek(key)) != expectedValue {
		t.Fatalf("Unexpected value for key %q: %q. Expected %q", key, h.Peek(key), expectedValue)
	}
}

func expectResponseHeaderGet(t *testing.T, h *ResponseHeader, key, expectedValue string) {
	if string(h.Peek(key)) != expectedValue {
		t.Fatalf("Unexpected value for key %q: %q. Expected %q", key, h.Peek(key), expectedValue)
	}
}

func TestResponseHeaderConnectionClose(t *testing.T) {
	testResponseHeaderConnectionClose(t, true)
	testResponseHeaderConnectionClose(t, false)
}

func testResponseHeaderConnectionClose(t *testing.T, connectionClose bool) {
	h := &ResponseHeader{}
	if connectionClose {
		h.SetConnectionClose()
	}
	h.SetContentLength(123)

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	err := h.Write(bw)
	if err != nil {
		t.Fatalf("Unexpected error when writing response header: %s", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Unexpected error when flushing response header: %s", err)
	}

	var h1 ResponseHeader
	br := bufio.NewReader(w)
	err = h1.Read(br)
	if err != nil {
		t.Fatalf("Unexpected error when reading response header: %s", err)
	}
	if h1.ConnectionClose() != h.ConnectionClose() {
		t.Fatalf("Unexpected value for ConnectionClose: %v. Expected %v", h1.ConnectionClose(), h.ConnectionClose())
	}
}

func TestRequestHeaderTooBig(t *testing.T) {
	s := "GET / HTTP/1.1\r\nHost: aaa.com\r\n" + getHeaders(10500) + "\r\n"
	r := bytes.NewBufferString(s)
	br := bufio.NewReaderSize(r, 4096)
	h := &RequestHeader{}
	err := h.Read(br)
	if err == nil {
		t.Fatalf("Expecting error when reading too big header")
	}
}

func TestResponseHeaderTooBig(t *testing.T) {
	s := "HTTP/1.1 200 OK\r\nContent-Type: sss\r\nContent-Length: 0\r\n" + getHeaders(100500) + "\r\n"
	r := bytes.NewBufferString(s)
	br := bufio.NewReaderSize(r, 4096)
	h := &ResponseHeader{}
	err := h.Read(br)
	if err == nil {
		t.Fatalf("Expecting error when reading too big header")
	}
}

type bufioPeekReader struct {
	s string
	n int
}

func (r *bufioPeekReader) Read(b []byte) (int, error) {
	if len(r.s) == 0 {
		return 0, io.EOF
	}

	r.n++
	n := r.n
	if len(r.s) < n {
		n = len(r.s)
	}
	src := []byte(r.s[:n])
	r.s = r.s[n:]
	n = copy(b, src)
	return n, nil
}

func TestRequestHeaderBufioPeek(t *testing.T) {
	r := &bufioPeekReader{
		s: "GET / HTTP/1.1\r\nHost: foobar.com\r\n" + getHeaders(10) + "\r\naaaa",
	}
	br := bufio.NewReaderSize(r, 4096)
	h := &RequestHeader{}
	if err := h.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading request: %s", err)
	}
	verifyRequestHeader(t, h, 0, "/", "foobar.com", "", "")
	verifyTrailer(t, br, "aaaa")
}

func TestResponseHeaderBufioPeek(t *testing.T) {
	r := &bufioPeekReader{
		s: "HTTP/1.1 200 OK\r\nContent-Length: 10\r\nContent-Type: aaa\r\n" + getHeaders(10) + "\r\n0123456789",
	}
	br := bufio.NewReaderSize(r, 4096)
	h := &ResponseHeader{}
	if err := h.Read(br); err != nil {
		t.Fatalf("Unexpected error when reading response: %s", err)
	}
	verifyResponseHeader(t, h, 200, 10, "aaa")
	verifyTrailer(t, br, "0123456789")
}

func getHeaders(n int) string {
	var h []string
	for i := 0; i < n; i++ {
		h = append(h, fmt.Sprintf("Header_%d: Value_%d\r\n", i, i))
	}
	return strings.Join(h, "")
}

func TestResponseHeaderReadSuccess(t *testing.T) {
	h := &ResponseHeader{}

	// straight order of content-length and content-type
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n\r\n",
		200, 123, "text/html", "")
	if h.ConnectionClose() {
		t.Fatalf("unexpected connection: close")
	}

	// reverse order of content-length and content-type
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 202 OK\r\nContent-Type: text/plain; encoding=utf-8\r\nContent-Length: 543\r\nConnection: close\r\n\r\n",
		202, 543, "text/plain; encoding=utf-8", "")
	if !h.ConnectionClose() {
		t.Fatalf("expecting connection: close")
	}

	// tranfer-encoding: chunked
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 505 Internal error\r\nContent-Type: text/html\r\nTransfer-Encoding: chunked\r\n\r\n",
		505, -1, "text/html", "")
	if h.ConnectionClose() {
		t.Fatalf("unexpected connection: close")
	}

	// reverse order of content-type and tranfer-encoding
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 343 foobar\r\nTransfer-Encoding: chunked\r\nContent-Type: text/json\r\n\r\n",
		343, -1, "text/json", "")

	// additional headers
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 100 Continue\r\nFoobar: baz\r\nContent-Type: aaa/bbb\r\nUser-Agent: x\r\nContent-Length: 123\r\nZZZ: werer\r\n\r\n",
		100, 123, "aaa/bbb", "")

	// trailer (aka body)
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 32245\r\n\r\nqwert aaa",
		200, 32245, "text/plain", "qwert aaa")

	// ancient http protocol
	testResponseHeaderReadSuccess(t, h, "HTTP/0.9 300 OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n\r\nqqqq",
		300, 123, "text/html", "qqqq")

	// lf instead of crlf
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\nContent-Length: 123\nContent-Type: text/html\n\n",
		200, 123, "text/html", "")

	// Zero-length headers with mixed crlf and lf
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 400 OK\nContent-Length: 345\nZero-Value: \r\nContent-Type: aaa\n: zero-key\r\n\r\nooa",
		400, 345, "aaa", "ooa")

	// No space after colon
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\nContent-Length:34\nContent-Type: sss\n\naaaa",
		200, 34, "sss", "aaaa")

	// invalid case
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 400 OK\nconTEnt-leNGTH: 123\nConTENT-TYPE: ass\n\n",
		400, 123, "ass", "")

	// duplicate content-length
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Length: 456\r\nContent-Type: foo/bar\r\nContent-Length: 321\r\n\r\n",
		200, 321, "foo/bar", "")

	// duplicate content-type
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Length: 234\r\nContent-Type: foo/bar\r\nContent-Type: baz/bar\r\n\r\n",
		200, 234, "baz/bar", "")

	// both transfer-encoding: chunked and content-length
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Type: foo/bar\r\nContent-Length: 123\r\nTransfer-Encoding: chunked\r\n\r\n",
		200, -1, "foo/bar", "")

	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 300 OK\r\nContent-Type: foo/barr\r\nTransfer-Encoding: chunked\r\nContent-Length: 354\r\n\r\n",
		300, -1, "foo/barr", "")

	// duplicate transfer-encoding: chunked
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nTransfer-Encoding: chunked\r\nTransfer-Encoding: chunked\r\n\r\n",
		200, -1, "text/html", "")

	// no reason string in the first line
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 456\r\nContent-Type: xxx/yyy\r\nContent-Length: 134\r\n\r\naaaxxx",
		456, 134, "xxx/yyy", "aaaxxx")

	// blank lines before the first line
	testResponseHeaderReadSuccess(t, h, "\r\nHTTP/1.1 200 OK\r\nContent-Type: aa\r\nContent-Length: 0\r\n\r\nsss",
		200, 0, "aa", "sss")
	if h.ConnectionClose() {
		t.Fatalf("unexpected connection: close")
	}

	// no content-length (informational responses)
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 101 OK\r\n\r\n",
		101, -2, "text/plain; charset=utf-8", "")
	if h.ConnectionClose() {
		t.Fatalf("expecting connection: keep-alive for informational response")
	}

	// no content-length (no-content responses)
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 204 OK\r\n\r\n",
		204, -2, "text/plain; charset=utf-8", "")
	if h.ConnectionClose() {
		t.Fatalf("expecting connection: keep-alive for no-content response")
	}

	// no content-length (not-modified responses)
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 304 OK\r\n\r\n",
		304, -2, "text/plain; charset=utf-8", "")
	if h.ConnectionClose() {
		t.Fatalf("expecting connection: keep-alive for not-modified response")
	}

	// no content-length (identity transfer-encoding)
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Type: foo/bar\r\n\r\nabcdefg",
		200, -2, "foo/bar", "abcdefg")
	if !h.ConnectionClose() {
		t.Fatalf("expecting connection: close for identity response")
	}

	// non-numeric content-length
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Length: faaa\r\nContent-Type: text/html\r\n\r\nfoobar",
		200, -2, "text/html", "foobar")
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 201 OK\r\nContent-Length: 123aa\r\nContent-Type: text/ht\r\n\r\naaa",
		201, -2, "text/ht", "aaa")
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Length: aa124\r\nContent-Type: html\r\n\r\nxx",
		200, -2, "html", "xx")

	// no content-type
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 400 OK\r\nContent-Length: 123\r\n\r\nfoiaaa",
		400, 123, string(defaultContentType), "foiaaa")

	// no headers
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\n\r\naaaabbb",
		200, -2, string(defaultContentType), "aaaabbb")
	if !h.IsHTTP11() {
		t.Fatalf("expecting http/1.1 protocol")
	}

	// ancient http protocol
	testResponseHeaderReadSuccess(t, h, "HTTP/1.0 203 OK\r\nContent-Length: 123\r\nContent-Type: foobar\r\n\r\naaa",
		203, 123, "foobar", "aaa")
	if h.IsHTTP11() {
		t.Fatalf("ancient protocol must be non-http/1.1")
	}
	if !h.ConnectionClose() {
		t.Fatalf("expecting connection: close for ancient protocol")
	}

	// ancient http protocol with 'Connection: keep-alive' header.
	testResponseHeaderReadSuccess(t, h, "HTTP/1.0 403 aa\r\nContent-Length: 0\r\nContent-Type: 2\r\nConnection: Keep-Alive\r\n\r\nww",
		403, 0, "2", "ww")
	if h.IsHTTP11() {
		t.Fatalf("ancient protocol must be non-http/1.1")
	}
	if h.ConnectionClose() {
		t.Fatalf("expecting connection: keep-alive for ancient protocol")
	}
}

func TestRequestHeaderReadSuccess(t *testing.T) {
	h := &RequestHeader{}

	// simple headers
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar HTTP/1.1\r\nHost: google.com\r\n\r\n",
		0, "/foo/bar", "google.com", "", "", "")
	if h.ConnectionClose() {
		t.Fatalf("unexpected connection: close header")
	}

	// simple headers with body
	testRequestHeaderReadSuccess(t, h, "GET /a/bar HTTP/1.1\r\nHost: gole.com\r\nconneCTION: close\r\n\r\nfoobar",
		0, "/a/bar", "gole.com", "", "", "foobar")
	if !h.ConnectionClose() {
		t.Fatalf("connection: close unset")
	}

	// ancient http protocol
	testRequestHeaderReadSuccess(t, h, "GET /bar HTTP/1.0\r\nHost: gole\r\n\r\npppp",
		0, "/bar", "gole", "", "", "pppp")
	if h.IsHTTP11() {
		t.Fatalf("ancient http protocol cannot be http/1.1")
	}
	if !h.ConnectionClose() {
		t.Fatalf("expecting connectionClose for ancient http protocol")
	}

	// ancient http protocol with 'Connection: keep-alive' header
	testRequestHeaderReadSuccess(t, h, "GET /aa HTTP/1.0\r\nHost: bb\r\nConnection: keep-alive\r\n\r\nxxx",
		0, "/aa", "bb", "", "", "xxx")
	if h.IsHTTP11() {
		t.Fatalf("ancient http protocol cannot be http/1.1")
	}
	if h.ConnectionClose() {
		t.Fatalf("unexpected 'connection: close' for ancient http protocol")
	}

	// complex headers with body
	testRequestHeaderReadSuccess(t, h, "GET /aabar HTTP/1.1\r\nAAA: bbb\r\nHost: ole.com\r\nAA: bb\r\n\r\nzzz",
		0, "/aabar", "ole.com", "", "", "zzz")
	if !h.IsHTTP11() {
		t.Fatalf("expecting http/1.1 protocol")
	}
	if h.ConnectionClose() {
		t.Fatalf("unexpected connection: close")
	}

	// lf instead of crlf
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar HTTP/1.1\nHost: google.com\n\n",
		0, "/foo/bar", "google.com", "", "", "")

	// post method
	testRequestHeaderReadSuccess(t, h, "POST /aaa?bbb HTTP/1.1\r\nHost: foobar.com\r\nContent-Length: 1235\r\nContent-Type: aaa\r\n\r\nabcdef",
		1235, "/aaa?bbb", "foobar.com", "", "aaa", "abcdef")

	// zero-length headers with mixed crlf and lf
	testRequestHeaderReadSuccess(t, h, "GET /a HTTP/1.1\nHost: aaa\r\nZero: \n: Zero-Value\n\r\nxccv",
		0, "/a", "aaa", "", "", "xccv")

	// no space after colon
	testRequestHeaderReadSuccess(t, h, "GET /a HTTP/1.1\nHost:aaaxd\n\nsdfds",
		0, "/a", "aaaxd", "", "", "sdfds")

	// get with zero content-length
	testRequestHeaderReadSuccess(t, h, "GET /xxx HTTP/1.1\nHost: aaa.com\nContent-Length: 0\n\n",
		0, "/xxx", "aaa.com", "", "", "")

	// get with non-zero content-length
	testRequestHeaderReadSuccess(t, h, "GET /xxx HTTP/1.1\nHost: aaa.com\nContent-Length: 123\n\n",
		0, "/xxx", "aaa.com", "", "", "")

	// invalid case
	testRequestHeaderReadSuccess(t, h, "GET /aaa HTTP/1.1\nhoST: bbb.com\n\naas",
		0, "/aaa", "bbb.com", "", "", "aas")

	// referer
	testRequestHeaderReadSuccess(t, h, "GET /asdf HTTP/1.1\nHost: aaa.com\nReferer: bb.com\n\naaa",
		0, "/asdf", "aaa.com", "bb.com", "", "aaa")

	// duplicate host
	testRequestHeaderReadSuccess(t, h, "GET /aa HTTP/1.1\r\nHost: aaaaaa.com\r\nHost: bb.com\r\n\r\n",
		0, "/aa", "bb.com", "", "", "")

	// post with duplicate content-type
	testRequestHeaderReadSuccess(t, h, "POST /a HTTP/1.1\r\nHost: aa\r\nContent-Type: ab\r\nContent-Length: 123\r\nContent-Type: xx\r\n\r\n",
		123, "/a", "aa", "", "xx", "")

	// post with duplicate content-length
	testRequestHeaderReadSuccess(t, h, "POST /xx HTTP/1.1\r\nHost: aa\r\nContent-Type: s\r\nContent-Length: 13\r\nContent-Length: 1\r\n\r\n",
		1, "/xx", "aa", "", "s", "")

	// non-post with content-type
	testRequestHeaderReadSuccess(t, h, "GET /aaa HTTP/1.1\r\nHost: bbb.com\r\nContent-Type: aaab\r\n\r\n",
		0, "/aaa", "bbb.com", "", "aaab", "")

	// non-post with content-length
	testRequestHeaderReadSuccess(t, h, "HEAD / HTTP/1.1\r\nHost: aaa.com\r\nContent-Length: 123\r\n\r\n",
		0, "/", "aaa.com", "", "", "")

	// non-post with content-type and content-length
	testRequestHeaderReadSuccess(t, h, "GET /aa HTTP/1.1\r\nHost: aa.com\r\nContent-Type: abd/test\r\nContent-Length: 123\r\n\r\n",
		0, "/aa", "aa.com", "", "abd/test", "")

	// request uri with hostname
	testRequestHeaderReadSuccess(t, h, "GET http://gooGle.com/foO/%20bar?xxx#aaa HTTP/1.1\r\nHost: aa.cOM\r\n\r\ntrail",
		0, "http://gooGle.com/foO/%20bar?xxx#aaa", "aa.cOM", "", "", "trail")

	// no protocol in the first line
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar\r\nHost: google.com\r\n\r\nisdD",
		0, "/foo/bar", "google.com", "", "", "isdD")

	// blank lines before the first line
	testRequestHeaderReadSuccess(t, h, "\r\n\n\r\nGET /aaa HTTP/1.1\r\nHost: aaa.com\r\n\r\nsss",
		0, "/aaa", "aaa.com", "", "", "sss")

	// request uri with spaces
	testRequestHeaderReadSuccess(t, h, "GET /foo/ bar baz HTTP/1.1\r\nHost: aa.com\r\n\r\nxxx",
		0, "/foo/ bar baz", "aa.com", "", "", "xxx")

	// no host
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar HTTP/1.1\r\nFOObar: assdfd\r\n\r\naaa",
		0, "/foo/bar", "", "", "", "aaa")

	// no host, no headers
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar HTTP/1.1\r\n\r\nfoobar",
		0, "/foo/bar", "", "", "", "foobar")

	// post with invalid content-length
	testRequestHeaderReadSuccess(t, h, "POST /a HTTP/1.1\r\nHost: bb\r\nContent-Type: aa\r\nContent-Length: dff\r\n\r\nqwerty",
		-2, "/a", "bb", "", "aa", "qwerty")

	// post without content-length and content-type
	testRequestHeaderReadSuccess(t, h, "POST /aaa HTTP/1.1\r\nHost: aaa.com\r\n\r\nzxc",
		-2, "/aaa", "aaa.com", "", "", "zxc")

	// post without content-type
	testRequestHeaderReadSuccess(t, h, "POST /abc HTTP/1.1\r\nHost: aa.com\r\nContent-Length: 123\r\n\r\npoiuy",
		123, "/abc", "aa.com", "", "", "poiuy")

	// post without content-length
	testRequestHeaderReadSuccess(t, h, "POST /abc HTTP/1.1\r\nHost: aa.com\r\nContent-Type: adv\r\n\r\n123456",
		-2, "/abc", "aa.com", "", "adv", "123456")

	// invalid method
	testRequestHeaderReadSuccess(t, h, "POST /foo/bar HTTP/1.1\r\nHost: google.com\r\n\r\nmnbv",
		-2, "/foo/bar", "google.com", "", "", "mnbv")

	// put request
	testRequestHeaderReadSuccess(t, h, "PUT /faa HTTP/1.1\r\nHost: aaa.com\r\nContent-Length: 123\r\nContent-Type: aaa\r\n\r\nxwwere",
		123, "/faa", "aaa.com", "", "aaa", "xwwere")
}

func TestResponseHeaderReadError(t *testing.T) {
	h := &ResponseHeader{}

	// incorrect first line
	testResponseHeaderReadError(t, h, "")
	testResponseHeaderReadError(t, h, "fo")
	testResponseHeaderReadError(t, h, "foobarbaz")
	testResponseHeaderReadError(t, h, "HTTP/1.1")
	testResponseHeaderReadError(t, h, "HTTP/1.1 ")
	testResponseHeaderReadError(t, h, "HTTP/1.1 s")

	// non-numeric status code
	testResponseHeaderReadError(t, h, "HTTP/1.1 foobar OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n\r\n")
	testResponseHeaderReadError(t, h, "HTTP/1.1 123foobar OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n\r\n")
	testResponseHeaderReadError(t, h, "HTTP/1.1 foobar344 OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n\r\n")

	// no headers
	testResponseHeaderReadError(t, h, "HTTP/1.1 200 OK\r\n")

	// no trailing crlf
	testResponseHeaderReadError(t, h, "HTTP/1.1 200 OK\r\nContent-Length: 123\r\nContent-Type: text/html\r\n")
}

func TestRequestHeaderReadError(t *testing.T) {
	h := &RequestHeader{}

	// incorrect first line
	testRequestHeaderReadError(t, h, "")
	testRequestHeaderReadError(t, h, "fo")
	testRequestHeaderReadError(t, h, "GET ")
	testRequestHeaderReadError(t, h, "GET / HTTP/1.1\r")

	// missing RequestURI
	testRequestHeaderReadError(t, h, "GET  HTTP/1.1\r\nHost: google.com\r\n\r\n")
}

func testResponseHeaderReadError(t *testing.T, h *ResponseHeader, headers string) {
	r := bytes.NewBufferString(headers)
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err == nil {
		t.Fatalf("Expecting error when reading response header %q", headers)
	}

	// make sure response header works after error
	testResponseHeaderReadSuccess(t, h, "HTTP/1.1 200 OK\r\nContent-Type: foo/bar\r\nContent-Length: 12345\r\n\r\nsss",
		200, 12345, "foo/bar", "sss")
}

func testRequestHeaderReadError(t *testing.T, h *RequestHeader, headers string) {
	r := bytes.NewBufferString(headers)
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err == nil {
		t.Fatalf("Expecting error when reading request header %q", headers)
	}

	// make sure request header works after error
	testRequestHeaderReadSuccess(t, h, "GET /foo/bar HTTP/1.1\r\nHost: aaaa\r\n\r\nxxx",
		0, "/foo/bar", "aaaa", "", "", "xxx")
}

func testResponseHeaderReadSuccess(t *testing.T, h *ResponseHeader, headers string, expectedStatusCode, expectedContentLength int,
	expectedContentType, expectedTrailer string) {
	r := bytes.NewBufferString(headers)
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err != nil {
		t.Fatalf("Unexpected error when parsing response headers: %s. headers=%q", err, headers)
	}
	verifyResponseHeader(t, h, expectedStatusCode, expectedContentLength, expectedContentType)
	verifyTrailer(t, br, expectedTrailer)
}

func testRequestHeaderReadSuccess(t *testing.T, h *RequestHeader, headers string, expectedContentLength int,
	expectedRequestURI, expectedHost, expectedReferer, expectedContentType, expectedTrailer string) {
	r := bytes.NewBufferString(headers)
	br := bufio.NewReader(r)
	err := h.Read(br)
	if err != nil {
		t.Fatalf("Unexpected error when parsing request headers: %s. headers=%q", err, headers)
	}
	verifyRequestHeader(t, h, expectedContentLength, expectedRequestURI, expectedHost, expectedReferer, expectedContentType)
	verifyTrailer(t, br, expectedTrailer)
}

func verifyResponseHeader(t *testing.T, h *ResponseHeader, expectedStatusCode, expectedContentLength int, expectedContentType string) {
	if h.StatusCode() != expectedStatusCode {
		t.Fatalf("Unexpected status code %d. Expected %d", h.StatusCode(), expectedStatusCode)
	}
	if h.ContentLength() != expectedContentLength {
		t.Fatalf("Unexpected content length %d. Expected %d", h.ContentLength(), expectedContentLength)
	}
	if string(h.Peek("Content-Type")) != expectedContentType {
		t.Fatalf("Unexpected content type %q. Expected %q", h.Peek("Content-Type"), expectedContentType)
	}
}

func verifyRequestHeader(t *testing.T, h *RequestHeader, expectedContentLength int,
	expectedRequestURI, expectedHost, expectedReferer, expectedContentType string) {
	if h.ContentLength() != expectedContentLength {
		t.Fatalf("Unexpected Content-Length %d. Expected %d", h.ContentLength(), expectedContentLength)
	}
	if string(h.RequestURI()) != expectedRequestURI {
		t.Fatalf("Unexpected RequestURI %q. Expected %q", h.RequestURI(), expectedRequestURI)
	}
	if string(h.Peek("Host")) != expectedHost {
		t.Fatalf("Unexpected host %q. Expected %q", h.Peek("Host"), expectedHost)
	}
	if string(h.Peek("Referer")) != expectedReferer {
		t.Fatalf("Unexpected referer %q. Expected %q", h.Peek("Referer"), expectedReferer)
	}
	if string(h.Peek("Content-Type")) != expectedContentType {
		t.Fatalf("Unexpected content-type %q. Expected %q", h.Peek("Content-Type"), expectedContentType)
	}
}

func verifyTrailer(t *testing.T, r *bufio.Reader, expectedTrailer string) {
	trailer, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("Cannot read trailer: %s", err)
	}
	if !bytes.Equal(trailer, []byte(expectedTrailer)) {
		t.Fatalf("Unexpected trailer %q. Expected %q", trailer, expectedTrailer)
	}
}
