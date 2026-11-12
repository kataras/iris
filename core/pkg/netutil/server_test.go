package netutil

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestIsTLSAndResolveSchemeFromServer(t *testing.T) {
	tests := []struct {
		name       string
		srv        *http.Server
		wantTLS    bool
		wantScheme string
	}{
		{"NoTLSConfig", &http.Server{}, false, "http"},
		{"EmptyTLSConfig", &http.Server{TLSConfig: &tls.Config{}}, false, "http"},
		{"WithCertificates", &http.Server{TLSConfig: &tls.Config{Certificates: make([]tls.Certificate, 1)}}, true, "https"},
		{"WithGetCertificate", &http.Server{TLSConfig: &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return nil, nil },
		}}, true, "https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTLS(tt.srv); got != tt.wantTLS {
				t.Errorf("IsTLS() = %v, want %v", got, tt.wantTLS)
			}
			if got := ResolveSchemeFromServer(tt.srv); got != tt.wantScheme {
				t.Errorf("ResolveSchemeFromServer() = %q, want %q", got, tt.wantScheme)
			}
		})
	}
}
