package cors

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		origin  string
		match   bool
	}{
		// "*" matches any origin.
		{"*", "https://example.com", true},
		{"*", "https://evil.com", true},

		// An exact pattern matches exactly.
		{"https://example.com", "https://example.com", true},
		{"https://example.com", "https://example.com.evil.com", false},
		{"https://example.com", "https://evil.com", false},
		{"https://example.com", "http://example.com", false},

		// "https://*.example.com" matches any subdomain but not the domain itself.
		{"https://*.example.com", "https://www.example.com", true},
		{"https://*.example.com", "https://api.staging.example.com", true},
		{"https://*.example.com", "https://example.com", false},
		{"https://*.example.com", "https://www.example.com.evil.com", false},
		{"https://*.example.com", "https://evil.com/?x=.example.com", false},

		// "https://www.example.*" matches any TLD for www.example - a single
		// label, not an arbitrary suffix.
		{"https://www.example.*", "https://www.example.com", true},
		{"https://www.example.*", "https://www.example.org", true},
		{"https://www.example.*", "https://www.example.co", true},
		// The bypass: an attacker registers evil.com and points
		// www.example.evil.com at their own server.
		{"https://www.example.*", "https://www.example.evil.com", false},
		{"https://www.example.*", "https://www.example.co.uk", false},
		{"https://www.example.*", "https://www.example.", false},
		{"https://www.example.*", "https://www.example.com/path", false},

		// "https://www.example*" is documented as a prefix match, so it stays
		// permissive on purpose - but it still may not cross into a path.
		{"https://www.example*", "https://www.example.com", true},
		{"https://www.example*", "https://www.exampleshop.com", true},
		{"https://www.example*", "https://www.example.com/path", false},
		{"https://www.example*", "https://evil.com", false},

		// A wildcard in the middle.
		{"https://*.example.*", "https://www.example.com", true},
		{"https://*.example.*", "https://www.example.evil.com", false},
	}

	for _, tt := range tests {
		if got := MatchPattern(tt.origin, tt.pattern); got != tt.match {
			t.Errorf("MatchPattern(origin=%q, pattern=%q) = %v, expected %v", tt.origin, tt.pattern, got, tt.match)
		}
	}
}
