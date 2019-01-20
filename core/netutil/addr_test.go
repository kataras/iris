package netutil

import (
	"testing"
)

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host  string
		valid bool
	}{
		{"subdomain.127.0.0.1:8080", true},
		{"subdomain.127.0.0.1", true},
		{"subdomain.localhost:8080", true},
		{"subdomain.localhost", true},
		{"subdomain.127.0000.0000.1:8080", true},
		{"subdomain.127.0000.0000.1", true},
		{"subdomain.127.255.255.254:8080", true},
		{"subdomain.127.255.255.254", true},

		{"subdomain.0000:0:0000::01.1:8080", false},
		{"subdomain.0000:0:0000::01", false},
		{"subdomain.0000:0:0000::01.1:8080", false},
		{"subdomain.0000:0:0000::01", false},
		{"subdomain.0000:0000:0000:0000:0000:0000:0000:0001:8080", true},
		{"subdomain.0000:0000:0000:0000:0000:0000:0000:0001", false},

		{"subdomain.example:8080", false},
		{"subdomain.example", false},
		{"subdomain.example.com:8080", false},
		{"subdomain.example.com", false},
		{"subdomain.com", false},
		{"subdomain", false},
		{".subdomain", false},
		{"127.0.0.1.com", false},
	}

	for i, tt := range tests {
		if expected, got := tt.valid, IsLoopbackHost(tt.host); expected != got {
			t.Fatalf("[%d] expected %t but got %t for %s", i, expected, got, tt.host)
		}
	}
}
