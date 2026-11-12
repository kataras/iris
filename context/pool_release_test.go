package context

import (
	"testing"
)

// A released Context goes back to the pool with its values and params re-sliced
// to zero length. The backing array must be zeroed too, or the finished
// request's data - the decoded body, the User and the plaintext password an auth
// middleware puts on it - stays reachable from the pooled Context.
func TestForceReleaseZeroesTheEntries(t *testing.T) {
	pool := New(func() any { return &Context{} })

	ctx := pool.Acquire()
	ctx.values.Set(UserContextKey, &SimpleUser{Username: "admin", Password: "correct horse"})
	ctx.values.Set("tenant", "acme")
	ctx.params.Set("id", "42")

	// Keep the backing arrays: ForceRelease re-slices the Context's own headers.
	values, params := ctx.values, ctx.params
	if len(values) != 2 || len(params) != 1 {
		t.Fatalf("setup: values=%d params=%d, want 2 and 1", len(values), len(params))
	}

	pool.ForceRelease(ctx)

	for i, entry := range values {
		if entry.Key != "" || entry.Value != nil {
			t.Errorf("values[%d] survived the release as %q=%v", i, entry.Key, entry.Value)
		}
	}

	for i, entry := range params {
		if entry.Key != "" || entry.Value != nil {
			t.Errorf("params[%d] survived the release as %q=%v", i, entry.Key, entry.Value)
		}
	}
}

func TestAcquireAfterReleaseStartsEmpty(t *testing.T) {
	pool := New(func() any { return &Context{} })

	ctx := pool.Acquire()
	ctx.values.Set("tenant", "acme")
	pool.ForceRelease(ctx)

	next := pool.Acquire()
	if got := next.values.Get("tenant"); got != nil {
		t.Errorf("a pooled Context handed over the previous request's value: %v", got)
	}

	if len(next.values) != 0 || len(next.params) != 0 {
		t.Errorf("a pooled Context was acquired with %d values and %d params, want none",
			len(next.values), len(next.params))
	}
}
