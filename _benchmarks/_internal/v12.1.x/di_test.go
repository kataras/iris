package main

import (
	"testing"

	"github.com/kataras/iris/v12/hero"
)

func BenchmarkHero(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := hero.New()
		c.Register(dependency)
		_ = c.Handler(handler)
	}
}
