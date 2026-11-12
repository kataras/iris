package errors

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14/context"
)

// Filter accepts the request context; a nil context is tolerated for direct
// callers, and context-unaware mappers keep working unchanged.
func TestErrorCodeMapFilterWithContext(t *testing.T) {
	sentinel := fmt.Errorf("filter sentinel")

	m := make(ErrorCodeMap)
	m.MapErrors(NotFound, sentinel)

	var sawNilCtx bool
	m.MapErrorFuncCtx(InvalidArgument, func(ctx *context.Context, err error) error {
		if ctx == nil {
			sawNilCtx = true
		}
		return nil // never claims the error.
	})

	code, err := m.Filter(nil, sentinel)
	if code != NotFound {
		t.Fatalf("expected NotFound, got %q", code)
	}
	if err != sentinel {
		t.Fatalf("expected the original error back, got %v", err)
	}

	// The ctx-aware mapper ran (registered under a lower status code) and
	// received the nil context without panicking.
	if !sawNilCtx {
		t.Fatal("expected the ctx-aware mapper to run with a nil context")
	}

	code, _ = m.Filter(nil, fmt.Errorf("unmapped"))
	if code != "" {
		t.Fatalf("expected no mapping for an unknown error, got %q", code)
	}
}
