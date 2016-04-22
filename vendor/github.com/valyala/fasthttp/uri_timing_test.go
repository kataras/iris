package fasthttp

import (
	"testing"
)

func BenchmarkURIParsePath(b *testing.B) {
	benchmarkURIParse(b, "google.com", "/foo/bar")
}

func BenchmarkURIParsePathQueryString(b *testing.B) {
	benchmarkURIParse(b, "google.com", "/foo/bar?query=string&other=value")
}

func BenchmarkURIParsePathQueryStringHash(b *testing.B) {
	benchmarkURIParse(b, "google.com", "/foo/bar?query=string&other=value#hashstring")
}

func BenchmarkURIParseHostname(b *testing.B) {
	benchmarkURIParse(b, "google.com", "http://foobar.com/foo/bar?query=string&other=value#hashstring")
}

func BenchmarkURIFullURI(b *testing.B) {
	host := []byte("foobar.com")
	requestURI := []byte("/foobar/baz?aaa=bbb&ccc=ddd")
	uriLen := len(host) + len(requestURI) + 7

	b.RunParallel(func(pb *testing.PB) {
		var u URI
		u.Parse(host, requestURI)
		for pb.Next() {
			uri := u.FullURI()
			if len(uri) != uriLen {
				b.Fatalf("unexpected uri len %d. Expecting %d", len(uri), uriLen)
			}
		}
	})
}

func benchmarkURIParse(b *testing.B, host, uri string) {
	strHost, strURI := []byte(host), []byte(uri)

	b.RunParallel(func(pb *testing.PB) {
		var u URI
		for pb.Next() {
			u.Parse(strHost, strURI)
		}
	})
}
