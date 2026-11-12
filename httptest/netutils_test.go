package httptest

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestNewLocalTLSListener tests that the listener serves a TLS connection
// which a client can verify against the generated LocalhostCert.
func TestNewLocalTLSListener(t *testing.T) {
	ln := NewLocalTLSListener(nil)

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(LocalhostCert()) {
		t.Fatal("failed to parse LocalhostCert PEM")
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
	}

	resp, err := client.Get("https://" + ln.Addr().String())
	if err != nil {
		t.Fatalf("TLS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.TLS == nil {
		t.Fatal("expected a TLS connection state")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("expected body %q but got %q", "ok", string(body))
	}
}

// TestLocalhostCertProperties tests the shape of the generated certificate:
// ECDSA P-256, SANs for localhost, 127.0.0.1 and ::1, and a validity window
// that starts in the past and lasts long enough for any test run.
func TestLocalhostCertProperties(t *testing.T) {
	block, _ := pem.Decode(LocalhostCert())
	if block == nil {
		t.Fatal("failed to decode LocalhostCert PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	if cert.PublicKeyAlgorithm != x509.ECDSA {
		t.Errorf("expected an ECDSA certificate but got %s", cert.PublicKeyAlgorithm)
	}

	if err := cert.VerifyHostname("localhost"); err != nil {
		t.Errorf("expected localhost SAN: %v", err)
	}
	for _, ip := range []string{"127.0.0.1", "::1"} {
		if err := cert.VerifyHostname(ip); err != nil {
			t.Errorf("expected %s SAN: %v", ip, err)
		}
	}

	now := time.Now()
	if cert.NotBefore.After(now) {
		t.Errorf("expected NotBefore in the past, got %s", cert.NotBefore)
	}
	if minValid := now.Add(300 * 24 * time.Hour); cert.NotAfter.Before(minValid) {
		t.Errorf("expected NotAfter of about a year, got %s", cert.NotAfter)
	}

	// The key must match the certificate.
	if _, err := tls.X509KeyPair(LocalhostCert(), LocalhostKey()); err != nil {
		t.Errorf("LocalhostCert and LocalhostKey do not pair: %v", err)
	}
}
