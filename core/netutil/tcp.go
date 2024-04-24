package netutil

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by Run, ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// A raw copy of standar library.
type tcpKeepAliveListener struct {
	*net.TCPListener
	keepAliveDur time.Duration
}

// Accept accepts tcp connections aka clients.
func (l tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return tc, err
	}
	if err = tc.SetKeepAlive(true); err != nil {
		return tc, err
	}
	if err = tc.SetKeepAlivePeriod(l.keepAliveDur); err != nil {
		return tc, err
	}
	return tc, nil
}

// TCP returns a new tcp(ipv6 if supported by network) and an error on failure.
func TCP(addr string, reuse bool) (net.Listener, error) {
	var cfg net.ListenConfig
	if reuse {
		cfg.Control = control
	}

	return cfg.Listen(context.Background(), "tcp", addr)
}

// TCPKeepAlive returns a new tcp keep alive Listener and an error on failure.
func TCPKeepAlive(addr string, reuse bool, keepAliveDur time.Duration) (ln net.Listener, err error) {
	// if strings.HasPrefix(addr, "127.0.0.1") {
	// 	// it's ipv4, use ipv4 tcp listener instead of the default ipv6. Don't.
	// 	ln, err = net.Listen("tcp4", addr)
	// } else {
	// 	ln, err = TCP(addr)
	// }

	ln, err = TCP(addr, reuse)
	if err != nil {
		return nil, err
	}
	return tcpKeepAliveListener{ln.(*net.TCPListener), keepAliveDur}, nil
}

// UNIX returns a new unix(file) Listener.
func UNIX(socketFile string, mode os.FileMode) (net.Listener, error) {
	if errOs := os.Remove(socketFile); errOs != nil && !os.IsNotExist(errOs) {
		return nil, fmt.Errorf("%s: %w", socketFile, errOs)
	}

	l, err := net.Listen("unix", socketFile)
	if err != nil {
		return nil, fmt.Errorf("port already in use: %w", err)
	}

	if err = os.Chmod(socketFile, mode); err != nil {
		return nil, fmt.Errorf("cannot chmod %#o for %q: %w", mode, socketFile, err)
	}

	return l, nil
}

// TLS returns a new TLS Listener and an error on failure.
func TLS(addr, certFile, keyFile string) (net.Listener, error) {
	if certFile == "" || keyFile == "" {
		return nil, errors.New("empty certFile or KeyFile")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return CERT(addr, cert)
}

// CERT returns a listener which contans tls.Config with the provided certificate, use for ssl.
func CERT(addr string, cert tls.Certificate) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
	}
	return tls.NewListener(l, tlsConfig), nil
}

// LETSENCRYPT returns a new Automatic TLS Listener using letsencrypt.org service
// receives three parameters,
// the first is the host of the server,
// second one should declare if the underline tcp listener can be binded more than once,
// third can be the server name(domain) or empty if skip verification is the expected behavior (not recommended),
// and the forth is optionally, the cache directory, if you skip it then the cache directory is "./certcache"
// if you want to disable cache directory then simple give it a value of empty string ""
//
// does NOT supports localhost domains for testing.
//
// this is the recommended function to use when you're ready for production state.
func LETSENCRYPT(addr string, reuse bool, serverName string, cacheDirOptional ...string) (net.Listener, error) {
	if portIdx := strings.IndexByte(addr, ':'); portIdx == -1 {
		addr += ":443"
	}

	l, err := TCP(addr, reuse)
	if err != nil {
		return nil, err
	}

	cacheDir := "./certcache"
	if len(cacheDirOptional) > 0 {
		cacheDir = cacheDirOptional[0]
	}

	m := autocert.Manager{
		Prompt: autocert.AcceptTOS,
	} // HostPolicy is missing, if user wants it, then she/he should manually

	if cacheDir == "" {
		// then the user passed empty by own will, then I guess she/he doesnt' want any cache directory
	} else {
		m.Cache = autocert.DirCache(cacheDir)
	}
	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate, MinVersion: tls.VersionTLS13}

	// use InsecureSkipVerify or ServerName to a value
	if serverName == "" {
		// if server name is invalid then bypass it
		tlsConfig.InsecureSkipVerify = true
	} else {
		tlsConfig.ServerName = serverName
	}

	return tls.NewListener(l, tlsConfig), nil
}
