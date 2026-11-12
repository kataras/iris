package encoding

import (
	"net/url"
	"reflect"
	"testing"
)

// The trim helpers remove a suffix and nothing else. They used to hand the
// suffix to strings.TrimRight as a cutset over a lowercased value, so
// "math.html" came back as "ma" and "Index.html" as "index".
func TestEntryTrimKnownExtensionsRemoveOnlyTheSuffix(t *testing.T) {
	tests := []struct {
		name  string
		trim  func(*Entry[string]) string
		value string
		want  string
	}{
		{"TrimHTML", (*Entry[string]).TrimHTML, "math.html", "math"},
		{"TrimHTML", (*Entry[string]).TrimHTML, "length", "length"},
		{"TrimHTML", (*Entry[string]).TrimHTML, "Index.html", "Index"},
		{"TrimHTML", (*Entry[string]).TrimHTML, "REPORT.HTML", "REPORT"},
		{"TrimHTML", (*Entry[string]).TrimHTML, "a.html.html", "a.html"},
		{"TrimJSON", (*Entry[string]).TrimJSON, "jason.json", "jason"},
		{"TrimJSON", (*Entry[string]).TrimJSON, "Users.JSON", "Users"},
		{"TrimJSON", (*Entry[string]).TrimJSON, "son", "son"},
		{"TrimXML", (*Entry[string]).TrimXML, "llm.xml", "llm"},
		{"TrimXML", (*Entry[string]).TrimXML, "Feed.Xml", "Feed"},
		{"TrimXML", (*Entry[string]).TrimXML, "html", "html"},
	}

	for _, tt := range tests {
		e := &Entry[string]{Key: "file", Value: tt.value}
		if got := tt.trim(e); got != tt.want {
			t.Errorf("%s(%q): got %q, want %q", tt.name, tt.value, got, tt.want)
		}
	}
}

// The same through the store, which is how a handler reaches it.
func TestEntriesGetTrimHTMLRemovesOnlyTheSuffix(t *testing.T) {
	var params Entries[string]
	params.Set("page", "math.html")

	if got, want := params.GetTrimHTML("page"), "math"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// ToQuery has no map to give for an empty store, and MergeQuery used to write
// into that nil map.
func TestEntriesMergeQueryOnEmptyStore(t *testing.T) {
	var entries Entries[string]

	got := entries.MergeQuery(url.Values{"a": {"1"}})
	if want := (url.Values{"a": {"1"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A stored []string with spare capacity must not be appended to in place.
// Two merges over the same store would share it, and the second would
// overwrite what the first returned.
func TestEntriesMergeQueryDoesNotWriteIntoStoredSlices(t *testing.T) {
	stored := make([]string, 1, 4)
	stored[0] = "x"
	entries := Entries[any]{{Key: "k", Value: stored}}

	first := entries.MergeQuery(url.Values{"k": {"a"}})
	entries.MergeQuery(url.Values{"k": {"b"}})

	if want := []string{"x", "a"}; !reflect.DeepEqual(first["k"], want) {
		t.Fatalf("the first merge changed after the second: got %v, want %v", first["k"], want)
	}
}

// The probe shown in the documentation of As has no Type. Error must describe
// it instead of dereferencing the nil Type.
func TestErrEntryNotFoundErrorWithoutType(t *testing.T) {
	err := &ErrEntryNotFound{Key: "key", Kind: reflect.Int}

	if got, want := err.Error(), "not found: key as int"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
