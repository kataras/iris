package encoding

import (
	"maps"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"unsafe"
)

// keysOf returns the keys in the order they are stored.
func keysOf[V any](entries Entries[V]) []string {
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
	}
	return keys
}

func sameValues(got, want map[string][]string) bool {
	return maps.EqualFunc(got, want, slices.Equal[[]string])
}

func TestEntriesStoreSetInsertsThenReplaces(t *testing.T) {
	var entries Entries[string]

	entries.Set("name", "iris")
	entries.Set("lang", "go")
	if got, want := keysOf(entries), []string{"name", "lang"}; !slices.Equal(got, want) {
		t.Fatalf("keys after two inserts: got %v, want %v", got, want)
	}

	// An existing key is replaced where it sits, it is not appended again.
	entries.Set("name", "kataras")
	if got, want := keysOf(entries), []string{"name", "lang"}; !slices.Equal(got, want) {
		t.Errorf("keys after replacing a value: got %v, want %v", got, want)
	}
	if got := entries.Get("name"); got != "kataras" {
		t.Errorf("Get(name): got %q, want %q", got, "kataras")
	}
	if got := entries.Get("lang"); got != "go" {
		t.Errorf("Get(lang): got %q, want %q", got, "go")
	}
}

// The doc comment of SetImmutable holds the whole rule: Set, SetImmutable and
// Replace cannot change the entry, while Remove, Pop, SetByIndex and
// SetEntryByIndex can.
func TestEntriesStoreImmutableEntry(t *testing.T) {
	newStore := func() Entries[int] {
		var entries Entries[int]
		entries.Set("open", 1)
		entries.SetImmutable("locked", 2)
		return entries
	}

	stillLocked := func(t *testing.T, entries Entries[int]) {
		t.Helper()
		entry, ok := entries.GetEntry("locked")
		if !ok || entry.Value != 2 || !entry.Immutable {
			t.Errorf("locked entry: got (%+v, %v), want the value 2, still immutable", entry, ok)
		}
		if got := entries.Len(); got != 2 {
			t.Errorf("Len: got %d, want 2", got)
		}
	}

	t.Run("Set leaves it untouched", func(t *testing.T) {
		entries := newStore()
		entries.Set("locked", 20)
		stillLocked(t, entries)
	})

	t.Run("SetImmutable leaves it untouched", func(t *testing.T) {
		entries := newStore()
		entries.SetImmutable("locked", 20)
		stillLocked(t, entries)
	})

	t.Run("Replace reports false and leaves it untouched", func(t *testing.T) {
		entries := newStore()
		if entries.Replace("locked", 20) {
			t.Error("Replace reported true for an immutable entry")
		}
		stillLocked(t, entries)
	})

	t.Run("SetImmutable locks an entry that Set created", func(t *testing.T) {
		entries := newStore()
		entries.SetImmutable("open", 10)

		entries.Set("open", 11)
		entries.SetImmutable("open", 12)
		if entries.Replace("open", 13) {
			t.Error("Replace reported true for an immutable entry")
		}

		if got := entries.Get("open"); got != 10 {
			t.Errorf("Get(open): got %d, want 10", got)
		}
		if got := entries.Len(); got != 2 {
			t.Errorf("Len: got %d, want 2", got)
		}
	})

	t.Run("Remove deletes it", func(t *testing.T) {
		entries := newStore()
		if !entries.Remove("locked") {
			t.Error("Remove reported false for an immutable entry")
		}
		if entries.Exists("locked") {
			t.Error("the immutable entry is still there after Remove")
		}
	})

	t.Run("Pop deletes it and returns its value", func(t *testing.T) {
		entries := newStore()
		if got := entries.Pop("locked"); got != 2 {
			t.Errorf("Pop: got %d, want 2", got)
		}
		if entries.Exists("locked") {
			t.Error("the immutable entry is still there after Pop")
		}
	})

	t.Run("SetByIndex overwrites it", func(t *testing.T) {
		entries := newStore()
		entries.SetByIndex(1, 20)
		if got := entries.Get("locked"); got != 20 {
			t.Errorf("Get(locked): got %d, want 20", got)
		}
	})

	t.Run("SetEntryByIndex overwrites it", func(t *testing.T) {
		entries := newStore()
		entries.SetEntryByIndex(1, Entry[int]{Key: "locked", Value: 20})
		if got := entries.Get("locked"); got != 20 {
			t.Errorf("Get(locked): got %d, want 20", got)
		}
	})

	t.Run("a removed key can be set again", func(t *testing.T) {
		entries := newStore()
		entries.Remove("locked")

		entries.Set("locked", 3)
		entries.Set("locked", 4)
		if got := entries.Get("locked"); got != 4 {
			t.Errorf("Get(locked): got %d, want 4", got)
		}
	})
}

func TestEntriesStoreAddAppends(t *testing.T) {
	var entries Entries[string]

	// Add does not look for the key, unlike Set.
	entries.Add("tag", "a")
	entries.Add("tag", "b")

	full := Entry[string]{Key: "list", Value: "x,y", Immutable: true, Sep: ","}
	entries.AddEntry(full)

	if got, want := keysOf(entries), []string{"tag", "tag", "list"}; !slices.Equal(got, want) {
		t.Fatalf("keys: got %v, want %v", got, want)
	}
	if got, _ := entries.LookupByIndex(0); got != "a" {
		t.Errorf("first value: got %q, want %q", got, "a")
	}
	if got, _ := entries.LookupByIndex(1); got != "b" {
		t.Errorf("second value: got %q, want %q", got, "b")
	}

	// AddEntry keeps every field of the entry it is given.
	if got, ok := entries.GetEntry("list"); !ok || got != full {
		t.Errorf("GetEntry(list): got (%+v, %v), want (%+v, true)", got, ok, full)
	}
}

// Reset and Remove keep the backing array, so the next write lands on a slot
// that still holds an old entry. Nothing of the old entry may show through.
func TestEntriesStoreReusedSlotStartsClean(t *testing.T) {
	old := Entry[string]{Key: "k", Value: "old", Immutable: true, Sep: ","}

	t.Run("Set after Reset", func(t *testing.T) {
		entries := Entries[string]{old}
		entries.Reset()

		entries.Set("k", "new")
		if got, want := entries[0], (Entry[string]{Key: "k", Value: "new"}); got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}

		entries.Set("k", "newer")
		if got := entries.Get("k"); got != "newer" {
			t.Errorf("the entry is not mutable: got %q, want %q", got, "newer")
		}
	})

	t.Run("SetImmutable after Reset", func(t *testing.T) {
		entries := Entries[string]{old}
		entries.Reset()

		entries.SetImmutable("other", "new")
		if got, want := entries[0], (Entry[string]{Key: "other", Value: "new", Immutable: true}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("Add after Reset", func(t *testing.T) {
		entries := Entries[string]{old}
		entries.Reset()

		entries.Add("other", "new")
		if got, want := entries[0], (Entry[string]{Key: "other", Value: "new"}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("Set after Remove", func(t *testing.T) {
		entries := Entries[string]{{Key: "first", Value: "1"}, old}
		entries.Remove("first")

		entries.Set("other", "new")
		if got, want := entries[1], (Entry[string]{Key: "other", Value: "new"}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
		if got, want := entries[0], old; got != want {
			t.Errorf("the entry that stayed: got %+v, want %+v", got, want)
		}
	})
}

func TestEntriesStoreSetByIndex(t *testing.T) {
	var entries Entries[int]
	entries.Set("a", 1)
	entries.Set("b", 2)

	entries.SetByIndex(1, 20)
	if got := entries.Get("b"); got != 20 {
		t.Errorf("Get(b) after SetByIndex: got %d, want 20", got)
	}
	if got := entries.Get("a"); got != 1 {
		t.Errorf("Get(a) after SetByIndex on another index: got %d, want 1", got)
	}

	// SetEntryByIndex swaps the whole pair, the key included.
	replacement := Entry[int]{Key: "c", Value: 30, Immutable: true}
	entries.SetEntryByIndex(0, replacement)
	if got, ok := entries.GetEntryByIndex(0); !ok || got != replacement {
		t.Errorf("GetEntryByIndex(0): got (%+v, %v), want (%+v, true)", got, ok, replacement)
	}
	if entries.Exists("a") {
		t.Error("the replaced key is still there")
	}
	if got, want := keysOf(entries), []string{"c", "b"}; !slices.Equal(got, want) {
		t.Errorf("keys: got %v, want %v", got, want)
	}
}

func TestEntriesStoreReplaceNeverInserts(t *testing.T) {
	var entries Entries[int]
	entries.Set("a", 1)

	if !entries.Replace("a", 10) {
		t.Error("Replace reported false for an existing key")
	}
	if got := entries.Get("a"); got != 10 {
		t.Errorf("Get(a): got %d, want 10", got)
	}

	if entries.Replace("missing", 5) {
		t.Error("Replace reported true for a missing key")
	}
	if entries.Exists("missing") || entries.Len() != 1 {
		t.Errorf("Replace added a missing key: %v", keysOf(entries))
	}
}

func TestEntriesStoreRemoveKeepsOrder(t *testing.T) {
	tests := []struct {
		remove string
		found  bool
		want   []string
	}{
		{"a", true, []string{"b", "c", "d"}},
		{"b", true, []string{"a", "c", "d"}},
		{"d", true, []string{"a", "b", "c"}},
		{"missing", false, []string{"a", "b", "c", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.remove, func(t *testing.T) {
			values := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}

			var entries Entries[int]
			for _, key := range []string{"a", "b", "c", "d"} {
				entries.Set(key, values[key])
			}

			if got := entries.Remove(tt.remove); got != tt.found {
				t.Errorf("Remove: got %v, want %v", got, tt.found)
			}
			if got := keysOf(entries); !slices.Equal(got, tt.want) {
				t.Errorf("keys: got %v, want %v", got, tt.want)
			}

			// Every key that stayed still has its own value.
			for _, key := range tt.want {
				if got := entries.Get(key); got != values[key] {
					t.Errorf("Get(%s): got %d, want %d", key, got, values[key])
				}
			}
		})
	}
}

func TestEntriesStoreReset(t *testing.T) {
	var entries Entries[int]
	entries.Set("a", 1)
	entries.SetImmutable("b", 2)

	entries.Reset()
	if got := entries.Len(); got != 0 {
		t.Fatalf("Len after Reset: got %d, want 0", got)
	}
	if entries.Exists("a") || entries.Exists("b") {
		t.Error("a key is still there after Reset")
	}
	if _, ok := entries.Lookup("b"); ok {
		t.Error("Lookup found a key after Reset")
	}
	if _, ok := entries.GetLastEntry(); ok {
		t.Error("GetLastEntry found an entry after Reset")
	}

	// The store is usable again.
	entries.Set("c", 3)
	if got, want := keysOf(entries), []string{"c"}; !slices.Equal(got, want) {
		t.Errorf("keys after Reset and Set: got %v, want %v", got, want)
	}

	var empty Entries[int]
	empty.Reset()
	if got := empty.Len(); got != 0 {
		t.Errorf("Len of a never used store after Reset: got %d, want 0", got)
	}
}

func TestEntriesStoreExists(t *testing.T) {
	var entries Entries[string]
	if entries.Exists("a") {
		t.Error("Exists reported true on an empty store")
	}

	// An empty value is still a key that exists.
	entries.Set("a", "")
	entries.Set("b", "x")

	for key, want := range map[string]bool{"a": true, "b": true, "c": false, "": false} {
		if got := entries.Exists(key); got != want {
			t.Errorf("Exists(%q): got %v, want %v", key, got, want)
		}
	}

	entries.Remove("a")
	if entries.Exists("a") {
		t.Error("Exists reported true for a removed key")
	}
}

// A stored zero value and a missing key read the same through Get, and only
// the second result of Lookup and GetEntry tells them apart.
func TestEntriesStoreLookupTellsZeroFromMissing(t *testing.T) {
	var entries Entries[int]
	entries.Set("zero", 0)
	entries.Set("seven", 7)

	if got, ok := entries.Lookup("zero"); !ok || got != 0 {
		t.Errorf("Lookup(zero): got (%d, %v), want (0, true)", got, ok)
	}
	if got, ok := entries.Lookup("seven"); !ok || got != 7 {
		t.Errorf("Lookup(seven): got (%d, %v), want (7, true)", got, ok)
	}
	if got, ok := entries.Lookup("missing"); ok || got != 0 {
		t.Errorf("Lookup(missing): got (%d, %v), want (0, false)", got, ok)
	}

	if got := entries.Get("seven"); got != 7 {
		t.Errorf("Get(seven): got %d, want 7", got)
	}
	if got := entries.Get("missing"); got != 0 {
		t.Errorf("Get(missing): got %d, want 0", got)
	}

	if got, ok := entries.GetEntry("zero"); !ok || got != (Entry[int]{Key: "zero"}) {
		t.Errorf("GetEntry(zero): got (%+v, %v), want the stored entry", got, ok)
	}
	if got, ok := entries.GetEntry("missing"); ok || got != (Entry[int]{}) {
		t.Errorf("GetEntry(missing): got (%+v, %v), want an empty entry and false", got, ok)
	}
}

func TestEntriesStoreReadByIndex(t *testing.T) {
	var entries Entries[int]
	entries.Set("a", 10)
	entries.SetImmutable("b", 20)
	entries.Set("c", 30)

	stored := []Entry[int]{
		{Key: "a", Value: 10},
		{Key: "b", Value: 20, Immutable: true},
		{Key: "c", Value: 30},
	}
	for index, want := range stored {
		if got, ok := entries.LookupByIndex(index); !ok || got != want.Value {
			t.Errorf("LookupByIndex(%d): got (%d, %v), want (%d, true)", index, got, ok, want.Value)
		}
		if got, ok := entries.GetEntryByIndex(index); !ok || got != want {
			t.Errorf("GetEntryByIndex(%d): got (%+v, %v), want (%+v, true)", index, got, ok, want)
		}
		if got, ok := entries.GetByIndex(index); !ok || got != strconv.Itoa(want.Value) {
			t.Errorf("GetByIndex(%d): got (%q, %v), want (%q, true)", index, got, ok, strconv.Itoa(want.Value))
		}
	}

	for _, index := range []int{-1, 3, 100} {
		if got, ok := entries.LookupByIndex(index); ok || got != 0 {
			t.Errorf("LookupByIndex(%d): got (%d, %v), want (0, false)", index, got, ok)
		}
		if got, ok := entries.GetEntryByIndex(index); ok || got != (Entry[int]{}) {
			t.Errorf("GetEntryByIndex(%d): got (%+v, %v), want an empty entry and false", index, got, ok)
		}
		if got, ok := entries.GetByIndex(index); ok || got != "" {
			t.Errorf("GetByIndex(%d): got (%q, %v), want (\"\", false)", index, got, ok)
		}
	}
}

func TestEntriesStoreGetLastEntry(t *testing.T) {
	var entries Entries[string]
	if _, ok := entries.GetLastEntry(); ok {
		t.Error("GetLastEntry found an entry on an empty store")
	}

	entries.Set("a", "1")
	entries.SetImmutable("b", "2")
	if got, ok := entries.GetLastEntry(); !ok || got != (Entry[string]{Key: "b", Value: "2", Immutable: true}) {
		t.Errorf("GetLastEntry: got (%+v, %v), want the b entry", got, ok)
	}

	entries.Remove("b")
	if got, ok := entries.GetLastEntry(); !ok || got != (Entry[string]{Key: "a", Value: "1"}) {
		t.Errorf("GetLastEntry after removing the last one: got (%+v, %v), want the a entry", got, ok)
	}
}

func TestEntriesStoreBind(t *testing.T) {
	entries := Entries[int]{{Key: "a", Value: 7}}

	var dst int
	if !entries.Bind("a", &dst) || dst != 7 {
		t.Errorf("Bind(a): destination is %d, want 7 and true", dst)
	}

	// A missing key leaves the destination alone.
	dst = 99
	if entries.Bind("missing", &dst) {
		t.Error("Bind reported true for a missing key")
	}
	if dst != 99 {
		t.Errorf("Bind(missing) changed the destination to %d", dst)
	}
}

func TestEntriesStorePop(t *testing.T) {
	var entries Entries[int]
	entries.Set("a", 1)
	entries.Set("b", 2)
	entries.Set("c", 3)

	if got := entries.Pop("b"); got != 2 {
		t.Errorf("Pop(b): got %d, want 2", got)
	}
	if got, want := keysOf(entries), []string{"a", "c"}; !slices.Equal(got, want) {
		t.Errorf("keys after Pop: got %v, want %v", got, want)
	}

	// Popped once, gone for good.
	if got := entries.Pop("b"); got != 0 {
		t.Errorf("second Pop(b): got %d, want 0", got)
	}
	if got := entries.Pop("missing"); got != 0 {
		t.Errorf("Pop(missing): got %d, want 0", got)
	}
	if got := entries.Len(); got != 2 {
		t.Errorf("Len after popping keys that are not there: got %d, want 2", got)
	}
}

func TestEntriesStorePointers(t *testing.T) {
	type payload struct{ n int }
	first, second := &payload{n: 1}, &payload{n: 2}

	var entries Entries[unsafe.Pointer]
	entries.Set("first", unsafe.Pointer(first))
	entries.Set("second", unsafe.Pointer(second))

	if got := (*payload)(entries.GetPointer("second")); got != second {
		t.Errorf("GetPointer(second): got %p, want %p", got, second)
	}
	if got := entries.GetPointer("missing"); got != nil {
		t.Errorf("GetPointer(missing): got %v, want nil", got)
	}

	if got := (*payload)(entries.PopPointer("first")); got != first {
		t.Errorf("PopPointer(first): got %p, want %p", got, first)
	}
	if got, want := keysOf(entries), []string{"second"}; !slices.Equal(got, want) {
		t.Errorf("keys after PopPointer: got %v, want %v", got, want)
	}
	if got := entries.PopPointer("first"); got != nil {
		t.Errorf("second PopPointer(first): got %v, want nil", got)
	}
	if got := entries.PopPointer("missing"); got != nil {
		t.Errorf("PopPointer(missing): got %v, want nil", got)
	}

	// The same through a store of any, next to values of other types.
	var values Entries[any]
	values.Set("name", "iris")
	values.Set("ptr", unsafe.Pointer(first))

	if got := (*payload)(values.PopPointer("ptr")); got != first {
		t.Errorf("PopPointer(ptr): got %p, want %p", got, first)
	}
	if got, want := keysOf(values), []string{"name"}; !slices.Equal(got, want) {
		t.Errorf("keys after PopPointer: got %v, want %v", got, want)
	}
}

func TestEntriesStoreForEach(t *testing.T) {
	var entries Entries[any]
	entries.Set("n", 42)
	entries.Set("s", "text")
	entries.Set("b", true)

	var (
		keys   []string
		values []any
	)
	entries.ForEach(func(key string, value any) {
		keys = append(keys, key)
		values = append(values, value)
	})
	if want := []string{"n", "s", "b"}; !slices.Equal(keys, want) {
		t.Errorf("ForEach keys: got %v, want %v", keys, want)
	}
	if want := []any{42, "text", true}; !slices.Equal(values, want) {
		t.Errorf("ForEach values: got %v, want %v", values, want)
	}

	var pairs []string
	entries.ForEachString(func(key, value string) {
		pairs = append(pairs, key+"="+value)
	})
	if want := []string{"n=42", "s=text", "b=true"}; !slices.Equal(pairs, want) {
		t.Errorf("ForEachString: got %v, want %v", pairs, want)
	}

	var empty Entries[any]
	empty.ForEach(func(string, any) {
		t.Error("ForEach called the function on an empty store")
	})
	empty.ForEachString(func(string, string) {
		t.Error("ForEachString called the function on an empty store")
	})
}

func TestEntriesStoreCloneIsIndependent(t *testing.T) {
	var original Entries[string]
	original.Set("a", "1")
	original.SetImmutable("b", "2")

	clone := original.Clone()
	if !slices.Equal(clone, original) {
		t.Fatalf("Clone: got %+v, want %+v", clone, original)
	}

	// Writes to the copy do not reach the original.
	clone.Set("a", "clone")
	clone.SetByIndex(1, "clone")
	clone.Add("c", "clone")
	want := Entries[string]{{Key: "a", Value: "1"}, {Key: "b", Value: "2", Immutable: true}}
	if !slices.Equal(original, want) {
		t.Errorf("the original after writing to the clone: got %+v, want %+v", original, want)
	}

	// And writes to the original do not reach the copy.
	original.Set("a", "original")
	original.Remove("b")
	want = Entries[string]{
		{Key: "a", Value: "clone"},
		{Key: "b", Value: "clone", Immutable: true},
		{Key: "c", Value: "clone"},
	}
	if !slices.Equal(clone, want) {
		t.Errorf("the clone after writing to the original: got %+v, want %+v", clone, want)
	}

	var empty Entries[string]
	if got := empty.Clone().Len(); got != 0 {
		t.Errorf("Len of the clone of an empty store: got %d, want 0", got)
	}
}

func TestEntriesStoreLen(t *testing.T) {
	var entries Entries[int]

	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"nothing yet", func() {}, 0},
		{"Set a new key", func() { entries.Set("a", 1) }, 1},
		{"Set the same key", func() { entries.Set("a", 2) }, 1},
		{"SetImmutable a new key", func() { entries.SetImmutable("b", 3) }, 2},
		{"Add", func() { entries.Add("c", 4) }, 3},
		{"Replace", func() { entries.Replace("c", 5) }, 3},
		{"Remove", func() { entries.Remove("a") }, 2},
		{"Remove a missing key", func() { entries.Remove("a") }, 2},
		{"Pop", func() { entries.Pop("b") }, 1},
		{"Reset", func() { entries.Reset() }, 0},
	}

	for _, step := range steps {
		step.do()
		if got := entries.Len(); got != step.want {
			t.Fatalf("Len after %q: got %d, want %d", step.name, got, step.want)
		}
	}
}

func TestEntriesStoreWithoutDefaults(t *testing.T) {
	var entries Entries[any]
	entries.Set("iris.locale", "en-US")
	entries.Set("user", "kataras")
	entries.Set("iris.view.layout", "main")
	entries.Set("irisette", 1) // looks alike, but it does not start with "iris.".

	filtered := entries.WithoutDefaults()
	if got, want := keysOf(filtered), []string{"user", "irisette"}; !slices.Equal(got, want) {
		t.Fatalf("keys: got %v, want %v", got, want)
	}
	if got := filtered.Get("user"); got != "kataras" {
		t.Errorf("Get(user): got %v, want kataras", got)
	}

	// It is a new store: the original keeps everything, and does not see writes.
	filtered.Set("user", "changed")
	filtered.Set("extra", true)
	if got := entries.Len(); got != 4 {
		t.Errorf("Len of the original: got %d, want 4", got)
	}
	if got := entries.Get("user"); got != "kataras" {
		t.Errorf("Get(user) on the original: got %v, want kataras", got)
	}

	var empty Entries[any]
	if got := empty.WithoutDefaults().Len(); got != 0 {
		t.Errorf("Len of an empty store without defaults: got %d, want 0", got)
	}
}

func TestEntriesStoreToQuery(t *testing.T) {
	entries := Entries[any]{
		{Key: "csv", Value: "a,b"},
		{Key: "path", Value: "a/b"}, // "/" is not the query separator.
		{Key: "list", Value: []string{"x", "y"}},
		{Key: "int", Value: 7},
		{Key: "bool", Value: true},
	}

	want := map[string][]string{
		"csv":  {"a", "b"},
		"path": {"a/b"},
		"list": {"x", "y"},
		"int":  {"7"},
		"bool": {"true"},
	}
	if got := entries.ToQuery(); !sameValues(got, want) {
		t.Errorf("ToQuery: got %v, want %v", got, want)
	}

	var empty Entries[any]
	if got := empty.ToQuery(); len(got) != 0 {
		t.Errorf("ToQuery of an empty store: got %v, want no values", got)
	}
}

func TestEntriesStoreToMapUsesEachSeparator(t *testing.T) {
	entries := Entries[any]{
		{Key: "default", Value: "a/b"}, // no Sep means "/".
		{Key: "comma", Value: "a,b", Sep: ","},
		{Key: "kept", Value: "a/b", Sep: ","},
		{Key: "list", Value: []string{"x", "y"}},
		{Key: "int", Value: 7},
		{Key: "float", Value: 1.5},
	}

	want := map[string][]string{
		"default": {"a", "b"},
		"comma":   {"a", "b"},
		"kept":    {"a/b"},
		"list":    {"x", "y"},
		"int":     {"7"},
		"float":   {"1.5"},
	}
	if got := entries.ToMap(); !sameValues(got, want) {
		t.Errorf("ToMap: got %v, want %v", got, want)
	}

	var empty Entries[any]
	if got := empty.ToMap(); len(got) != 0 {
		t.Errorf("ToMap of an empty store: got %v, want no values", got)
	}
}

func TestEntriesStoreMergeQueryAppendsToSharedKeys(t *testing.T) {
	entries := Entries[string]{
		{Key: "tag", Value: "a,b"},
		{Key: "page", Value: "1"},
	}

	got := entries.MergeQuery(url.Values{"tag": {"c"}, "sort": {"asc"}})
	want := map[string][]string{
		"tag":  {"a", "b", "c"},
		"page": {"1"},
		"sort": {"asc"},
	}
	if !sameValues(got, want) {
		t.Errorf("MergeQuery: got %v, want %v", got, want)
	}

	// The store itself is not changed by a merge.
	if got := entries.Get("tag"); got != "a,b" || entries.Len() != 2 {
		t.Errorf("the store after MergeQuery: %+v", entries)
	}

	// With nothing to merge the result is the plain query.
	for _, extra := range []url.Values{nil, {}} {
		if got := entries.MergeQuery(extra); !sameValues(got, entries.ToQuery()) {
			t.Errorf("MergeQuery(%v): got %v, want %v", extra, got, entries.ToQuery())
		}
	}
}
