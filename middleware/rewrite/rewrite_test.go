package rewrite

import "testing"

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
