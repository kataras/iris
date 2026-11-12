package rewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOptionsFileExtension(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		filename string
		contents string
	}{
		{"redirects.yml", "RedirectMatch:\n  - 301 /seo/(.*) /$1\n"},
		{"redirects.yaml", "RedirectMatch:\n  - 301 /seo/(.*) /$1\n"},
		{"redirects.json", `{"redirectMatch": ["301 /seo/(.*) /$1"]}`},
		{"redirects", "RedirectMatch:\n  - 301 /seo/(.*) /$1\n"}, // no extension defaults to yaml.
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if !strings.Contains(tt.filename, ".") && strings.Contains(dir, ".") {
				// the extension is cut from the full path,
				// a dotted temp dir would hide the no-extension case.
				t.Skipf("temp dir contains a dot: %s", dir)
			}

			filename := filepath.Join(dir, tt.filename)
			if err := os.WriteFile(filename, []byte(tt.contents), 0600); err != nil {
				t.Fatal(err)
			}

			opts := LoadOptions(filename)
			if expected, got := 1, len(opts.RedirectMatch); expected != got {
				t.Fatalf("expected %d redirect match lines but got %d", expected, got)
			}

			if expected, got := "301 /seo/(.*) /$1", opts.RedirectMatch[0]; expected != got {
				t.Fatalf("expected %q but got %q", expected, got)
			}
		})
	}
}

func TestRedirectMatch(t *testing.T) {
	tests := []struct {
		line     string
		parseErr string
		inputs   map[string]string // input, expected. Order should not matter.
	}{
		{
			"301 /seo/(.*) /$1",
			"",
			map[string]string{
				"/seo/path": "/path",
			},
		},
		{
			"301 /old(.*) /deprecated$1",
			"",
			map[string]string{
				"/old":            "/deprecated",
				"/old/any":        "/deprecated/any",
				"/old/thing/here": "/deprecated/thing/here",
			},
		},
		{
			"301 /old(.*) /",
			"",
			map[string]string{
				"/oldblabla":      "/",
				"/old/any":        "/",
				"/old/thing/here": "/",
			},
		},
		{
			"301 /old/(.*) /deprecated/$1",
			"",
			map[string]string{
				"/old/":           "/deprecated/",
				"/old/any":        "/deprecated/any",
				"/old/thing/here": "/deprecated/thing/here",
			},
		},
		{
			"3d /seo/(.*) /$1",
			"redirect match: status code digits: 3d [1:d]",
			nil,
		},
		{
			"301 /$1",
			"redirect match: invalid line: 301 /$1",
			nil,
		},
		{
			"301 /* /$1",
			"redirect match: loop detected: pattern: /* vs target: /$1",
			nil,
		},
		{
			"301 /* /",
			"redirect match: loop detected: pattern: /* vs target: /",
			nil,
		},
	}

	for i, tt := range tests {
		r, err := parseRedirectMatchLine(tt.line)
		if err != nil {
			if tt.parseErr == "" {
				t.Fatalf("[%d] unexpected parse error: %v", i, err)
			}

			errStr := err.Error()
			if tt.parseErr != err.Error() {
				t.Fatalf("[%d] a parse error was expected but it differs: expected: %s but got: %s", i, tt.parseErr, errStr)
			}
		} else if tt.parseErr != "" {
			t.Fatalf("[%d] expected an error of: %s but got nil", i, tt.parseErr)
		}

		for input, expected := range tt.inputs {
			got, _ := r.matchAndReplace(input)
			if expected != got {
				t.Fatalf(`[%d:%s] expected: "%s" but got: "%s"`, i, input, expected, got)
			}
		}
	}
}
