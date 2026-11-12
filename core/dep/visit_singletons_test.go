package dep_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14/core/dep"
)

type thing struct{ name string }

type other struct{ t *thing }

func TestVisitSingletons(t *testing.T) {
	c := dep.NewContainer()

	th := &thing{name: "a"}
	if err := c.RegisterDependency(th); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterDependency(func(x *thing) *other { return &other{t: x} }); err != nil {
		t.Fatal(err)
	}
	// Request-scoped (time.Time input): resolved per request, not visited.
	if err := c.RegisterDependency(func(now time.Time) *thing { return &thing{name: "req"} }); err != nil {
		t.Fatal(err)
	}

	var got []any
	c.VisitSingletons(func(v any) { got = append(got, v) })

	if len(got) != 2 {
		t.Fatalf("expected 2 singleton values, got %d: %#v", len(got), got)
	}
	if got[0] != any(th) {
		t.Fatalf("expected the registered value first, got %#v", got[0])
	}
	if o, ok := got[1].(*other); !ok || o.t != th {
		t.Fatalf("expected the constructor output second, got %#v", got[1])
	}
}

func TestVisitSingletonsDedupesPointerIdentity(t *testing.T) {
	c := dep.NewContainer()

	th := &thing{}
	if err := c.RegisterDependency(th); err != nil {
		t.Fatal(err)
	}
	// A pass-through constructor returns the same pointer: visited once.
	if err := c.RegisterDependency(func(x *thing) *thing { return x }); err != nil {
		t.Fatal(err)
	}

	count := 0
	c.VisitSingletons(func(v any) { count++ })

	if count != 1 {
		t.Fatalf("expected identical pointers to be visited once, got %d visits", count)
	}
}
