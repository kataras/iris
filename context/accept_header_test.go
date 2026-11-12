package context

import "testing"

func TestNegotiationMatch(t *testing.T) {
	tests := []struct {
		name       string
		in         []string
		priorities []string
		expected   string
	}{
		{
			name:       "exact match",
			in:         []string{"text/html", "application/json"},
			priorities: []string{"application/json"},
			expected:   "application/json",
		},
		{
			name:       "first accepted wins",
			in:         []string{"text/html", "application/json"},
			priorities: []string{"application/json", "text/html"},
			expected:   "text/html",
		},
		{
			name:       "no match",
			in:         []string{"text/html", "application/json"},
			priorities: []string{"text/xml"},
			expected:   "",
		},
		{
			// A server-side wildcard answers with the concrete type the client
			// asked for, per the contract documented on negotiationMatch.
			name:       "server wildcard yields the client's type",
			in:         []string{"text/html", "application/json"},
			priorities: []string{"*/*"},
			expected:   "text/html",
		},
		{
			name:       "server subtype wildcard yields the client's type",
			in:         []string{"text/html"},
			priorities: []string{"text/*"},
			expected:   "text/html",
		},
		{
			name:       "server subtype wildcard does not cross types",
			in:         []string{"application/json"},
			priorities: []string{"text/*"},
			expected:   "",
		},
		{
			name:       "client subtype wildcard",
			in:         []string{"text/html", "application/*"},
			priorities: []string{"application/json"},
			expected:   "application/json",
		},
		{
			name:       "no priorities",
			in:         []string{"application/json"},
			priorities: nil,
			expected:   "",
		},
		{
			name:       "no accept returns first priority",
			in:         nil,
			priorities: []string{"application/json", "text/html"},
			expected:   "application/json",
		},
		{
			// Regression: a client value that is a strict prefix-extension of a
			// registered mime used to index the priority string past its end and
			// panic the handler. Reported as a remote DoS: a single
			// "Accept: application/jsonX" request against any negotiated route.
			name:       "longer accept value does not panic and does not match",
			in:         []string{"application/jsonX"},
			priorities: []string{"application/json"},
			expected:   "",
		},
		{
			// The mirror case: a client value that is a strict prefix of a
			// registered mime must not match either, otherwise "text/h" would
			// select "text/html".
			name:       "shorter accept value does not match",
			in:         []string{"text/h"},
			priorities: []string{"text/html"},
			expected:   "",
		},
		{
			name:       "much longer accept value does not panic",
			in:         []string{"application/json-and-then-a-lot-more-characters"},
			priorities: []string{"application/json"},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := negotiationMatch(tt.in, tt.priorities); got != tt.expected {
				t.Fatalf("negotiationMatch(%q, %q) = %q, expected %q", tt.in, tt.priorities, got, tt.expected)
			}
		})
	}
}
