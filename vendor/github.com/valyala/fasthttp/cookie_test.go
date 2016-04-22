package fasthttp

import (
	"strings"
	"testing"
	"time"
)

func TestCookieSecureHttpOnly(t *testing.T) {
	var c Cookie

	if err := c.Parse("foo=bar; HttpOnly; secure"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !c.Secure() {
		t.Fatalf("secure must be set")
	}
	if !c.HTTPOnly() {
		t.Fatalf("HttpOnly must be set")
	}
	s := c.String()
	if !strings.Contains(s, "; secure") {
		t.Fatalf("missing secure flag in cookie %q", s)
	}
	if !strings.Contains(s, "; HttpOnly") {
		t.Fatalf("missing HttpOnly flag in cookie %q", s)
	}
}

func TestCookieSecure(t *testing.T) {
	var c Cookie

	if err := c.Parse("foo=bar; secure"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !c.Secure() {
		t.Fatalf("secure must be set")
	}
	s := c.String()
	if !strings.Contains(s, "; secure") {
		t.Fatalf("missing secure flag in cookie %q", s)
	}

	if err := c.Parse("foo=bar"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if c.HTTPOnly() {
		t.Fatalf("Unexpected secure flag set")
	}
	s = c.String()
	if strings.Contains(s, "secure") {
		t.Fatalf("unexpected secure flag in cookie %q", s)
	}
}

func TestCookieHttpOnly(t *testing.T) {
	var c Cookie

	if err := c.Parse("foo=bar; HttpOnly"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !c.HTTPOnly() {
		t.Fatalf("HTTPOnly must be set")
	}
	s := c.String()
	if !strings.Contains(s, "; HttpOnly") {
		t.Fatalf("missing HttpOnly flag in cookie %q", s)
	}

	if err := c.Parse("foo=bar"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if c.HTTPOnly() {
		t.Fatalf("Unexpected HTTPOnly flag set")
	}
	s = c.String()
	if strings.Contains(s, "HttpOnly") {
		t.Fatalf("unexpected HttpOnly flag in cookie %q", s)
	}
}

func TestCookieAcquireReleaseSequential(t *testing.T) {
	testCookieAcquireRelease(t)
}

func TestCookieAcquireReleaseConcurrent(t *testing.T) {
	ch := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			testCookieAcquireRelease(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testCookieAcquireRelease(t *testing.T) {
	c := AcquireCookie()

	key := "foo"
	c.SetKey(key)

	value := "bar"
	c.SetValue(value)

	domain := "foo.bar.com"
	c.SetDomain(domain)

	path := "/foi/bar/aaa"
	c.SetPath(path)

	s := c.String()
	c.Reset()
	if err := c.Parse(s); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if string(c.Key()) != key {
		t.Fatalf("unexpected cookie name %q. Expecting %q", c.Key(), key)
	}
	if string(c.Value()) != value {
		t.Fatalf("unexpected cookie value %q. Expecting %q", c.Value(), value)
	}
	if string(c.Domain()) != domain {
		t.Fatalf("unexpected domain %q. Expecting %q", c.Domain(), domain)
	}
	if string(c.Path()) != path {
		t.Fatalf("unexpected path %q. Expecting %q", c.Path(), path)
	}

	ReleaseCookie(c)
}

func TestCookieParse(t *testing.T) {
	testCookieParse(t, "foo", "foo")
	testCookieParse(t, "foo=bar", "foo=bar")
	testCookieParse(t, "foo=", "foo=")
	testCookieParse(t, "foo=bar; domain=aaa.com; path=/foo/bar", "foo=bar; domain=aaa.com; path=/foo/bar")
	testCookieParse(t, " xxx = yyy  ; path=/a/b;;;domain=foobar.com ; expires= Tue, 10 Nov 2009 23:00:00 GMT ; ;;",
		"xxx=yyy; expires=Tue, 10 Nov 2009 23:00:00 GMT; domain=foobar.com; path=/a/b")
}

func testCookieParse(t *testing.T, s, expectedS string) {
	var c Cookie
	if err := c.Parse(s); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	result := string(c.Cookie())
	if result != expectedS {
		t.Fatalf("unexpected cookies %q. Expected %q. Original %q", result, expectedS, s)
	}
}

func TestCookieAppendBytes(t *testing.T) {
	c := &Cookie{}

	testCookieAppendBytes(t, c, "", "bar", "bar")
	testCookieAppendBytes(t, c, "foo", "", "foo=")
	testCookieAppendBytes(t, c, "ффф", "12 лодлы", "%D1%84%D1%84%D1%84=12%20%D0%BB%D0%BE%D0%B4%D0%BB%D1%8B")

	c.SetDomain("foobar.com")
	testCookieAppendBytes(t, c, "a", "b", "a=b; domain=foobar.com")

	c.SetPath("/a/b")
	testCookieAppendBytes(t, c, "aa", "bb", "aa=bb; domain=foobar.com; path=/a/b")

	c.SetExpire(CookieExpireDelete)
	testCookieAppendBytes(t, c, "xxx", "yyy", "xxx=yyy; expires=Tue, 10 Nov 2009 23:00:00 GMT; domain=foobar.com; path=/a/b")
}

func testCookieAppendBytes(t *testing.T, c *Cookie, key, value, expectedS string) {
	c.SetKey(key)
	c.SetValue(value)
	result := string(c.AppendBytes(nil))
	if result != expectedS {
		t.Fatalf("Unexpected cookie %q. Expected %q", result, expectedS)
	}
}

func TestParseRequestCookies(t *testing.T) {
	testParseRequestCookies(t, "", "")
	testParseRequestCookies(t, "=", "")
	testParseRequestCookies(t, "foo", "foo")
	testParseRequestCookies(t, "=foo", "foo")
	testParseRequestCookies(t, "bar=", "bar=")
	testParseRequestCookies(t, "xxx=aa;bb=c; =d; ;;e=g", "xxx=aa; bb=c; d; e=g")
	testParseRequestCookies(t, "a;b;c; d=1;d=2", "a; b; c; d=1; d=2")
	testParseRequestCookies(t, "   %D0%B8%D0%B2%D0%B5%D1%82=a%20b%3Bc   ;s%20s=aaa  ", "%D0%B8%D0%B2%D0%B5%D1%82=a%20b%3Bc; s%20s=aaa")
}

func testParseRequestCookies(t *testing.T, s, expectedS string) {
	cookies := parseRequestCookies(nil, []byte(s))
	ss := string(appendRequestCookieBytes(nil, cookies))
	if ss != expectedS {
		t.Fatalf("Unexpected cookies after parsing: %q. Expected %q. String to parse %q", ss, expectedS, s)
	}
}

func TestAppendRequestCookieBytes(t *testing.T) {
	testAppendRequestCookieBytes(t, "=", "")
	testAppendRequestCookieBytes(t, "foo=", "foo=")
	testAppendRequestCookieBytes(t, "=bar", "bar")
	testAppendRequestCookieBytes(t, "привет=a b;c&s s=aaa", "%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82=a%20b%3Bc; s%20s=aaa")
}

func testAppendRequestCookieBytes(t *testing.T, s, expectedS string) {
	var cookies []argsKV
	for _, ss := range strings.Split(s, "&") {
		tmp := strings.SplitN(ss, "=", 2)
		if len(tmp) != 2 {
			t.Fatalf("Cannot find '=' in %q, part of %q", ss, s)
		}
		cookies = append(cookies, argsKV{
			key:   []byte(tmp[0]),
			value: []byte(tmp[1]),
		})
	}

	prefix := "foobar"
	result := string(appendRequestCookieBytes([]byte(prefix), cookies))
	if result[:len(prefix)] != prefix {
		t.Fatalf("unexpected prefix %q. Expected %q for cookie %q", result[:len(prefix)], prefix, s)
	}
	result = result[len(prefix):]
	if result != expectedS {
		t.Fatalf("Unexpected result %q. Expected %q for cookie %q", result, expectedS, s)
	}
}
