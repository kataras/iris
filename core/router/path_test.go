package router

import (
	"testing"
)

func TestSplitSubdomainAndPath(t *testing.T) {
	tests := []struct {
		original  string
		subdomain string
		path      string
	}{
		{"admin./users/42", "admin.", "/users/42"},
		{"//api/users\\42", "", "/api/users/42"},
		{"admin./users/\\42", "admin.", "/users/42"},
		{"*./users/\\42", "*.", "/users/42"},
	}

	for i, tt := range tests {
		subdomain, path := splitSubdomainAndPath(tt.original)

		if expected, got := tt.subdomain, subdomain; expected != got {
			t.Fatalf("[%d] - expected subdomain '%s' but got '%s'", i, expected, got)
		}
		if expected, got := tt.path, path; expected != got {
			t.Fatalf("[%d] - expected path '%s' but got '%s'", i, expected, got)
		}
	}
}
