package validation_test

import (
	"testing"

	"github.com/kataras/iris/v14/rest/errors/validation"
)

// Numeric range messages must talk about values, not characters.
func TestInRangeMessagesAreNumeric(t *testing.T) {
	tests := []struct {
		name     string
		check    func() string
		expected string
	}{
		{
			name:     "below both bounds",
			check:    func() string { return validation.InRange(18, 130)(17) },
			expected: "must be between 18 and 130",
		},
		{
			name:     "above both bounds",
			check:    func() string { return validation.InRange(18, 130)(131) },
			expected: "must be between 18 and 130",
		},
		{
			name:     "only minimum",
			check:    func() string { return validation.InRange(10, 0)(9) },
			expected: "must be at least 10",
		},
		{
			name:     "only maximum",
			check:    func() string { return validation.InRange(0, 10)(11) },
			expected: "must be at most 10",
		},
		{
			name:     "exact value",
			check:    func() string { return validation.InRange(42, 42)(41) },
			expected: "must be 42",
		},
		{
			name:     "within range is valid",
			check:    func() string { return validation.InRange(18, 130)(30) },
			expected: "",
		},
		{
			name:     "no constraints",
			check:    func() string { return validation.InRange(0, 0)(-5) },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(); got != tt.expected {
				t.Fatalf("expected %q but got %q", tt.expected, got)
			}
		})
	}
}

// FormatRange renders the bounds of a numeric field as an interval.
func TestFormatRange(t *testing.T) {
	if got, expected := validation.FormatRange(18, 130), "[18, 130]"; got != expected {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}
