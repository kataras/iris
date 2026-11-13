package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/acme"
)

// The Letsencrypt option serves no HTTP-01 handler, so tls-alpn-01 (RFC 8737) is
// the one ACME challenge its listener can answer: the CA connects with the domain
// as the server name and "acme-tls/1" as the only application protocol, and expects
// the challenge certificate back. The tests of this file play the CA's side of that
// handshake over loopback. None of them starts an ACME order or leaves the machine,
// so what they cannot show is a certificate being issued: that takes a CA which
// reaches the server on port 443 for a name it resolves.

// acmeTestDomain is the name under validation. Nothing resolves or contacts it:
// the ".test" top-level domain is reserved (RFC 6761) and the autocert manager
// never contacts the CA for a tls-alpn-01 handshake.
const acmeTestDomain = "iris-acme.test"

// dialTLSALPN01 performs the handshake of a tls-alpn-01 validation against addr.
func dialTLSALPN01(addr, domain string) (tls.ConnectionState, error) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{
		ServerName:         domain,
		NextProtos:         []string{acme.ALPNProto},
		InsecureSkipVerify: true, // a challenge certificate is self-signed.
	})
	if err != nil {
		return tls.ConnectionState{}, err
	}
	defer conn.Close()

	return conn.ConnectionState(), nil
}

// serverLogLines receives what the http.Server reports through the LogWriter option.
type serverLogLines chan string

func (l serverLogLines) Write(p []byte) (int, error) {
	select {
	case l <- string(p):
	default: // never block the server.
	}

	return len(p), nil
}

// waitFor returns the first reported line that contains substr.
func (l serverLogLines) waitFor(t *testing.T, substr string) string {
	t.Helper()

	var (
		timeout = time.After(10 * time.Second)
		others  []string
	)

	for {
		select {
		case line := <-l:
			if strings.Contains(line, substr) {
				return line
			}
			others = append(others, line)
		case <-timeout:
			t.Fatalf("no server log line contains %q, got %q", substr, others)
			return ""
		}
	}
}

// TestNewTLSConfigACMEProtocol tests that "acme-tls/1" is offered by the Letsencrypt
// configuration, after "h2" and "http/1.1" so their preference does not move, and
// that the configurations without Letsencrypt keep offering exactly what they did.
func TestNewTLSConfigACMEProtocol(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)

	tests := []struct {
		name    string
		options ServerOptions
		want    []string
	}{
		{"letsencrypt", ServerOptions{Letsencrypt: true}, []string{"h2", "http/1.1", acme.ALPNProto}},
		{"letsencrypt with a local fallback", ServerOptions{Letsencrypt: true, CertFile: certPEM, KeyFile: keyPEM}, []string{"h2", "http/1.1", acme.ALPNProto}},
		{"certificate and key", ServerOptions{CertFile: certPEM, KeyFile: keyPEM}, []string{"h2", "http/1.1"}},
		{"custom config", ServerOptions{TLSConfig: &tls.Config{}}, []string{"h2", "http/1.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.options.Addr = ":8080"
			s := &Server{options: &tt.options}

			if got := s.newTLSConfig().NextProtos; !slices.Equal(got, tt.want) {
				t.Errorf("expected NextProtos %v but got %v", tt.want, got)
			}
		})
	}
}

// TestLetsencryptHandshakeReachesAutocert is a regression test: the Letsencrypt
// configuration listed only "h2" and "http/1.1", so a tls-alpn-01 validation was
// refused at protocol negotiation ("tls: no application protocol") before the
// autocert manager was asked for anything, and no certificate could be issued.
//
// It proves that the handshake now passes negotiation and that the manager takes
// it for a tls-alpn-01 challenge: with no order in flight it holds no challenge
// certificate for the name, and that refusal is the error the server logs.
func TestLetsencryptHandshakeReachesAutocert(t *testing.T) {
	logLines := make(serverLogLines, 8)
	_, addr := startTestTLSServer(t, ServerOptions{Letsencrypt: true, LogWriter: logLines})

	_, err := dialTLSALPN01(addr, acmeTestDomain)
	if err == nil {
		t.Fatal("expected the handshake to be refused: the manager holds no challenge certificate for the name")
	}
	if strings.Contains(err.Error(), "no application protocol") {
		t.Fatalf("the handshake stopped at protocol negotiation, before the autocert manager: %v", err)
	}

	// The wording belongs to golang.org/x/crypto/acme/autocert (Manager.GetCertificate).
	logLines.waitFor(t, `acme/autocert: no token cert for "`+acmeTestDomain+`"`)
}

// TestLetsencryptServesChallengeCertificate completes the same handshake. During
// an order the manager keeps the challenge certificate in memory and in its cache
// (so that several instances behind one address can answer); the test puts one in
// the cache and proves that the server selects "acme-tls/1" and presents that very
// certificate, which is all a tls-alpn-01 validation looks at.
//
// The cache entry follows autocert's DirCache layout: a "<domain>+token" file
// holding the PEM key followed by the PEM certificate.
func TestLetsencryptServesChallengeCertificate(t *testing.T) {
	// The cache is the "./letsencrypt" directory (cacheDir) of the working directory.
	dir := t.TempDir()
	t.Chdir(dir)

	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate account key: %v", err)
	}
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate certificate key: %v", err)
	}
	client := &acme.Client{Key: accountKey}
	challengeCert, err := client.TLSALPN01ChallengeCert("token", acmeTestDomain, acme.WithKey(certKey))
	if err != nil {
		t.Fatalf("generate challenge certificate: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		t.Fatalf("marshal certificate key: %v", err)
	}
	entry := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	entry = append(entry, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: challengeCert.Certificate[0]})...)

	cachePath := filepath.Join(dir, cacheDir)
	if err = os.Mkdir(cachePath, 0o700); err != nil {
		t.Fatalf("create cache directory: %v", err)
	}
	if err = os.WriteFile(filepath.Join(cachePath, acmeTestDomain+"+token"), entry, 0o600); err != nil {
		t.Fatalf("write cache entry: %v", err)
	}

	_, addr := startTestTLSServer(t, ServerOptions{Letsencrypt: true})

	state, err := dialTLSALPN01(addr, acmeTestDomain)
	if err != nil {
		t.Fatalf("tls-alpn-01 handshake failed: %v", err)
	}
	if state.NegotiatedProtocol != acme.ALPNProto {
		t.Errorf("expected protocol %q but got %q", acme.ALPNProto, state.NegotiatedProtocol)
	}
	if len(state.PeerCertificates) != 1 || !bytes.Equal(state.PeerCertificates[0].Raw, challengeCert.Certificate[0]) {
		t.Error("expected the server to present the challenge certificate of the autocert manager")
	}
}

// TestLetsencryptServesHTTPProtocols tests that the ordinary negotiation of a
// Letsencrypt server is unchanged: HTTP/2 for a client that offers it and HTTP/1.1
// for one that does not.
//
// CertFile and KeyFile are the local fallback of the Letsencrypt option: a handshake
// to an IP address carries no server name, the manager refuses it without contacting
// the CA and the local certificate is served instead.
func TestLetsencryptServesHTTPProtocols(t *testing.T) {
	certPEM, keyPEM := generateTestCertPEM(t)
	_, addr := startTestTLSServer(t, ServerOptions{Letsencrypt: true, CertFile: certPEM, KeyFile: keyPEM})

	resp, err := newHTTP2Client().Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTP/2 request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || resp.ProtoMajor != 2 {
		t.Errorf("expected a 200 HTTP/2 response but got %d %s", resp.StatusCode, resp.Proto)
	}

	http1Client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"http/1.1"}},
		},
	}
	resp, err = http1Client.Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTP/1.1 request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || resp.ProtoMajor != 1 || resp.TLS.NegotiatedProtocol != "http/1.1" {
		t.Errorf("expected a 200 HTTP/1.1 response but got %d %s (negotiated %q)", resp.StatusCode, resp.Proto, resp.TLS.NegotiatedProtocol)
	}
}
