package funcer

import (
	"testing"
)

// Dummy values for testing.
//
// The bodies are deliberately DISTINCT: since Go 1.27 the compiler may merge
// code-identical function literals to share a single code pointer, and these
// fixtures must remain three different functions under any toolchain.
var (
	names = NewNameResolver()
	pc    = NewProgramCounter[func()](names)

	fixtureSink int

	funcA = func() { fixtureSink = 1 }
	funcB = func() { fixtureSink = 2 }
	funcC = func() { fixtureSink = 3 }
)

// TestList_Append ensures that appending values works correctly.
func TestList_Append(t *testing.T) {
	list := NewList(pc, nil)
	list.Append(funcA)
	list.Append(funcB)

	if len(list.funcs) != 2 {
		t.Fatalf("Expected 2 values, got %d", len(list.funcs))
	}
}

// TestList_Prepend ensures that prepending values works correctly.
func TestList_Prepend(t *testing.T) {
	list := NewList(pc, nil)
	list.Prepend(funcA)
	list.Prepend(funcB)

	if len(list.funcs) != 2 {
		t.Fatalf("Expected 2 values, got %d", len(list.funcs))
	}
	if expected, got := list.funcs[0].Name, names.GetName(funcB); expected != got {
		t.Fatalf("Expected funcB to be first (%s vs %s)", expected, got)
	}
}

// TestList_Replace ensures that replacing values works correctly:
// only functions with a trusted (explicit) name can be replaced.
func TestList_Replace(t *testing.T) {
	list := NewList(pc, nil)
	list.Append(funcA, WithFuncName[func()]("A"))
	list.Replace(funcB) // funcB carries no trusted name: nothing is replaced.

	if list.funcs[0].Name != "A" || len(list.funcs) != 1 {
		t.Fatalf("funcB should not have replaced funcA")
	}

	// Replace by the trusted name.
	list.Upsert(funcB, WithFuncName[func()]("A"))
	if len(list.funcs) != 1 {
		t.Fatalf("Expected funcB to replace funcA in place, got %d entries", len(list.funcs))
	}
}

// TestList_PrependBefore ensures that prepending before a named func works correctly.
func TestList_PrependBefore(t *testing.T) {
	list := NewList(pc, nil, funcA)
	list.PrependBefore(funcB, list.FuncName(funcA))

	if len(list.funcs) != 2 || list.funcs[0].Name != names.GetName(funcB) {
		t.Fatalf("Expected funcB to be prepended before funcA")
	}
}

// TestList_AppendAfter ensures that appending after a named func works correctly.
func TestList_AppendAfter(t *testing.T) {
	list := NewList(pc, nil, funcA)
	list.AppendAfter(funcB, list.FuncName(funcA))

	if len(list.funcs) != 2 || list.funcs[1].Name != names.GetName(funcB) {
		t.Fatalf("Expected funcB to be appended after funcA")
	}
}

// TestList_Upsert ensures that upserting values works correctly for functions
// with trusted (explicit) names: replace in place, keep order.
func TestList_Upsert(t *testing.T) {
	list := NewList(pc, nil)
	list.Upsert(funcA, WithFuncName[func()]("A"))
	list.Upsert(funcB, WithFuncName[func()]("B"))
	if len(list.funcs) != 2 || list.funcs[1].Name != "B" {
		t.Fatalf("Expected funcB to be appended")
	}

	// Upsert an existing named func: it replaces the original in place.
	list.Upsert(funcA, WithFuncName[func()]("A"))
	if len(list.funcs) != 2 {
		t.Fatalf("Expected funcA to be replaced, not appended (got %d entries)", len(list.funcs))
	}

	list.Upsert(funcC, WithFuncName[func()]("C"))
	if len(list.funcs) != 3 || list.funcs[2].Name != "C" {
		t.Fatalf("Expected funcC to be appended")
	}

	// Ensure the order is maintained after upserts.
	if list.funcs[0].Name != "A" || list.funcs[1].Name != "B" || list.funcs[2].Name != "C" {
		t.Fatalf("Expected the order of values to be maintained")
	}
}

// TestList_UpsertAnonymousAppends pins the Go 1.27 identity contract:
// distinct anonymous closures always survive an Upsert as separate entries,
// even when their code is byte-identical (the compiler may give them a shared
// code pointer and therefore a shared symbol name — which is exactly why the
// derived name cannot serve as their identity).
func TestList_UpsertAnonymousAppends(t *testing.T) {
	var x, y int
	h1 := func() { x++ } // code-identical bodies with different closure data:
	h2 := func() { y++ } // the merge candidate shape.

	list := NewList(pc, nil)
	list.Upsert(h1)
	list.Upsert(h2)
	if got := list.Count(); got != 2 {
		t.Fatalf("Expected both anonymous closures to survive, got %d entries", got)
	}

	// Even re-upserting the very same anonymous closure appends: without a
	// trusted name there is no identity to replace. Name it to opt back in.
	list.Upsert(h1)
	if got := list.Count(); got != 3 {
		t.Fatalf("Expected the unnamed re-upsert to append, got %d entries", got)
	}
}

// TestList_Values ensures unique-value semantics: trusted names deduplicate,
// anonymous registrations never collapse.
func TestList_Values(t *testing.T) {
	list := NewList(pc, nil)
	list.Append(funcA, WithFuncName[func()]("same"))
	list.Append(funcB, WithFuncName[func()]("same"))
	if got := len(list.UniqueValues()); got != 1 {
		t.Fatalf("Expected 1 unique value for a shared trusted name, got %d", got)
	}

	anon := NewList(pc, nil, funcA, funcA) // two registrations of the same anonymous func.
	if got := len(anon.UniqueValues()); got != 2 {
		t.Fatalf("Expected 2 values (anonymous registrations are distinct), got %d", got)
	}
}

// TestConcatLists ensures that concatenating func lists works correctly.
func TestConcatLists(t *testing.T) {
	hc1 := NewList(pc, nil, funcA)
	hc2 := NewList(pc, nil, funcB)

	newList := ConcatLists(hc1, hc2)

	if len(newList.funcs) != 2 {
		t.Fatalf("Expected 2 values after concat, got %d", len(newList.funcs))
	}
}

// TestList_Clone ensures that cloning the func list works correctly,
// including the registration identity.
func TestList_Clone(t *testing.T) {
	list := NewList(pc, nil, funcA)
	clone := list.Clone()

	if len(clone.funcs) != 1 || names.GetName(clone.funcs[0].Value) != names.GetName(funcA) {
		t.Fatalf("Expected clone to have the same values as the original")
	}

	if clone.funcs[0].ID == 0 || clone.funcs[0].ID != list.funcs[0].ID {
		t.Fatalf("Expected the clone to keep the registration id (%d vs %d)", clone.funcs[0].ID, list.funcs[0].ID)
	}
}

// TestList_SubscribePrependBefore ensures that subscribing and prepending before a named func works correctly.
func TestList_SubscribePrependBefore(t *testing.T) {
	list := NewList(pc, nil)
	list.SubscribePrependBefore(funcA, names.GetName(funcB))
	list.Append(funcB)

	if expected, got := 2, len(list.funcs); expected != got {
		t.Fatalf("Expected values count %d but got: %d", expected, got)
	}

	if list.funcs[0].Name != names.GetName(funcA) {
		t.Fatalf("Expected funcA to be prepended before funcB")
	}

	if list.funcs[1].Name != names.GetName(funcB) {
		t.Fatalf("Expected funcB to be the last func")
	}
}

// TestList_SubscribeAppendAfter ensures that subscribing and appending after a named func works correctly.
func TestList_SubscribeAppendAfter(t *testing.T) {
	list := NewList(pc, nil)
	list.SubscribeAppendAfter(funcA, names.GetName(funcB))
	list.Append(funcB)

	if expected, got := 2, len(list.funcs); expected != got {
		t.Fatalf("Expected values count %d but got: %d", expected, got)
	}

	if list.funcs[1].Name != names.GetName(funcA) {
		t.Fatalf("Expected funcA to be after before funcB")
	}

	if list.funcs[0].Name != names.GetName(funcB) {
		t.Fatalf("Expected funcB to be the first func")
	}
}

// TestFunc_Equal ensures that the Equal method works correctly.
func TestFunc_Equal(t *testing.T) {
	infoA := pc.Resolve(funcA)
	infoB := pc.Resolve(funcB)

	if infoA.Equal(infoB) {
		t.Fatalf("Expected infoA and infoB to be not equal")
	}

	// An anonymous derived name is not an identity: even the same function,
	// resolved twice outside a list (no registration id), does not compare
	// equal — since Go 1.27 the symbol may be shared with another literal.
	if pc.Resolve(funcA).Equal(pc.Resolve(funcA)) {
		t.Fatalf("Expected two unregistered anonymous resolutions to be not equal")
	}

	// Trusted (explicit) names are identities.
	na, nb := pc.Resolve(funcA), pc.Resolve(funcB)
	WithFuncName[func()]("same")(na)
	WithFuncName[func()]("same")(nb)
	if !na.Equal(nb) {
		t.Fatalf("Expected two funcs with the same explicit name to be equal")
	}

	// A shared registration id is an identity.
	ia, ib := pc.Resolve(funcA), pc.Resolve(funcB)
	ia.ID, ib.ID = 42, 42
	if !ia.Equal(ib) {
		t.Fatalf("Expected two funcs with the same registration id to be equal")
	}
}
