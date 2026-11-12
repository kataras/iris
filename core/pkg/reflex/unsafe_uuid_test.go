package reflex

import (
	"testing"
	"uuid"
)

// The rtype map in unsafe.go resolves types by their runtime type pointer and
// misses silently to UnsafeInvalidKind. If the uuid entry ever points at a
// different uuid type than the rest of the framework uses (as it would in a
// partial migration between uuid packages), this test goes red instead of the
// mismatch hiding behind an invalid kind at runtime.
func TestUnsafeKindOfUUID(t *testing.T) {
	if kind := UnsafeKindOf(uuid.UUID{}); kind != UnsafeUUIDKind {
		t.Fatalf("expected UnsafeKindOf(uuid.UUID{}) to be UnsafeUUIDKind, got %s", kind)
	}

	if got := UnsafeUUIDKind.String(); got != "uuid.UUID" {
		t.Fatalf("expected UnsafeUUIDKind.String() to be %q, got %q", "uuid.UUID", got)
	}
}

func TestUnsafeInterfaceOfUUIDRoundTrip(t *testing.T) {
	id := uuid.MustParse("a6ce7330-d584-4675-af9e-2c2d2c40de07")

	iface := UnsafeInterfaceOf(id)
	if kind := iface.UnsafeKind(); kind != UnsafeUUIDKind {
		t.Fatalf("expected the unsafe interface kind to be UnsafeUUIDKind, got %s", kind)
	}

	if got := iface.ValueUUID(); got != id {
		t.Fatalf("expected ValueUUID to round-trip the value, got %s want %s", got, id)
	}

	if got := *iface.ValueUUIDPtr(); got != id {
		t.Fatalf("expected ValueUUIDPtr to point at the value, got %s want %s", got, id)
	}
}
