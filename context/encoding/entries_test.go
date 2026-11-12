package encoding

import (
	"net/url"
	"testing"
	"unsafe"
)

func TestEntriesSetImmutable(t *testing.T) {
	entries := Entries[int]{}

	// Test saving a new key-value pair.
	entries.Set("key1", 100)
	if len(entries) != 1 || entries[0].Key != "key1" || entries[0].Value != 100 {
		t.Errorf("Save failed to add new key-value pair")
	}

	// Test saving an immutable key-value pair.
	entries.SetImmutable("key2", 200)
	if len(entries) != 2 || entries[1].Key != "key2" || entries[1].Value != 200 || !entries[1].Immutable {
		t.Errorf("Save failed to add immutable key-value pair")
	}

	// Test updating an existing mutable key-value pair.
	entries.Set("key1", 150)
	if entries[0].Value != 150 {
		t.Errorf("Save failed to update existing mutable key-value pair")
	}

	// Test that updating an immutable key-value pair does not work.
	entries.Set("key2", 250)
	if entries[1].Value != 200 {
		t.Errorf("Save updated an immutable key-value pair")
	}
}

func TestEntriesSet(t *testing.T) {
	entries := Entries[int]{}

	// Test setting a new key-value pair.
	entries.Set("key1", 100)
	if len(entries) != 1 || entries[0].Key != "key1" || entries[0].Value != 100 {
		t.Errorf("Set failed to add new key-value pair")
	}

	// Test updating an existing key-value pair.
	entries.Set("key1", 150)
	if entries[0].Value != 150 {
		t.Errorf("Set failed to update existing key-value pair")
	}
}

func TestEntriesAdd(t *testing.T) {
	entries := Entries[int]{}

	// Test adding a new key-value pair.
	entries.Add("key1", 100)
	if len(entries) != 1 || entries[0].Key != "key1" || entries[0].Value != 100 {
		t.Errorf("Add failed to add new key-value pair")
	}

	// Test adding another new key-value pair.
	entries.Add("key2", 200)
	if len(entries) != 2 || entries[1].Key != "key2" || entries[1].Value != 200 {
		t.Errorf("Add failed to add second new key-value pair")
	}
}

func TestEntriesReplace(t *testing.T) {
	entries := Entries[int]{}

	// Setup initial entries.
	entries.Add("key1", 100)
	entries.SetImmutable("key2", 200)

	// Test replacing a mutable key-value pair.
	replaced := entries.Replace("key1", 150)
	if !replaced || entries[0].Value != 150 {
		t.Errorf("Replace failed to replace mutable key-value pair")
	}

	// Test that replacing an immutable key-value pair does not work.
	replaced = entries.Replace("key2", 250)
	if replaced || entries[1].Value != 200 {
		t.Errorf("Replace incorrectly replaced an immutable key-value pair")
	}
}

// TestRemove tests the Remove method.
func TestEntriesRemove(t *testing.T) {
	entries := Entries[int]{}
	entries.Add("key1", 100)
	entries.Add("key2", 200)

	// Test removing an existing key.
	removed := entries.Remove("key1")
	if !removed || len(entries) != 1 || entries[0].Key != "key2" {
		t.Errorf("Remove failed to delete the existing key")
	}

	// Test removing a non-existing key.
	removed = entries.Remove("key3")
	if removed {
		t.Errorf("Remove incorrectly reported a non-existing key as removed")
	}
}

// TestGet tests the Get method.
func TestEntriesGet(t *testing.T) {
	entries := Entries[int]{}
	entries.Add("key1", 100)

	// Test getting an existing key.
	value := entries.Get("key1")
	if value != 100 {
		t.Errorf("Get failed to retrieve the value of an existing key")
	}

	// Test getting a non-existing key.
	value = entries.Get("key2")
	if value != 0 { // Assuming 0 is the empty value for int.
		t.Errorf("Get incorrectly retrieved a value for a non-existing key")
	}
}

func TestEntriesGetPointer(t *testing.T) {
	type myStruct struct {
		Field1 string
		field2 string
	}

	entries := Entries[any]{}
	entries.Add("key1", unsafe.Pointer(&myStruct{
		Field1: "value1",
		field2: "value2",
	}))

	// Test getting an existing key.
	ptr := entries.GetPointer("key1")
	if ptr == nil {
		t.Errorf("GetPointer failed to retrieve the value of an existing key")
	}

	value := (*myStruct)(ptr)
	if expected, got := "value1", value.Field1; expected != got {
		t.Errorf("GetPointer failed to retrieve the correct value for Field1, expected: %s, got: %s", expected, got)
	}

	if expected, got := "value2", value.field2; expected != got {
		t.Errorf("GetPointer failed to retrieve the correct value for field2, expected: %s, got: %s", expected, got)
	}
}

// TestLookup tests the Lookup method.
func TestEntriesLookup(t *testing.T) {
	entries := Entries[int]{}
	entries.Add("key1", 100)

	// Test looking up an existing key.
	value, found := entries.Lookup("key1")
	if !found || value != 100 {
		t.Errorf("Lookup failed to find an existing key")
	}

	// Test looking up a non-existing key.
	_, found = entries.Lookup("key2")
	if found {
		t.Errorf("Lookup incorrectly found a non-existing key")
	}
}

// TestGetEntry tests the GetEntry method.
func TestEntriesGetEntry(t *testing.T) {
	entries := Entries[int]{}
	entries.Add("key1", 100)

	// Test getting an existing key-value pair.
	entry, found := entries.GetEntry("key1")
	if !found || entry.Key != "key1" || entry.Value != 100 {
		t.Errorf("GetEntry failed to retrieve an existing key-value pair")
	}

	// Test getting a non-existing key-value pair.
	_, found = entries.GetEntry("key2")
	if found {
		t.Errorf("GetEntry incorrectly found a non-existing key-value pair")
	}
}

// TestToMap tests the ToMap method.
func TestEntriesToMap(t *testing.T) {
	entries := Entries[string]{}
	entries.Add("key1", "value1")
	entries.Add("key2", "value2/value3")

	// Test converting entries to url.Values.
	values := entries.ToMap()
	expectedValues := url.Values{
		"key1": []string{"value1"},
		"key2": []string{"value2", "value3"},
	}

	for key, expected := range expectedValues {
		if got, present := values[key]; !present || !equalStringSlice(got, expected) {
			t.Errorf("ToQuery - key: %s, expected: %v, got: %v", key, expected, got)
		}
	}
}

// TestMergeQuery tests the MergeQuery method.
func TestEntriesMergeQuery(t *testing.T) {
	entries := Entries[string]{}
	entries.Add("key1", "value1")

	extra := url.Values{
		"key2": []string{"extra1", "extra2"},
	}

	// Test merging entries with extra url.Values.
	mergedValues := entries.MergeQuery(extra)
	expectedMergedValues := url.Values{
		"key1": []string{"value1"},
		"key2": []string{"extra1", "extra2"},
	}

	for key, expected := range expectedMergedValues {
		if got, present := mergedValues[key]; !present || !equalStringSlice(got, expected) {
			t.Errorf("MergeQuery - key: %s, expected: %v, got: %v", key, expected, got)
		}
	}
}

// Helper function to compare two slices of strings.
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Include the Entry, Entries type definitions and other related methods here.
