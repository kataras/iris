// white-box testing
package memstore

import (
	"testing"
)

func TestStoreSetAndGet(t *testing.T) {
	var p Store

	entry, inserted := p.Set("key", "value")
	if !inserted {
		t.Fatalf("expected the first Set of a key to report an insert")
	}
	if entry.Key != "key" {
		t.Fatalf("expected entry key to be %q but got %q", "key", entry.Key)
	}
	if got := p.Get("key"); got != "value" {
		t.Fatalf("expected %q but got %v", "value", got)
	}

	// setting an existing key updates it and reports that it was not inserted
	_, inserted = p.Set("key", "other")
	if inserted {
		t.Fatalf("expected an update of an existing key to report no insert")
	}
	if got := p.Get("key"); got != "other" {
		t.Fatalf("expected %q but got %v", "other", got)
	}
	if p.Len() != 1 {
		t.Fatalf("expected the store to still hold one entry but it holds %d", p.Len())
	}
}

func TestStoreGetMissing(t *testing.T) {
	var p Store

	if got := p.Get("missing"); got != nil {
		t.Fatalf("expected nil for a missing key but got %v", got)
	}
	if got := p.GetDefault("missing", "def"); got != "def" {
		t.Fatalf("expected the default value but got %v", got)
	}
	if _, ok := p.GetEntry("missing"); ok {
		t.Fatalf("expected GetEntry to report a missing key")
	}
	if p.Exists("missing") {
		t.Fatalf("expected Exists to report false for a missing key")
	}
}

// a key stored with a nil value is treated as absent by the value getters
func TestStoreNilValueFallsBackToDefault(t *testing.T) {
	var p Store
	p.Set("key", nil)

	if !p.Exists("key") {
		t.Fatalf("expected the key to exist even though its value is nil")
	}
	if got := p.GetDefault("key", "def"); got != "def" {
		t.Fatalf("expected the default value but got %v", got)
	}
	if got := p.Get("key"); got != nil {
		t.Fatalf("expected nil but got %v", got)
	}
}

func TestStoreGetEntryAt(t *testing.T) {
	var p Store
	p.Set("first", 1)
	p.Set("second", 2)

	entry, ok := p.GetEntryAt(0)
	if !ok || entry.Key != "first" {
		t.Fatalf("expected the entry at index 0 to be %q but got %q (ok=%v)", "first", entry.Key, ok)
	}

	entry, ok = p.GetEntryAt(1)
	if !ok || entry.Key != "second" {
		t.Fatalf("expected the entry at index 1 to be %q but got %q (ok=%v)", "second", entry.Key, ok)
	}

	if _, ok = p.GetEntryAt(2); ok {
		t.Fatalf("expected an out of range index to report not found")
	}
}

func TestStoreRemove(t *testing.T) {
	var p Store
	p.Set("a", 1)
	p.Set("b", 2)
	p.Set("c", 3)

	if !p.Remove("b") {
		t.Fatalf("expected Remove to report that an entry was removed")
	}
	if p.Len() != 2 {
		t.Fatalf("expected two remaining entries but got %d", p.Len())
	}
	if p.Exists("b") {
		t.Fatalf("expected the removed key to be gone")
	}
	// the surrounding entries survive the removal
	if p.Get("a") != 1 || p.Get("c") != 3 {
		t.Fatalf("expected the other entries to be untouched")
	}

	if p.Remove("missing") {
		t.Fatalf("expected Remove of a missing key to report false")
	}
}

func TestStoreReset(t *testing.T) {
	var p Store
	p.Set("a", 1)
	p.Set("b", 2)

	p.Reset()

	if p.Len() != 0 {
		t.Fatalf("expected an empty store after Reset but it holds %d entries", p.Len())
	}
	if p.Exists("a") {
		t.Fatalf("expected entries to be gone after Reset")
	}
}

func TestStoreVisit(t *testing.T) {
	var p Store
	p.Set("a", 1)
	p.Set("b", 2)

	visited := make(map[string]any)
	p.Visit(func(key string, value any) {
		visited[key] = value
	})

	if len(visited) != 2 {
		t.Fatalf("expected to visit two entries but visited %d", len(visited))
	}
	if visited["a"] != 1 || visited["b"] != 2 {
		t.Fatalf("expected the visitor to receive every key and value, got %v", visited)
	}
}

func TestStoreGetOrSet(t *testing.T) {
	var p Store

	calls := 0
	setFunc := func() any {
		calls++
		return "computed"
	}

	if got := p.GetOrSet("key", setFunc); got != "computed" {
		t.Fatalf("expected the computed value but got %v", got)
	}
	if calls != 1 {
		t.Fatalf("expected the set function to be called once but it was called %d times", calls)
	}

	// the value is now stored, so the function must not be called again
	if got := p.GetOrSet("key", setFunc); got != "computed" {
		t.Fatalf("expected the stored value but got %v", got)
	}
	if calls != 1 {
		t.Fatalf("expected the set function not to be called again, call count is %d", calls)
	}
}

func TestStoreImmutable(t *testing.T) {
	var p Store

	p.SetImmutable("key", "original")
	if got := p.Get("key"); got != "original" {
		t.Fatalf("expected %q but got %v", "original", got)
	}

	// a plain Set must not overwrite an immutable entry
	p.Set("key", "changed")
	if got := p.Get("key"); got != "original" {
		t.Fatalf("expected an immutable entry to reject Set, but it became %v", got)
	}

	// SetImmutable may still update it
	p.SetImmutable("key", "updated")
	if got := p.Get("key"); got != "updated" {
		t.Fatalf("expected SetImmutable to update an immutable entry, got %v", got)
	}
}

func TestStoreGetStringDefault(t *testing.T) {
	var p Store
	p.Set("str", "value")
	p.Set("num", 42)

	if got := p.GetStringDefault("str", "def"); got != "value" {
		t.Fatalf("expected %q but got %q", "value", got)
	}
	// non-string values are formatted
	if got := p.GetStringDefault("num", "def"); got != "42" {
		t.Fatalf("expected %q but got %q", "42", got)
	}
	if got := p.GetStringDefault("missing", "def"); got != "def" {
		t.Fatalf("expected the default but got %q", got)
	}
}

func TestStoreGetIntDefault(t *testing.T) {
	var p Store
	p.Set("num", 42)
	p.Set("str", "7")
	p.Set("bad", "not a number")

	if got := p.GetIntDefault("num", -1); got != 42 {
		t.Fatalf("expected 42 but got %d", got)
	}
	// numeric strings are parsed
	if got := p.GetIntDefault("str", -1); got != 7 {
		t.Fatalf("expected 7 but got %d", got)
	}
	if got := p.GetIntDefault("bad", -1); got != -1 {
		t.Fatalf("expected the default for an unparsable value but got %d", got)
	}
	if got := p.GetIntDefault("missing", -1); got != -1 {
		t.Fatalf("expected the default for a missing key but got %d", got)
	}
}

func TestStoreGetBoolDefault(t *testing.T) {
	var p Store
	p.Set("yes", true)
	p.Set("str", "true")
	p.Set("one", 1)
	p.Set("zero", 0)

	if !p.GetBoolDefault("yes", false) {
		t.Fatalf("expected true for a stored bool")
	}
	if !p.GetBoolDefault("str", false) {
		t.Fatalf("expected true for the string %q", "true")
	}
	if !p.GetBoolDefault("one", false) {
		t.Fatalf("expected int 1 to be true")
	}
	if p.GetBoolDefault("zero", true) {
		t.Fatalf("expected int 0 to be false")
	}
	if !p.GetBoolDefault("missing", true) {
		t.Fatalf("expected the default for a missing key")
	}
}

func TestStoreLen(t *testing.T) {
	var p Store

	if p.Len() != 0 {
		t.Fatalf("expected an empty store to have length 0")
	}

	p.Set("a", 1)
	p.Set("b", 2)
	if p.Len() != 2 {
		t.Fatalf("expected length 2 but got %d", p.Len())
	}

	// updating an existing key does not grow the store
	p.Set("a", 3)
	if p.Len() != 2 {
		t.Fatalf("expected length to stay 2 after an update but got %d", p.Len())
	}
}
