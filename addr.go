package iris

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/go-errors"
	"golang.org/x/crypto/acme/autocert"
)

var (
	errPortAlreadyUsed = errors.New("Port is already used")
	errRemoveUnix      = errors.New("Unexpected error when trying to remove unix socket file. Addr: %s | Trace: %s")
	errChmod           = errors.New("Cannot chmod %#o for %q: %s")
	errCertKeyMissing  = errors.New("You should provide certFile and keyFile for TLS/SSL")
	errParseTLS        = errors.New("Couldn't load TLS, certFile=%q, keyFile=%q. Trace: %s")
)

// TCP4 returns a new tcp4 Listener
func TCP4(addr string) (net.Listener, error) {
	return net.Listen("tcp4", ParseHost(addr))
}

// TCPKeepAlive returns a new tcp4 keep alive Listener
func TCPKeepAlive(addr string) (net.Listener, error) {
	ln, err := TCP4(addr)
	if err != nil {
		return nil, err
	}
	return TCPKeepAliveListener{ln.(*net.TCPListener)}, err
}

// UNIX returns a new unix(file) Listener
func UNIX(socketFile string, mode os.FileMode) (net.Listener, error) {
	if errOs := os.Remove(socketFile); errOs != nil && !os.IsNotExist(errOs) {
		return nil, errRemoveUnix.Format(socketFile, errOs.Error())
	}

	listener, err := net.Listen("unix", socketFile)
	if err != nil {
		return nil, errPortAlreadyUsed.AppendErr(err)
	}

	if err = os.Chmod(socketFile, mode); err != nil {
		return nil, errChmod.Format(mode, socketFile, err.Error())
	}

	return listener, nil
}

// TLS returns a new TLS Listener
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

// CERT returns a listener which contans tls.Config with the provided certificate, use for ssl
func CERT(addr string, cert tls.Certificate) (net.Listener, error) {
	ln, err := TCP4(addr)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
	}
	return tls.NewListener(ln, tlsConfig), nil
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
// this is the recommended function to use when you're ready for production state
func LETSENCRYPT(addr string, serverName string, cacheDirOptional ...string) (net.Listener, error) {
	if portIdx := strings.IndexByte(addr, ':'); portIdx == -1 {
		addr += ":443"
	}

	ln, err := TCP4(addr)
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

	tlsLn := tls.NewListener(ln, tlsConfig)

	return tlsLn, nil
}

// TCPKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections.
// Dead TCP connections (e.g. closing laptop mid-download) eventually
// go away
// It is not used by default if you want to pass a keep alive listener
// then just pass the child listener, example:
// listener := iris.TCPKeepAliveListener{iris.TCP4(":8080").(*net.TCPListener)}
type TCPKeepAliveListener struct {
	*net.TCPListener
}

// Accept implements the listener and sets the keep alive period which is 3minutes
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return
	}
	err = tc.SetKeepAlivePeriod(3 * time.Minute)
	if err != nil {
		return
	}
	return tc, nil
}

///TODO: ?
// func (ln TCPKeepAliveListener) Close() error {
// 	return nil
// }

// ParseHost tries to convert a given string to an address which is compatible with net.Listener and server
func ParseHost(addr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := addr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOSTNAME"); oshost != "" {
			a = oshost
			// check for port also here
			if osport := os.Getenv("PORT"); osport != "" {
				a += ":" + osport
			}
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		} else {
			a = ":http"
		}
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		if a[portIdx:] == ":https" {
			a = DefaultServerHostname + ":443"
		} else {
			// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
			a = DefaultServerHostname + a
		}
	}

	/* changed my mind, don't add 80, this will cause problems on unix listeners, and it's not really necessary because we take the port using parsePort
	if portIdx := strings.IndexByte(a, ':'); portIdx < 0 {
		// missing port part, add it
		a = a + ":80"
	}*/

	return a
}

// ParseHostname receives an addr of form host[:port] and returns the hostname part of it
// ex: localhost:8080 will return the `localhost`, mydomain.com:8080 will return the 'mydomain'
func ParseHostname(addr string) string {
	idx := strings.IndexByte(addr, ':')
	if idx == 0 {
		// only port, then return 0.0.0.0
		return "0.0.0.0"
	} else if idx > 0 {
		return addr[0:idx]
	}
	// it's already hostname
	return addr
}

// ParsePort receives an addr of form host[:port] and returns the port part of it
// ex: localhost:8080 will return the `8080`, mydomain.com will return the '80'
func ParsePort(addr string) int {
	if portIdx := strings.IndexByte(addr, ':'); portIdx != -1 {
		afP := addr[portIdx+1:]
		p, err := strconv.Atoi(afP)
		if err == nil {
			return p
		} else if afP == "https" { // it's not number, check if it's :https
			return 443
		}
	}
	return 80
}

const (
	// SchemeHTTPS returns "https://" (full)
	SchemeHTTPS = "https://"
	// SchemeHTTP returns "http://" (full)
	SchemeHTTP = "http://"
)

// ParseScheme returns the scheme based on the host,addr,domain
// Note: the full scheme not just http*,https* *http:// *https://
func ParseScheme(domain string) string {
	// pure check
	if strings.HasPrefix(domain, SchemeHTTPS) || ParsePort(domain) == 443 {
		return SchemeHTTPS
	}
	return SchemeHTTP
}

// ProxyHandler returns a new net/http.Handler which works as 'proxy', maybe doesn't suits you look its code before using that in production
var ProxyHandler = func(redirectSchemeAndHost string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// override the handler and redirect all requests to this addr
		redirectTo := redirectSchemeAndHost
		fakehost := r.URL.Host
		path := r.URL.EscapedPath()
		if strings.Count(fakehost, ".") >= 3 { // propably a subdomain, pure check but doesn't matters don't worry
			if sufIdx := strings.LastIndexByte(fakehost, '.'); sufIdx > 0 {
				// check if the last part is a number instead of .com/.gr...
				// if it's number then it's propably is 0.0.0.0 or 127.0.0.1... so it shouldn' use  subdomain
				if _, err := strconv.Atoi(fakehost[sufIdx+1:]); err != nil {
					// it's not number then process the try to parse the subdomain
					redirectScheme := ParseScheme(redirectSchemeAndHost)
					realHost := strings.Replace(redirectSchemeAndHost, redirectScheme, "", 1)
					redirectHost := strings.Replace(fakehost, fakehost, realHost, 1)
					redirectTo = redirectScheme + redirectHost + path
					http.Redirect(w, r, redirectTo, StatusMovedPermanently)
					return
				}
			}
		}
		if path != "/" {
			redirectTo += path
		}
		if redirectTo == r.URL.String() {
			return
		}

		//	redirectTo := redirectSchemeAndHost + r.RequestURI

		http.Redirect(w, r, redirectTo, StatusMovedPermanently)
	}
}

// Proxy not really a proxy, it's just
// starts a server listening on proxyAddr but redirects all requests to the redirectToSchemeAndHost+$path
// nothing special, use it only when you want to start a secondary server which its only work is to redirect from one requested path to another
//
// returns a close function
func Proxy(proxyAddr string, redirectSchemeAndHost string) func(context.Context) error {
	proxyAddr = ParseHost(proxyAddr)

	// override the handler and redirect all requests to this addr
	h := ProxyHandler(redirectSchemeAndHost)
	prx := New()

	prx.Adapt(RouterBuilderPolicy(func(RouteRepository, ContextPool) http.Handler {
		return h
	}))

	go prx.Listen(proxyAddr)
	time.Sleep(150 * time.Millisecond)

	return prx.Shutdown
}
