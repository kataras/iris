package fasthttp

import (
	"testing"
)

func BenchmarkCookieParseMin(b *testing.B) {
	var c Cookie
	s := []byte("xxx=yyy")
	for i := 0; i < b.N; i++ {
		if err := c.ParseBytes(s); err != nil {
			b.Fatalf("unexpected error when parsing cookies: %s", err)
		}
	}
}

func BenchmarkCookieParseNoExpires(b *testing.B) {
	var c Cookie
	s := []byte("xxx=yyy; domain=foobar.com; path=/a/b")
	for i := 0; i < b.N; i++ {
		if err := c.ParseBytes(s); err != nil {
			b.Fatalf("unexpected error when parsing cookies: %s", err)
		}
	}
}

func BenchmarkCookieParseFull(b *testing.B) {
	var c Cookie
	s := []byte("xxx=yyy; expires=Tue, 10 Nov 2009 23:00:00 GMT; domain=foobar.com; path=/a/b")
	for i := 0; i < b.N; i++ {
		if err := c.ParseBytes(s); err != nil {
			b.Fatalf("unexpected error when parsing cookies: %s", err)
		}
	}
}
