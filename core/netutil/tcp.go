package netutil

import (
	"crypto/tls"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris/core/errors"
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
}

// Accept accepts tcp connections aka clients.
func (l tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

var (
	errPortAlreadyUsed = errors.New("port is already used")
	errRemoveUnix      = errors.New("unexpected error when trying to remove unix socket file. Addr: %s | Trace: %s")
	errChmod           = errors.New("cannot chmod %#o for %q: %s")
	errCertKeyMissing  = errors.New("you should provide certFile and keyFile for TLS/SSL")
	errParseTLS        = errors.New("couldn't load TLS, certFile=%q, keyFile=%q. Trace: %s")
)

// TCP returns a new tcp(ipv6 if supported by network) and an error on failure.
func TCP(addr string) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// TCPKeepAlive returns a new tcp keep alive Listener and an error on failure.
func TCPKeepAlive(addr string) (ln net.Listener, err error) {
	// if strings.HasPrefix(addr, "127.0.0.1") {
	// 	// it's ipv4, use ipv4 tcp listener instead of the default ipv6. Don't.
	// 	ln, err = net.Listen("tcp4", addr)
	// } else {
	// 	ln, err = TCP(addr)
	// }

	ln, err = TCP(addr)
	if err != nil {
		return nil, err
	}
	return tcpKeepAliveListener{ln.(*net.TCPListener)}, nil
}

// UNIX returns a new unix(file) Listener.
func UNIX(socketFile string, mode os.FileMode) (net.Listener, error) {
	if errOs := os.Remove(socketFile); errOs != nil && !os.IsNotExist(errOs) {
		return nil, errRemoveUnix.Format(socketFile, errOs.Error())
	}

	l, err := net.Listen("unix", socketFile)
	if err != nil {
		return nil, errPortAlreadyUsed.AppendErr(err)
	}

	if err = os.Chmod(socketFile, mode); err != nil {
		return nil, errChmod.Format(mode, socketFile, err.Error())
	}

	return l, nil
}

// TLS returns a new TLS Listener and an error on failure.
func TLS(addr, certFile, keyFile string) (net.Listener, error) {

	if certFile == "" || keyFile == "" {
		return nil, errCertKeyMissing
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errParseTLS.Format(certFile, keyFile, err)
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
	}
	return tls.NewListener(l, tlsConfig), nil
}

// LETSENCRYPT returns a new Automatic TLS Listener using letsencrypt.org service
// receives three parameters,
// the first is the host of the server,
// second can be the server name(domain) or empty if skip verification is the expected behavior (not recommended)
// and the third is optionally, the cache directory, if you skip it then the cache directory is "./certcache"
// if you want to disable cache directory then simple give it a value of empty string ""
//
// does NOT supports localhost domains for testing.
//
// this is the recommended function to use when you're ready for production state.
func LETSENCRYPT(addr string, serverName string, cacheDirOptional ...string) (net.Listener, error) {
	if portIdx := strings.IndexByte(addr, ':'); portIdx == -1 {
		addr += ":443"
	}

	l, err := TCP(addr)
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
	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}

	// use InsecureSkipVerify or ServerName to a value
	if serverName == "" {
		// if server name is invalid then bypass it
		tlsConfig.InsecureSkipVerify = true
	} else {
		tlsConfig.ServerName = serverName
	}

	return tls.NewListener(l, tlsConfig), nil
}
