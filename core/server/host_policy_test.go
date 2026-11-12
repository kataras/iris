package server

import (
	"context"
	"testing"
)

func TestNewHostPolicy(t *testing.T) {
	patterns := []string{"example.com", "www.example.com", "*.example.org"}
	policy := newHostPolicy(patterns)

	tests := []struct {
		host    string
		wantErr bool
	}{
		{"example.com", false},
		{"www.example.com", false},
		{"sub.example.com", true},
		{"www.example.org", false},
		{"sub.example.org", false},
		{"another.example.org", false},
		{"example.org", true}, // root domain not registered.
		{"www.anotherdomain.com", true},
	}

	for i, tt := range tests {
		err := policy(context.Background(), tt.host)
		if (err != nil) != tt.wantErr {
			t.Errorf("[%d] newHostPolicy() error = %v, wantErr %v for host %v", i, err, tt.wantErr, tt.host)
		}
	}
}
