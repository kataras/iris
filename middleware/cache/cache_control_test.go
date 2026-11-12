package cache

import (
	"strings"
	"testing"
	"time"
)

func TestMaxAgeRejectsSubSecond(t *testing.T) {
	mustPanic := func(name string, fn func(), wantInMsg string) {
		t.Helper()
		defer func() {
			t.Helper()
			r := recover()
			if r == nil {
				t.Fatalf("%s: expected a panic for a sub-second duration", name)
			}
			if msg, _ := r.(string); !strings.Contains(msg, wantInMsg) {
				t.Fatalf("%s: panic message %q does not contain %q", name, r, wantInMsg)
			}
		}()
		fn()
	}

	mustPanic("MaxAge(3600)", func() { NewCacheControlBuilder().MaxAge(3600) }, "MaxAge(3.6µs) is below one second")
	mustPanic("SMaxAge(60)", func() { NewCacheControlBuilder().SMaxAge(60) }, "SMaxAge(60ns) is below one second")
	mustPanic("MaxAge(-1s)", func() { NewCacheControlBuilder().MaxAge(-time.Second) }, "MaxAge(-1s) is negative")

	if got, want := NewCacheControlBuilder().MaxAge(0).Build(), "max-age=0"; got != want {
		t.Fatalf("MaxAge(0): got %q, want %q", got, want)
	}
	if got, want := NewCacheControlBuilder().MaxAge(time.Hour).SMaxAge(2*time.Hour).Build(), "max-age=3600, s-maxage=7200"; got != want {
		t.Fatalf("MaxAge(time.Hour): got %q, want %q", got, want)
	}
}
