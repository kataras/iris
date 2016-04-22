package fasthttp

import (
	"testing"
)

func BenchmarkUserDataCustom(b *testing.B) {
	keys := []string{"foobar", "baz", "aaa", "bsdfs"}
	b.RunParallel(func(pb *testing.PB) {
		var u userData
		var v interface{} = u
		for pb.Next() {
			for _, key := range keys {
				u.Set(key, v)
			}
			for _, key := range keys {
				vv := u.Get(key)
				if _, ok := vv.(userData); !ok {
					b.Fatalf("unexpected value %v for key %q", vv, key)
				}
			}
			u.Reset()
		}
	})
}

func BenchmarkUserDataStdMap(b *testing.B) {
	keys := []string{"foobar", "baz", "aaa", "bsdfs"}
	b.RunParallel(func(pb *testing.PB) {
		u := make(map[string]interface{})
		var v interface{} = u
		for pb.Next() {
			for _, key := range keys {
				u[key] = v
			}
			for _, key := range keys {
				vv := u[key]
				if _, ok := vv.(map[string]interface{}); !ok {
					b.Fatalf("unexpected value %v for key %q", vv, key)
				}
			}

			for k := range u {
				delete(u, k)
			}
		}
	})
}
