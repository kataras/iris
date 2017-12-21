package router

import (
	"testing"
)

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/v2/stores/{id:string format(uuid)} /v3",
			[]string{"/v2/stores/{id:string format(uuid)}", "/v3"}},
		{"/user/{id:int} /admin/{id:int}",
			[]string{"/user/{id:int}", "/admin/{id:int}"}},
		{"/user /admin",
			[]string{"/user", "/admin"}},
		{"/single_no_params",
			[]string{"/single_no_params"}},
		{"/single/{id:int}",
			[]string{"/single/{id:int}"}},
	}

	equalSlice := func(s1 []string, s2 []string) bool {
		if len(s1) != len(s2) {
			return false
		}

		for i := range s1 {
			if s2[i] != s1[i] {
				return false
			}
		}

		return true
	}

	for i, tt := range tests {
		paths := splitPath(tt.path)
		if expected, got := tt.expected, paths; !equalSlice(expected, got) {
			t.Fatalf("[%d] - expected paths '%#v' but got '%#v'", i, expected, got)
		}
	}
}
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
