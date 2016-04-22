package fasthttp

import (
	"bytes"
	"testing"
)

func BenchmarkArgsParse(b *testing.B) {
	s := []byte("foo=bar&baz=qqq&aaaaa=bbbb")
	b.RunParallel(func(pb *testing.PB) {
		var a Args
		for pb.Next() {
			a.ParseBytes(s)
		}
	})
}

func BenchmarkArgsPeek(b *testing.B) {
	value := []byte("foobarbaz1234")
	key := "foobarbaz"
	b.RunParallel(func(pb *testing.PB) {
		var a Args
		a.SetBytesV(key, value)
		for pb.Next() {
			if !bytes.Equal(a.Peek(key), value) {
				b.Fatalf("unexpected arg value %q. Expecting %q", a.Peek(key), value)
			}
		}
	})
}
