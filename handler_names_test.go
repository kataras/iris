package iris_test

import (
	"testing"

	iris "github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/funcer"
	resterrors "github.com/kataras/iris/v14/rest/errors"

	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/middleware/httpcost"
	"github.com/kataras/iris/v14/middleware/methodoverride"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/middleware/recovery"
	"github.com/kataras/iris/v14/middleware/referrer"
	"github.com/kataras/iris/v14/middleware/requestid"
)

// TestSetHandlerNameMappingsResolve pins every SetHandlerName registration it
// can construct a real handler for: the friendly name must actually resolve.
//
// This is the tripwire for two silent failure modes that both shipped before
// it existed: a mapping whose pattern never matches the real symbol (the
// rest/errors and referrer mappings were dead for exactly that reason), and a
// compiler change to generated function names (Go 1.27 renamed function
// literals) that stops a pattern from matching. Without this test, either
// failure only shows up as raw Go symbols in route traces and broken
// name-anchored wiring (e.g. the rate-limit exceed handler).
func TestSetHandlerNameMappingsResolve(t *testing.T) {
	tests := []struct {
		name    string
		handler context.Handler
		want    string
	}{
		{"recovery", recovery.Handler, "iris.recovery"},
		{"compress", compress.Handler, "iris.compression"},
		{"rest/errors recovery", resterrors.RecoveryHandler, "iris.errors.recover"},
		{"rate method value", rate.Limit(1, 1, rate.Options{}), "iris.ratelimit"},
		{"referrer", referrer.Handler, "iris.referer"},
		{"requestid closure", requestid.New(), "iris.request.id"},
		{"bodylimit closure", bodylimit.New(1024), "iris.bodylimit"},
		{"methodoverride closure", methodoverride.New(methodoverride.Options{}), "iris.methodoverride"},
		{"httpcost closure", httpcost.New(httpcost.DefaultOptions), "iris.httpcost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := context.HandlerName(tt.handler); got != tt.want {
				t.Fatalf("HandlerName resolved to %q, want %q — the SetHandlerName pattern no longer matches the real symbol", got, tt.want)
			}
		})
	}

	// Not covered here: the auth mapping (auth/auth.go) — constructing an
	// *Auth[T] needs signing keys; its pattern is a suffix-tolerant regex on
	// `(*Auth[...]).VerifyHandler.*`, which survives literal-name changes.
	// Package-scoped patterns (accesslog, cors, sessions, ...) are matched by
	// FILENAME as well as symbol, so they cannot go stale the same way the
	// exact-symbol pins above could.
}

// TestEnableStatusCodeResponseName pins the explicit registration name that
// replaced the old exact-literal SetHandlerName pin, and its idempotency.
func TestEnableStatusCodeResponseName(t *testing.T) {
	app := iris.New()
	app.EnableStatusCodeResponse()
	app.EnableStatusCodeResponse() // must upsert by trusted name, not stack.

	if !app.RoutesHandlers().Exists("EnableStatusCodeResponse") {
		t.Fatal(`expected the "EnableStatusCodeResponse" handler to be registered under its explicit name`)
	}

	count := 0
	app.RoutesHandlers().Walk(func(f *funcer.Func[context.Handler]) bool {
		if f.Name == "EnableStatusCodeResponse" {
			count++
		}
		return true
	})
	if count != 1 {
		t.Fatalf("expected exactly one EnableStatusCodeResponse registration, got %d", count)
	}
}
