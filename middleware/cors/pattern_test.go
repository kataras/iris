package cors

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMergeOriginPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "exact domain and wildcard subdomain, both kept",
			input: []string{"https://example.com", "https://*.example.com"},
			want:  []string{"https://example.com", "https://*.example.com"},
		},
		{
			name:  "multiple wildcards, widest wins",
			input: []string{"https://foo.example.com", "https://*.example.com", "*"},
			want:  []string{"*"},
		},
		{
			name:  "multiple subdomain wildcards, only widest kept",
			input: []string{"https://foo.example.com", "https://*.example.com", "https://*.foo.example.com"},
			want:  []string{"https://*.example.com"},
		},
		{
			name:  "no wildcards, all unique",
			input: []string{"https://a.com", "https://b.com", "https://c.com"},
			want:  []string{"https://a.com", "https://b.com", "https://c.com"},
		},
		{
			name:  "prefix wildcard wins",
			input: []string{"https://www.example.com", "https://www.example*"},
			want:  []string{"https://www.example*"},
		},
		{
			name:  "tld wildcard wins",
			input: []string{"https://www.example.com", "https://www.example.*"},
			want:  []string{"https://www.example.*"},
		},
		{
			name:  "multiple covered by one",
			input: []string{"https://foo.example.com", "https://bar.example.com", "https://*.example.com"},
			want:  []string{"https://*.example.com"},
		},
		{
			name:  "multiple wildcards, only widest kept",
			input: []string{"https://foo.example.com", "https://*.example.com", "https://*.foo.example.com", "*"},
			want:  []string{"*"},
		},
		{
			name:  "duplicates removed",
			input: []string{"https://a.com", "https://a.com", "https://*.a.com"},
			want:  []string{"https://a.com", "https://*.a.com"},
		},
		{
			name:  "wildcard in the middle",
			input: []string{"https://foo.example.com:8080", "https://*.example.com:*"},
			want:  []string{"https://*.example.com:*"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("[%d] %s", i, tt.name), func(t *testing.T) {
			got := MergeOriginPatterns(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeOriginPatterns(%v) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}
