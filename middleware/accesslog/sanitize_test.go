package accesslog

import (
	"strings"
	"testing"
)

// ctx.Path() is percent-DECODED, so a request can put a real newline into the
// value that gets written to the log. Without sanitizing, that newline ends the
// log line and everything after it is parsed as a fresh, forged entry.
func TestSanitizeLogValue(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		notIn string
	}{
		{
			name: "plain value is untouched",
			in:   "/api/users/42",
			want: "/api/users/42",
		},
		{
			name:  "forged log line",
			in:    "/a\n2026-08-01 12:00:00|1ms|200|GET|/admin|127.0.0.1",
			want:  "/a 2026-08-01 12:00:00|1ms|200|GET|/admin|127.0.0.1",
			notIn: "\n",
		},
		{
			name:  "carriage return",
			in:    "/a\r\nGET /admin",
			want:  "/a  GET /admin",
			notIn: "\r",
		},
		{
			name:  "tab becomes a space",
			in:    "/a\tb",
			want:  "/a b",
			notIn: "\t",
		},
		{
			name:  "ANSI escape is stripped",
			in:    "/a\x1b[2K\x1b[1Gmalicious",
			notIn: "\x1b",
		},
		{
			name:  "other control characters are stripped",
			in:    "/a\x00\x07\x7fb",
			want:  "/ab",
			notIn: "\x00",
		},
		{
			name: "unicode survives",
			in:   "/καλημέρα",
			want: "/καλημέρα",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeLogValue(tt.in)

			if tt.want != "" && got != tt.want {
				t.Errorf("sanitizeLogValue(%q) = %q, expected %q", tt.in, got, tt.want)
			}
			if tt.notIn != "" && strings.Contains(got, tt.notIn) {
				t.Errorf("sanitizeLogValue(%q) = %q, still contains %q", tt.in, got, tt.notIn)
			}
			if strings.ContainsAny(got, "\n\r") {
				t.Errorf("sanitizeLogValue(%q) = %q, still contains a line break", tt.in, got)
			}
		})
	}
}

// The CSV formatter exists so the log can be opened in a spreadsheet, which is
// exactly what makes an unescaped leading "=" dangerous.
func TestEscapeCSVFormula(t *testing.T) {
	dangerous := []string{
		`=cmd|'/c calc'!A1`,
		`+1+1`,
		`-1+1`,
		`@SUM(1,2)`,
		"\tvalue",
		"\rvalue",
	}

	for _, s := range dangerous {
		got := escapeCSVFormula(s)
		if got == s {
			t.Errorf("escapeCSVFormula(%q) left it unescaped", s)
		}
		if !strings.HasPrefix(got, "'") {
			t.Errorf("escapeCSVFormula(%q) = %q, expected a leading single quote", s, got)
		}
	}

	safe := []string{"", "/api/users", "200", "1.5ms", "makis@example.com"}
	for _, s := range safe {
		if got := escapeCSVFormula(s); got != s {
			t.Errorf("escapeCSVFormula(%q) = %q, expected it unchanged", s, got)
		}
	}
}
