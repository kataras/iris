package encoding

import "testing"

func TestEntryTrimFile(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"main.css", "main"},
		{"main.min.css", "main.min"}, // only the last ".ext" part is trimmed.
		{"main", "main"},
		{"a.css", "a.css"}, // too short base name, kept as it is (idx > 1 guard).
		{".css", ".css"},
		{"", ""},
	}

	for i, tt := range tests {
		e := Entry[string]{Key: "file", Value: tt.value}
		if got := e.TrimFile(); got != tt.expected {
			t.Fatalf("[%d] expected %q but got %q for %q", i, tt.expected, got, tt.value)
		}
	}
}

func TestEntryTrimFileInt(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"42.json", 42},
		{"42", 42},
		{"4.json", 0}, // too short base name, kept as it is (idx > 1 guard), Atoi fails.
		{"main.css", 0},
	}

	for i, tt := range tests {
		e := Entry[string]{Key: "file", Value: tt.value}
		if got := e.TrimFileInt(); got != tt.expected {
			t.Fatalf("[%d] expected %d but got %d for %q", i, tt.expected, got, tt.value)
		}
	}
}
