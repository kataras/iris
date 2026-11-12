package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/kataras/iris/v14/core/pkg/netutil"
)

// generateTestCertPEM returns a throwaway self-signed ECDSA P-256 certificate
// and key as raw PEM contents (the CertFile and KeyFile options accept those directly),
// valid for localhost, 127.0.0.1 and ::1.
func generateTestCertPEM(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()

	cert, key, err := netutil.GenerateLocalhostCertPEM(time.Hour)
	if err != nil {
		t.Fatalf("generate certificate: %v", err)
	}

	return string(cert), string(key)
}

// startTestTLSServer starts the given options on an ephemeral 127.0.0.1 port
// and returns the server and its listening "host:port" address.
func startTestTLSServer(t *testing.T, options ServerOptions) (*Server, string) {
	t.Helper()

	options.Addr = "127.0.0.1:0"
	options.DisableStartupLog = true
	options.DisableInterruptHandler = true

	server := NewServer(options)
	addrCh := make(chan string, 1)
	server.OnListenerCreated(func(ln net.Listener, _ *url.URL) error {
		addrCh <- ln.Addr().String()
		return nil
	})

	go func() {
		if err := server.ListenAndServe(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	}()

	select {
	case addr := <-addrCh:
		t.Cleanup(func() { server.Close() })
		return server, addr
	case <-time.After(10 * time.Second):
		t.Fatal("server did not start listening in time")
		return nil, ""
	}
}

func newHTTP2Client() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			ForceAttemptHTTP2: true,
		},
	}
}

// TestServerCertFileKeyFileServesHTTP2 is a regression test: the http.Server
// instance carries the framework-built tls.Config, and HTTP/2 must still be
// negotiated over it ("h2" stays in NextProtos).
func TestServerCertFileKeyFileServesHTTP2(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)
	_, addr := startTestTLSServer(t, ServerOptions{CertFile: certPEM, KeyFile: keyPEM})

	resp, err := newHTTP2Client().Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code 200 but got %d", resp.StatusCode)
	}
	if resp.ProtoMajor != 2 {
		t.Errorf("expected an HTTP/2 response but got %s", resp.Proto)
	}
}

// TestResolveSchemeFromServerTLS is a regression test: the server used to build
// its http.Server without a TLSConfig, so netutil.IsTLS and ResolveSchemeFromServer
// (e.g. router.WithServer) always resolved "http" for HTTPS servers.
func TestResolveSchemeFromServerTLS(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)
	server, addr := startTestTLSServer(t, ServerOptions{CertFile: certPEM, KeyFile: keyPEM})

	// A completed request guarantees startHTTPServer bound the http.Server instance.
	resp, err := newHTTP2Client().Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	resp.Body.Close()

	if !netutil.IsTLS(server.httpServer) {
		t.Error("expected netutil.IsTLS to report true for a TLS server")
	}
	if got := netutil.ResolveSchemeFromServer(server.httpServer); got != "https" {
		t.Errorf("expected scheme %q but got %q", "https", got)
	}
}

// TestServerCustomTLSConfigServesHTTP2 is a regression test: a user-provided
// tls.Config without NextProtos used to be served as-is and silently lost HTTP/2.
// It now flows through the same normalization as the framework-built configs.
func TestServerCustomTLSConfigServesHTTP2(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)
	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("load key pair: %v", err)
	}

	// NextProtos deliberately nil.
	server, addr := startTestTLSServer(t, ServerOptions{TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}}})

	resp, err := newHTTP2Client().Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.ProtoMajor != 2 {
		t.Errorf("expected an HTTP/2 response but got %s", resp.Proto)
	}
	if got := netutil.ResolveSchemeFromServer(server.httpServer); got != "https" {
		t.Errorf("expected scheme %q but got %q", "https", got)
	}
}

// TestNewTLSConfigNormalizesCustomConfig tests the normalization rules of a
// user-provided tls.Config: work on a clone, fill NextProtos only when nil,
// always ensure "http/1.1", fill ServerName when empty and never touch MinVersion.
func TestNewTLSConfigNormalizesCustomConfig(t *testing.T) {
	userConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	s := &Server{options: &ServerOptions{Addr: ":8080", TLSConfig: userConfig}}

	config := s.newTLSConfig()
	if config == userConfig {
		t.Fatal("expected a clone, not the user's config itself")
	}
	if want := []string{"h2", "http/1.1"}; !slices.Equal(config.NextProtos, want) {
		t.Errorf("expected NextProtos %v but got %v", want, config.NextProtos)
	}
	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected the user's MinVersion to be respected, got %d", config.MinVersion)
	}
	if config.ServerName != "localhost:8080" {
		t.Errorf("expected ServerName to be filled, got %q", config.ServerName)
	}
	if userConfig.NextProtos != nil || userConfig.ServerName != "" {
		t.Error("the caller's config must not be mutated")
	}

	// A non-nil NextProtos without "h2" is the user's explicit choice, keep it.
	s = &Server{options: &ServerOptions{Addr: ":8080", TLSConfig: &tls.Config{NextProtos: []string{"http/1.1"}}}}
	if config = s.newTLSConfig(); slices.Contains(config.NextProtos, "h2") {
		t.Errorf("expected no h2 for an explicit http/1.1-only config, got %v", config.NextProtos)
	}

	// "http/1.1" is always ensured.
	s = &Server{options: &ServerOptions{Addr: ":8080", TLSConfig: &tls.Config{NextProtos: []string{"h2"}}}}
	if config = s.newTLSConfig(); !slices.Contains(config.NextProtos, "http/1.1") {
		t.Errorf("expected http/1.1 to be appended, got %v", config.NextProtos)
	}
}
