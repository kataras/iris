package fasthttp

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestURICopyToQueryArgs(t *testing.T) {
	var u URI
	a := u.QueryArgs()
	a.Set("foo", "bar")

	var u1 URI
	u.CopyTo(&u1)
	a1 := u1.QueryArgs()

	if string(a1.Peek("foo")) != "bar" {
		t.Fatalf("unexpected query args value %q. Expecting %q", a1.Peek("foo"), "bar")
	}
}

func TestURIAcquireReleaseSequential(t *testing.T) {
	testURIAcquireRelease(t)
}

func TestURIAcquireReleaseConcurrent(t *testing.T) {
	ch := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			testURIAcquireRelease(t)
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

func testURIAcquireRelease(t *testing.T) {
	for i := 0; i < 10; i++ {
		u := AcquireURI()
		host := fmt.Sprintf("host.%d.com", i*23)
		path := fmt.Sprintf("/foo/%d/bar", i*17)
		queryArgs := "?foo=bar&baz=aass"
		u.Parse([]byte(host), []byte(path+queryArgs))
		if string(u.Host()) != host {
			t.Fatalf("unexpected host %q. Expecting %q", u.Host(), host)
		}
		if string(u.Path()) != path {
			t.Fatalf("unexpected path %q. Expecting %q", u.Path(), path)
		}
		ReleaseURI(u)
	}
}

func TestURILastPathSegment(t *testing.T) {
	testURILastPathSegment(t, "", "")
	testURILastPathSegment(t, "/", "")
	testURILastPathSegment(t, "/foo/bar/", "")
	testURILastPathSegment(t, "/foobar.js", "foobar.js")
	testURILastPathSegment(t, "/foo/bar/baz.html", "baz.html")
}

func testURILastPathSegment(t *testing.T, path, expectedSegment string) {
	var u URI
	u.SetPath(path)
	segment := u.LastPathSegment()
	if string(segment) != expectedSegment {
		t.Fatalf("unexpected last path segment for path %q: %q. Expecting %q", path, segment, expectedSegment)
	}
}

func TestURIPathEscape(t *testing.T) {
	testURIPathEscape(t, "/foo/bar", "/foo/bar")
	testURIPathEscape(t, "/f_o-o=b:ar,b.c&q", "/f_o-o=b:ar,b.c&q")
	testURIPathEscape(t, "/aa?bb.тест~qq", "/aa%3Fbb.%D1%82%D0%B5%D1%81%D1%82~qq")
}

func testURIPathEscape(t *testing.T, path, expectedRequestURI string) {
	var u URI
	u.SetPath(path)
	requestURI := u.RequestURI()
	if string(requestURI) != expectedRequestURI {
		t.Fatalf("unexpected requestURI %q. Expecting %q. path %q", requestURI, expectedRequestURI, path)
	}
}

func TestURIUpdate(t *testing.T) {
	// full uri
	testURIUpdate(t, "http://foo.bar/baz?aaa=22#aaa", "https://aa.com/bb", "https://aa.com/bb")

	// empty uri
	testURIUpdate(t, "http://aaa.com/aaa.html?234=234#add", "", "http://aaa.com/aaa.html?234=234#add")

	// request uri
	testURIUpdate(t, "ftp://aaa/xxx/yyy?aaa=bb#aa", "/boo/bar?xx", "ftp://aaa/boo/bar?xx")

	// relative uri
	testURIUpdate(t, "http://foo.bar/baz/xxx.html?aaa=22#aaa", "bb.html?xx=12#pp", "http://foo.bar/baz/bb.html?xx=12#pp")
	testURIUpdate(t, "http://xx/a/b/c/d", "../qwe/p?zx=34", "http://xx/a/b/qwe/p?zx=34")
	testURIUpdate(t, "https://qqq/aaa.html?foo=bar", "?baz=434&aaa#xcv", "https://qqq/aaa.html?baz=434&aaa#xcv")
	testURIUpdate(t, "http://foo.bar/baz", "~a/%20b=c,тест?йцу=ке", "http://foo.bar/~a/%20b=c,%D1%82%D0%B5%D1%81%D1%82?йцу=ке")
	testURIUpdate(t, "http://foo.bar/baz", "/qwe#fragment", "http://foo.bar/qwe#fragment")
	testURIUpdate(t, "http://foobar/baz/xxx", "aaa.html#bb?cc=dd&ee=dfd", "http://foobar/baz/aaa.html#bb?cc=dd&ee=dfd")
}

func testURIUpdate(t *testing.T, base, update, result string) {
	var u URI
	u.Parse(nil, []byte(base))
	u.Update(update)
	s := u.String()
	if s != result {
		t.Fatalf("unexpected result %q. Expecting %q. base=%q, update=%q", s, result, base, update)
	}
}

func TestURIPathNormalize(t *testing.T) {
	var u URI

	// double slash
	testURIPathNormalize(t, &u, "/aa//bb", "/aa/bb")

	// triple slash
	testURIPathNormalize(t, &u, "/x///y/", "/x/y/")

	// multi slashes
	testURIPathNormalize(t, &u, "/abc//de///fg////", "/abc/de/fg/")

	// encoded slashes
	testURIPathNormalize(t, &u, "/xxxx%2fyyy%2f%2F%2F", "/xxxx/yyy/")

	// dotdot
	testURIPathNormalize(t, &u, "/aaa/..", "/")

	// dotdot with trailing slash
	testURIPathNormalize(t, &u, "/xxx/yyy/../", "/xxx/")

	// multi dotdots
	testURIPathNormalize(t, &u, "/aaa/bbb/ccc/../../ddd", "/aaa/ddd")

	// dotdots separated by other data
	testURIPathNormalize(t, &u, "/a/b/../c/d/../e/..", "/a/c/")

	// too many dotdots
	testURIPathNormalize(t, &u, "/aaa/../../../../xxx", "/xxx")
	testURIPathNormalize(t, &u, "/../../../../../..", "/")
	testURIPathNormalize(t, &u, "/../../../../../../", "/")

	// encoded dotdots
	testURIPathNormalize(t, &u, "/aaa%2Fbbb%2F%2E.%2Fxxx", "/aaa/xxx")

	// double slash with dotdots
	testURIPathNormalize(t, &u, "/aaa////..//b", "/b")

	// fake dotdot
	testURIPathNormalize(t, &u, "/aaa/..bbb/ccc/..", "/aaa/..bbb/")

	// single dot
	testURIPathNormalize(t, &u, "/a/./b/././c/./d.html", "/a/b/c/d.html")
	testURIPathNormalize(t, &u, "./foo/", "/foo/")
	testURIPathNormalize(t, &u, "./../.././../../aaa/bbb/../../../././../", "/")
}

func testURIPathNormalize(t *testing.T, u *URI, requestURI, expectedPath string) {
	u.Parse(nil, []byte(requestURI))
	if string(u.Path()) != expectedPath {
		t.Fatalf("Unexpected path %q. Expected %q. requestURI=%q", u.Path(), expectedPath, requestURI)
	}
}

func TestURIFullURI(t *testing.T) {
	var args Args

	// empty scheme, path and hash
	testURIFullURI(t, "", "foobar.com", "", "", &args, "http://foobar.com/")

	// empty scheme and hash
	testURIFullURI(t, "", "aa.com", "/foo/bar", "", &args, "http://aa.com/foo/bar")

	// empty hash
	testURIFullURI(t, "fTP", "XXx.com", "/foo", "", &args, "ftp://xxx.com/foo")

	// empty args
	testURIFullURI(t, "https", "xx.com", "/", "aaa", &args, "https://xx.com/#aaa")

	// non-empty args and non-ASCII path
	args.Set("foo", "bar")
	args.Set("xxx", "йух")
	testURIFullURI(t, "", "xxx.com", "/тест123", "2er", &args, "http://xxx.com/%D1%82%D0%B5%D1%81%D1%82123?foo=bar&xxx=%D0%B9%D1%83%D1%85#2er")

	// test with empty args and non-empty query string
	var u URI
	u.Parse([]byte("google.com"), []byte("/foo?bar=baz&baraz#qqqq"))
	uri := u.FullURI()
	expectedURI := "http://google.com/foo?bar=baz&baraz#qqqq"
	if string(uri) != expectedURI {
		t.Fatalf("Unexpected URI: %q. Expected %q", uri, expectedURI)
	}
}

func testURIFullURI(t *testing.T, scheme, host, path, hash string, args *Args, expectedURI string) {
	var u URI

	u.SetScheme(scheme)
	u.SetHost(host)
	u.SetPath(path)
	u.SetHash(hash)
	args.CopyTo(u.QueryArgs())

	uri := u.FullURI()
	if string(uri) != expectedURI {
		t.Fatalf("Unexpected URI: %q. Expected %q", uri, expectedURI)
	}
}

func TestURIParseNilHost(t *testing.T) {
	testURIParseScheme(t, "http://google.com/foo?bar#baz", "http")
	testURIParseScheme(t, "HTtP://google.com/", "http")
	testURIParseScheme(t, "://google.com/", "http")
	testURIParseScheme(t, "fTP://aaa.com", "ftp")
	testURIParseScheme(t, "httPS://aaa.com", "https")
}

func testURIParseScheme(t *testing.T, uri, expectedScheme string) {
	var u URI
	u.Parse(nil, []byte(uri))
	if string(u.Scheme()) != expectedScheme {
		t.Fatalf("Unexpected scheme %q. Expected %q for uri %q", u.Scheme(), expectedScheme, uri)
	}
}

func TestURIParse(t *testing.T) {
	var u URI

	// no args
	testURIParse(t, &u, "aaa", "sdfdsf",
		"http://aaa/sdfdsf", "aaa", "/sdfdsf", "sdfdsf", "", "")

	// args
	testURIParse(t, &u, "xx", "/aa?ss",
		"http://xx/aa?ss", "xx", "/aa", "/aa", "ss", "")

	// args and hash
	testURIParse(t, &u, "foobar.com", "/a.b.c?def=gkl#mnop",
		"http://foobar.com/a.b.c?def=gkl#mnop", "foobar.com", "/a.b.c", "/a.b.c", "def=gkl", "mnop")

	// '?' and '#' in hash
	testURIParse(t, &u, "aaa.com", "/foo#bar?baz=aaa#bbb",
		"http://aaa.com/foo#bar?baz=aaa#bbb", "aaa.com", "/foo", "/foo", "", "bar?baz=aaa#bbb")

	// encoded path
	testURIParse(t, &u, "aa.com", "/Test%20+%20%D0%BF%D1%80%D0%B8?asdf=%20%20&s=12#sdf",
		"http://aa.com/Test%20%2B%20%D0%BF%D1%80%D0%B8?asdf=%20%20&s=12#sdf", "aa.com", "/Test + при", "/Test%20+%20%D0%BF%D1%80%D0%B8", "asdf=%20%20&s=12", "sdf")

	// host in uppercase
	testURIParse(t, &u, "FOObar.COM", "/bC?De=F#Gh",
		"http://foobar.com/bC?De=F#Gh", "foobar.com", "/bC", "/bC", "De=F", "Gh")

	// uri with hostname
	testURIParse(t, &u, "xxx.com", "http://aaa.com/foo/bar?baz=aaa#ddd",
		"http://aaa.com/foo/bar?baz=aaa#ddd", "aaa.com", "/foo/bar", "/foo/bar", "baz=aaa", "ddd")
	testURIParse(t, &u, "xxx.com", "https://ab.com/f/b%20r?baz=aaa#ddd",
		"https://ab.com/f/b%20r?baz=aaa#ddd", "ab.com", "/f/b r", "/f/b%20r", "baz=aaa", "ddd")

	// no slash after hostname in uri
	testURIParse(t, &u, "aaa.com", "http://google.com",
		"http://google.com/", "google.com", "/", "/", "", "")

	// uppercase hostname in uri
	testURIParse(t, &u, "abc.com", "http://GoGLE.com/aaa",
		"http://gogle.com/aaa", "gogle.com", "/aaa", "/aaa", "", "")

	// http:// in query params
	testURIParse(t, &u, "aaa.com", "/foo?bar=http://google.com",
		"http://aaa.com/foo?bar=http://google.com", "aaa.com", "/foo", "/foo", "bar=http://google.com", "")
}

func testURIParse(t *testing.T, u *URI, host, uri,
	expectedURI, expectedHost, expectedPath, expectedPathOriginal, expectedArgs, expectedHash string) {
	u.Parse([]byte(host), []byte(uri))

	if !bytes.Equal(u.FullURI(), []byte(expectedURI)) {
		t.Fatalf("Unexpected uri %q. Expected %q. host=%q, uri=%q", u.FullURI(), expectedURI, host, uri)
	}
	if !bytes.Equal(u.Host(), []byte(expectedHost)) {
		t.Fatalf("Unexpected host %q. Expected %q. host=%q, uri=%q", u.Host(), expectedHost, host, uri)
	}
	if !bytes.Equal(u.PathOriginal(), []byte(expectedPathOriginal)) {
		t.Fatalf("Unexpected original path %q. Expected %q. host=%q, uri=%q", u.PathOriginal(), expectedPathOriginal, host, uri)
	}
	if !bytes.Equal(u.Path(), []byte(expectedPath)) {
		t.Fatalf("Unexpected path %q. Expected %q. host=%q, uri=%q", u.Path(), expectedPath, host, uri)
	}
	if !bytes.Equal(u.QueryString(), []byte(expectedArgs)) {
		t.Fatalf("Unexpected args %q. Expected %q. host=%q, uri=%q", u.QueryString(), expectedArgs, host, uri)
	}
	if !bytes.Equal(u.Hash(), []byte(expectedHash)) {
		t.Fatalf("Unexpected hash %q. Expected %q. host=%q, uri=%q", u.Hash(), expectedHash, host, uri)
	}
}
