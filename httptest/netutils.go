package httptest

import (
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/kataras/iris/v14/core/pkg/netutil"
)

var (
	localhostCertOnce sync.Once
	localhostCertPEM  []byte
	localhostKeyPEM   []byte
)

// generateLocalhostCert builds a self-signed ECDSA P-256 certificate for
// localhost, 127.0.0.1 and ::1, valid from an hour ago until a year from now.
// It replaces the hardcoded RSA/SHA-1 PEM pair this package used to ship,
// which had no subjectAltName and a fixed expiration date.
func generateLocalhostCert() {
	certPEM, keyPEM, err := netutil.GenerateLocalhostCertPEM(365 * 24 * time.Hour)
	if err != nil {
		panic("httptest: " + err.Error())
	}

	localhostCertPEM, localhostKeyPEM = certPEM, keyPEM
}

// LocalhostCert returns a PEM-encoded, self-signed ECDSA P-256 TLS certificate
// with subjectAltNames for "localhost", 127.0.0.1 and ::1.
// It is generated once per process on first use and stays valid for a year,
// so it passes modern hostname verification when added to a client's root pool.
func LocalhostCert() []byte {
	localhostCertOnce.Do(generateLocalhostCert)
	return localhostCertPEM
}

// LocalhostKey returns the PEM-encoded private key for `LocalhostCert`.
func LocalhostKey() []byte {
	localhostCertOnce.Do(generateLocalhostCert)
	return localhostKeyPEM
}

// NewLocalListener returns a new ipv4 "127.0.0.1:0"
// or tcp6 "[::1]:0" tcp listener.
func NewLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(err)
		}
	}
	return l
}

// NewLocalTLSListener returns a new tls listener
// based on the "tcpListener", if "tcpListener" is nil
// it make use of the `NewLocalListener`.
// Cert and Key are `LocalhostCert` and `LocalhostKey` respectfully.
func NewLocalTLSListener(tcpListener net.Listener) net.Listener {
	if tcpListener == nil {
		tcpListener = NewLocalListener()
	}

	cert, err := tls.X509KeyPair(LocalhostCert(), LocalhostKey())
	if err != nil {
		panic(err)
	}

	cfg := &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{cert},
	}
	return tls.NewListener(tcpListener, cfg)
}
