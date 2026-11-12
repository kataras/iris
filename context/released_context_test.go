package context_test

import (
	stdContext "context"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// An Iris Context handed to code that outlives the handler must behave as a
// canceled standard context once the router has released it to the pool, not
// dereference the request it no longer owns. pgx does exactly this: when the
// acquirer gives up mid-dial, jackc/puddle keeps dialing with the acquirer's
// context for Value lookups (valueCancelCtx), and the Go resolver's
// context.WithDeadline then calls Value on the released Context. That nil
// dereference took a production service down in a crash loop on 2026-10-14.
func TestReleasedContextBehavesAsCanceledStdContext(t *testing.T) {
	var escaped stdContext.Context

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		escaped = ctx // the Iris Context itself, used as a context.Context.
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	// The router has released the Context back to its pool by now.

	select {
	case <-escaped.Done():
	default:
		t.Fatal("Done() of a released Context must be closed")
	}

	if err := escaped.Err(); err != stdContext.Canceled {
		t.Fatalf("Err() of a released Context = %v, want context.Canceled", err)
	}

	if deadline, ok := escaped.Deadline(); ok {
		t.Fatalf("Deadline() of a released Context = %v, want none", deadline)
	}

	if v := escaped.Value("user"); v != nil {
		t.Fatalf("Value() of a released Context = %v, want nil", v)
	}

	// The exact call that crashed production: a child context derived from
	// the released Context (net.Resolver.exchange -> context.WithDeadline ->
	// propagateCancel -> parent.Value).
	child, cancel := stdContext.WithTimeout(escaped, time.Second)
	defer cancel()

	if err := child.Err(); err != stdContext.Canceled {
		t.Fatalf("child of a released Context: Err() = %v, want context.Canceled", err)
	}
}
