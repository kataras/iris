package netutil

import (
	"testing"
)

func TestSplitURL(t *testing.T) {
	tests := []struct {
		inputURL       string
		expectedScheme string
		expectedHost   string
		expectedPort   int
	}{
		{"https://example.com:8080", "https", "example.com", 8080},
		{"https://example.com", "https", "example.com", 443},
		{"http://localhost:8080", "http", "localhost", 8080},
		{"localhost:443", "https", "localhost", 443},
		{"127.0.0.1:8080", "http", "localhost", 8080},
		{"0.0.0.0:80", "http", "localhost", 80},
		{"myserver.local", "http", "localhost", 80},
	}

	for _, test := range tests {
		scheme, host, port, err := SplitURL(test.inputURL)
		if err != nil {
			t.Errorf("SplitURL(%q) returned an error: %v", test.inputURL, err)
		}
		if scheme != test.expectedScheme || host != test.expectedHost || port != test.expectedPort {
			t.Errorf("SplitURL(%q) = %q, %q, %d; want %q, %q, %d", test.inputURL, scheme, host, port, test.expectedScheme, test.expectedHost, test.expectedPort)
		}
	}
}

func TestJoinURL(t *testing.T) {
	tests := []struct {
		scheme   string
		host     string
		port     int
		expected string
	}{
		{"https", "example.com", 8080, "https://example.com:8080"},
		{"https", "example.com", 443, "https://example.com"},
		{"http", "localhost", 8080, "http://localhost:8080"},
		{"https", "localhost", 443, "https://localhost"},
		{"http", "localhost", 80, "http://localhost"},
	}

	for _, test := range tests {
		result := JoinURL(test.scheme, test.host, test.port)
		if result != test.expected {
			t.Errorf("JoinURL(%q, %q, %d) = %q; want %q", test.scheme, test.host, test.port, result, test.expected)
		}
	}
}

// Additional tests can be added as needed.
