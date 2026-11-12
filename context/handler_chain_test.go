package context

import (
	"reflect"
	"testing"
)

// Distinct top-level functions: HandlerName resolves a program counter, so two
// closures produced by the same factory would share a name. The bodies are
// deliberately DISTINCT too — since Go 1.27 the toolchain may fold
// code-identical functions together, and these fixtures must remain three
// different functions under any toolchain.
func chainHandlerA(ctx *Context) { _ = ctx.Method() }
func chainHandlerB(ctx *Context) { _ = ctx.Path() }
func chainHandlerC(ctx *Context) { ctx.Next() }

func handlerPointers(handlers Handlers) []uintptr {
	ptrs := make([]uintptr, 0, len(handlers))
	for _, h := range handlers {
		ptrs = append(ptrs, reflect.ValueOf(h).Pointer())
	}

	return ptrs
}

func recorder(order *[]string, name string) Handler {
	return func(ctx *Context) {
		*order = append(*order, name)
		ctx.Next()
	}
}

// A route's registered chain is shared by every request for that route: Do
// assigns it to the Context without copying. Appending to it at serve-time must
// never reach its backing array.
func TestAddHandlerDoesNotWriteIntoTheSharedChain(t *testing.T) {
	var order []string

	route := make(Handlers, 0, 4) // spare capacity, as append-built chains have.
	route = append(route, recorder(&order, "a"), recorder(&order, "b"))

	extra := recorder(&order, "extra")

	for i := range 2 {
		order = nil

		ctx := &Context{}
		ctx.Do(route)
		ctx.AddHandler(extra)
		ctx.Next()

		if len(route) != 2 {
			t.Fatalf("[request %d] the route chain grew to %d handlers", i, len(route))
		}

		if spare := route[:cap(route)]; spare[2] != nil {
			t.Fatalf("[request %d] AddHandler wrote into the route's spare capacity", i)
		}
	}
}

func TestNewConditionalHandlerKeepsItsInputsIntact(t *testing.T) {
	var order []string

	// The conditional handlers arrive as a slice with spare capacity, which is
	// what NewConditionalHandler(filter, mySlice...) passes straight through.
	conditional := make(Handlers, 0, 4)
	conditional = append(conditional, recorder(&order, "c"))

	always := func(*Context) bool { return true }
	route := Handlers{
		recorder(&order, "a"),
		NewConditionalHandler(always, conditional...),
		recorder(&order, "b"),
	}

	for i := range 2 {
		order = nil

		ctx := &Context{}
		ctx.Do(route)

		if got, want := order, []string{"a", "c", "b"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("[request %d] execution order = %v, want %v", i, got, want)
		}

		if len(conditional) != 1 {
			t.Fatalf("[request %d] the conditional chain grew to %d handlers", i, len(conditional))
		}

		if spare := conditional[:cap(conditional)]; spare[1] != nil {
			t.Fatalf("[request %d] the route's tail was written into the conditional chain's "+
				"spare capacity; two concurrent requests would race over it", i)
		}
	}
}

func TestNewConditionalHandlerSkipsWhenTheFilterFails(t *testing.T) {
	var order []string

	never := func(*Context) bool { return false }
	route := Handlers{
		recorder(&order, "a"),
		NewConditionalHandler(never, recorder(&order, "c")),
		recorder(&order, "b"),
	}

	ctx := &Context{}
	ctx.Do(route)

	if got, want := order, []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
}

// The last-handler branch appends instead of splicing.
func TestNewConditionalHandlerAsTheLastHandler(t *testing.T) {
	var order []string

	always := func(*Context) bool { return true }
	route := Handlers{
		recorder(&order, "a"),
		NewConditionalHandler(always, recorder(&order, "c")),
	}

	ctx := &Context{}
	ctx.Do(route)

	if got, want := order, []string{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
}

func TestUniqueHandlersRemovesEveryOccurrence(t *testing.T) {
	h1 := Handlers{chainHandlerA, chainHandlerB, chainHandlerA}
	h2 := Handlers{chainHandlerA}

	before := handlerPointers(h1)
	got := UniqueHandlers(h1, h2)

	if len(got) != 1 || reflect.ValueOf(got[0]).Pointer() != reflect.ValueOf(chainHandlerB).Pointer() {
		t.Fatalf("UniqueHandlers returned %d handlers, want only chainHandlerB", len(got))
	}

	if after := handlerPointers(h1); !reflect.DeepEqual(before, after) {
		t.Error("UniqueHandlers modified the input slice")
	}
}

func TestUniqueHandlersWithNothingToRemove(t *testing.T) {
	h1 := Handlers{chainHandlerA, chainHandlerB}

	if got := UniqueHandlers(h1, Handlers{chainHandlerC}); len(got) != 2 {
		t.Fatalf("UniqueHandlers returned %d handlers, want 2", len(got))
	}

	if got := UniqueHandlers(h1, nil); len(got) != 2 {
		t.Fatalf("UniqueHandlers with an empty h2 returned %d handlers, want 2", len(got))
	}
}

func TestUpsertHandlersKeepsTheInputIntact(t *testing.T) {
	h1 := Handlers{chainHandlerA, chainHandlerB}
	before := handlerPointers(h1)

	got := UpsertHandlers(h1, Handlers{chainHandlerA, chainHandlerC})

	if len(got) != 3 {
		t.Fatalf("UpsertHandlers returned %d handlers, want 3", len(got))
	}

	if reflect.ValueOf(got[2]).Pointer() != reflect.ValueOf(chainHandlerC).Pointer() {
		t.Error("UpsertHandlers did not append the new handler last")
	}

	if after := handlerPointers(h1); !reflect.DeepEqual(before, after) {
		t.Error("UpsertHandlers modified the input slice")
	}
}

// The recovery middleware asks for the current handler's name while unwinding a
// panic, after the deferred recover has already run. Panicking there escapes it.
func TestHandlerNameAfterStopExecution(t *testing.T) {
	ctx := &Context{}
	ctx.SetHandlers(Handlers{chainHandlerA})
	ctx.StopExecution()

	if got := ctx.HandlerName(); got != "" {
		t.Errorf("HandlerName() = %q, want an empty string", got)
	}

	if file, line := ctx.HandlerFileLine(); file != "" || line != 0 {
		t.Errorf("HandlerFileLine() = (%q, %d), want (%q, 0)", file, line, "")
	}
}

func TestHandlerNameWithoutHandlers(t *testing.T) {
	ctx := &Context{}

	if got := ctx.HandlerName(); got != "" {
		t.Errorf("HandlerName() = %q, want an empty string", got)
	}
}
