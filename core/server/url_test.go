package server

import (
	"testing"
)

// TestIsRootDomain checks the IsRootDomain function for various cases.
func TestIsRootDomain(t *testing.T) {
	tests := []struct {
		hostOrIP string
		want     bool
	}{
		{"example.com", true},
		{"sub.example.com", false},
		{"192.168.1.1", true},
		{"localhost:8080", true},
		{"invalid.", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostOrIP, func(t *testing.T) {
			if got := IsRootDomain(tt.hostOrIP); got != tt.want {
				t.Errorf("IsRootDomain(%q) = %v, want %v", tt.hostOrIP, got, tt.want)
			}
		})
	}
}

// TestIsLoopback checks the IsLoopback function for various cases.
func TestIsLoopback(t *testing.T) {
	tests := []struct {
		hostOrIP string
		want     bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"example.com", false}, // true if we add it to the system's hosts list.
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostOrIP, func(t *testing.T) {
			if got := IsLoopback(tt.hostOrIP); got != tt.want {
				t.Errorf("IsLoopback(%q) = %v, want %v", tt.hostOrIP, got, tt.want)
			}
		})
	}
}

// TestGetRootDomain checks the GetRootDomain function for various cases.
func TestGetRootDomain(t *testing.T) {
	tests := []struct {
		hostOrIP string
		want     string
	}{
		{"example.com", "example.com"},
		{"sub.example.com", "example.com"},
		{"localhost", "localhost"},
		{"192.168.1.1", "192.168.1.1"},
		{"invalid.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.hostOrIP, func(t *testing.T) {
			if got := GetRootDomain(tt.hostOrIP); got != tt.want {
				t.Errorf("GetRootDomain(%q) = %q, want %q", tt.hostOrIP, got, tt.want)
			}
		})
	}
}

// TestGetSubdomain checks the GetSubdomain function for various cases.
func TestGetSubdomain(t *testing.T) {
	tests := []struct {
		hostOrIP string
		want     string
	}{
		{"example.com", ""},
		{"sub.example.com", "sub"},
		{"localhost", ""},
		{"192.168.1.1", ""},
		{"invalid.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.hostOrIP, func(t *testing.T) {
			if got := GetSubdomain(tt.hostOrIP); got != tt.want {
				t.Errorf("GetSubdomain(%q) = %q, want %q", tt.hostOrIP, got, tt.want)
			}
		})
	}
}

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
		{"mylocal", "", "", 0},
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
