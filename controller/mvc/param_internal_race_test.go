package mvc

import (
	"sync"
	"testing"
)

// The controller path-parameter registry is written during routing setup and
// read at request time; concurrent access must be safe (run with -race).
func TestPathParamRegistryConcurrentAccess(t *testing.T) {
	type ctrl struct{ n int }

	var wg sync.WaitGroup
	for i := range 100 {
		c := &ctrl{n: i}
		p := &RawPathParam{Name: "x"}

		wg.Go(func() {
			storePathParam(c, p)
		})
		wg.Go(func() {
			_ = GetPathParam(c)
		})
	}
	wg.Wait()

	// Stored params must be retrievable by controller identity.
	c := &ctrl{n: -1}
	p := &RawPathParam{Name: "y"}
	storePathParam(c, p)
	if got := GetPathParam(c); got != p {
		t.Fatalf("expected stored path param back, got %#v", got)
	}
}
