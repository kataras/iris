package host

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v12/core/netutil"

	"golang.org/x/crypto/acme/autocert"
)

// Configurator provides an easy way to modify
// the Supervisor.
//
// Look the `Configure` func for more.
type Configurator func(su *Supervisor)

// NonBlocking sets the server to non-blocking mode. Use its `Wait` method to wait for server to be up and running.
func NonBlocking() Configurator {
	return func(su *Supervisor) {
		su.nonBlocking = true
	}
}

// Supervisor is the wrapper and the manager for a compatible server
// and it's relative actions, called Tasks.
//
// Interfaces are separated to return relative functionality to them.
type Supervisor struct {
	Server *http.Server
	// FriendlyAddr can be set to customize the "Now Listening on: {FriendlyAddr}".
	FriendlyAddr                   string // e.g mydomain.com instead of :443 when AutoTLS is used, see `WriteStartupLogOnServe` task.
	disableHTTP1ToHTTP2Redirection bool

	closedByInterruptHandler uint32 // non-zero means that the end-developer interrupted it by-purpose.
	manuallyTLS              bool   // we need that in order to determinate what to output on the console before the server begin.
	autoTLS                  bool

	mu sync.RWMutex

	onServe []func(TaskHost)
	// IgnoreErrors should contains the errors that should be ignored
	// on both serve functions return statements and error handlers.
	//
	// Note that this will match the string value instead of the equality of the type's variables.
	//
	// Defaults to empty.
	IgnoredErrors []string
	onErr         []func(error)

	// Fallback should return a http.Server, which may already running
	// to handle the HTTP/1.1 clients when TLS/AutoTLS.
	// On manual TLS the accepted "challengeHandler" just returns the passed handler,
	// otherwise it binds to the acme challenge wrapper.
	// Example:
	//      Fallback = func(h func(fallback http.Handler) http.Handler) *http.Server {
	//          s := &http.Server{
	//             Handler: h(myServerHandler),
	//             ...otherOptions
	//          }
	//          go s.ListenAndServe()
	//          return s
	//      }
	Fallback func(challegeHandler func(fallback http.Handler) http.Handler) *http.Server

	// See `iris.Configuration.SocketSharding`.
	SocketSharding bool
	// If more than zero then tcp keep alive listener is attached instead of the simple TCP listener.
	// See `iris.Configuration.KeepAlive`
	KeepAlive time.Duration

	address     string
	nonBlocking bool
	waiter      *Waiter
}

// New returns a new host supervisor
// based on a native net/http "srv".
//
// It contains all native net/http's Server methods.
// Plus you can add tasks on specific events.
// It has its own flow, which means that you can prevent
// to return and exit and restore the flow too.
func New(srv *http.Server) *Supervisor {
	su := &Supervisor{
		Server: srv,
	}

	su.waiter = NewWaiter(7, su.getAddress)
	return su
}

// Configure accepts one or more `Configurator`.
// With this function you can use simple functions
// that are spread across your app to modify
// the supervisor, these Configurators can be
// used on any Supervisor instance.
//
// Look `Configurator` too.
//
// Returns itself.
func (su *Supervisor) Configure(configurators ...Configurator) *Supervisor {
	for _, conf := range configurators {
		conf(su)
	}
	return su
}

// NoRedirect should be called before `ListenAndServeTLS` when
// secondary http1 to http2 server is not required. This method will disable
// the automatic registration of secondary http.Server
// which would redirect "http://" requests to their "https://" equivalent.
func (su *Supervisor) NoRedirect() {
	su.disableHTTP1ToHTTP2Redirection = true
}

func (su *Supervisor) newListener() (net.Listener, error) {
	var (
		l   net.Listener
		err error
	)

	if su.KeepAlive > 0 {
		l, err = netutil.TCPKeepAlive(su.Server.Addr, su.SocketSharding, su.KeepAlive)
	} else {
		l, err = netutil.TCP(su.Server.Addr, su.SocketSharding)
	}

	if err != nil {
		return nil, err
	}

	// here we can check for sure, without the need of the supervisor's `manuallyTLS` field.
	if netutil.IsTLS(su.Server) {
		// means tls
		tlsl := tls.NewListener(l, su.Server.TLSConfig)
		return tlsl, nil
	}

	return l, nil
}

// RegisterOnError registers a function to call when errors occurred by the underline http server.
func (su *Supervisor) RegisterOnError(cb func(error)) {
	su.mu.Lock()
	su.onErr = append(su.onErr, cb)
	su.mu.Unlock()
}

func (su *Supervisor) validateErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, http.ErrServerClosed) && atomic.LoadUint32(&su.closedByInterruptHandler) > 0 {
		return nil
	}

	su.mu.Lock()
	defer su.mu.Unlock()

	for _, e := range su.IgnoredErrors {
		if err.Error() == e {
			return nil
		}
	}

	return err
}

func (su *Supervisor) notifyErr(err error) {
	if err == nil {
		return
	}

	su.mu.Lock()
	for _, f := range su.onErr {
		go f(err)
	}
	su.mu.Unlock()
}

func (su *Supervisor) getAddress() string {
	su.mu.RLock()
	addr := su.address
	su.mu.RUnlock()
	return addr
}

func (su *Supervisor) setAddress(addr string) {
	su.mu.Lock()
	su.address = addr
	su.mu.Unlock()
}

// RegisterOnServe registers a function to call on
// Serve/ListenAndServe/ListenAndServeTLS/ListenAndServeAutoTLS.
func (su *Supervisor) RegisterOnServe(cb func(TaskHost)) {
	su.mu.Lock()
	su.onServe = append(su.onServe, cb)
	su.mu.Unlock()
}

func (su *Supervisor) notifyServe(host TaskHost) {
	su.mu.Lock()
	for _, f := range su.onServe {
		go f(host)
	}
	su.mu.Unlock()
}

func (su *Supervisor) supervise(blockFunc func() error) error {
	host := createTaskHost(su)

	su.notifyServe(host)
	atomic.StoreUint32(&su.closedByInterruptHandler, 0)

	if su.nonBlocking {
		go func() {
			err := blockFunc()
			if err != nil {
				su.waiter.Fail(err)
			}

			err = su.validateErr(err)
			su.notifyErr(err)
		}()

		return nil
	}

	err := blockFunc()
	err = su.validateErr(err)
	su.notifyErr(err)

	return err
}

// Wait blocks until server is up and running or a serve failure.
func (su *Supervisor) Wait(ctx context.Context) error {
	return su.waiter.Wait(ctx)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines read requests and
// then call su.server.Handler to reply to them.
//
// For HTTP/2 support, server.TLSConfig should be initialized to the
// provided listener's TLS Config before calling Serve. If
// server.TLSConfig is non-nil and doesn't include the string "h2" in
// Config.NextProtos, HTTP/2 support is not enabled.
//
// Serve always returns a non-nil error. After Shutdown or Close, the
// returned error is http.ErrServerClosed.
func (su *Supervisor) Serve(l net.Listener) error {
	su.setAddress(l.Addr().String())

	return su.supervise(func() error {
		return su.Server.Serve(l)
	})
}

// ListenAndServe listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
func (su *Supervisor) ListenAndServe() error {
	l, err := su.newListener()
	if err != nil {
		return err
	}
	return su.Serve(l)
}

func loadCertificate(c, k string) (*tls.Certificate, error) {
	var (
		cert tls.Certificate
		err  error
	)

	if fileExists(c) && fileExists(k) {
		// act them as files in the system.
		cert, err = tls.LoadX509KeyPair(c, k)
	} else {
		// act them as raw contents.
		cert, err = tls.X509KeyPair([]byte(c), []byte(k))
	}

	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (su *Supervisor) ListenAndServeTLS(certFileOrContents string, keyFileOrContents string) error {
	var getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)

	// If tls.Config configured manually through a host configurator then skip that
	// and let the redirection service registered alone.
	// e.g. https://github.com/kataras/iris/issues/1481#issuecomment-605621255
	if su.Server.TLSConfig == nil {
		if certFileOrContents == "" && keyFileOrContents == "" {
			return errors.New("empty certFileOrContents or keyFileOrContents and Server.TLSConfig")
		}

		cert, err := loadCertificate(certFileOrContents, keyFileOrContents)
		if err != nil {
			return err
		}

		getCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert, nil
		}
	}

	su.manuallyTLS = true
	return su.runTLS(getCertificate, nil)
}

// ListenAndServeAutoTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Server's certificates are auto generated from LETSENCRYPT using
// the golang/x/net/autocert package.
//
// The whitelisted domains are separated by whitespace in "domain" argument, i.e "iris-go.com".
// If empty, all hosts are currently allowed. This is not recommended,
// as it opens a potential attack where clients connect to a server
// by IP address and pretend to be asking for an incorrect host name.
// Manager will attempt to obtain a certificate for that host, incorrectly,
// eventually reaching the CA's rate limit for certificate requests
// and making it impossible to obtain actual certificates.
//
// For an "e-mail" use a non-public one, letsencrypt needs that for your own security.
//
// The "cacheDir" is being, optionally, used to provide cache
// stores and retrieves previously-obtained certificates.
// If empty, certs will only be cached for the lifetime of the auto tls manager.
//
// Note: The domain should be like "iris-go.com www.iris-go.com",
// the e-mail like "kataras2006@hotmail.com" and the cacheDir like "letscache"
// The `ListenAndServeAutoTLS` will start a new server for you,
// which will redirect all http versions to their https, including subdomains as well.
func (su *Supervisor) ListenAndServeAutoTLS(domain string, email string, cacheDir string) error {
	var (
		cache      autocert.Cache
		hostPolicy autocert.HostPolicy
	)

	if cacheDir != "" {
		cache = autocert.DirCache(cacheDir)
	}

	if strings.TrimSpace(domain) != "" {
		domains := strings.Split(domain, " ")
		su.FriendlyAddr = strings.Join(domains, ", ")
		hostPolicy = autocert.HostWhitelist(domains...)
	}

	su.autoTLS = true

	autoTLSManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Email:      email,
		Cache:      cache,
	}

	return su.runTLS(autoTLSManager.GetCertificate, autoTLSManager.HTTPHandler)
}

func (su *Supervisor) runTLS(getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error), challengeHandler func(fallback http.Handler) http.Handler) error {
	if su.manuallyTLS && !su.disableHTTP1ToHTTP2Redirection {
		// If manual TLS and auto-redirection is enabled,
		// then create an empty challenge handler so the :80 server starts.
		challengeHandler = func(h http.Handler) http.Handler { // it is always nil on manual TLS.
			target, _ := url.Parse("https://" + netutil.ResolveVHost(su.Server.Addr)) // e.g. https://localhost:443
			http1Handler := RedirectHandler(target, http.StatusMovedPermanently)
			return http1Handler
		}
	}

	if challengeHandler != nil {
		http1Server := &http.Server{
			Addr:              ":http",
			Handler:           challengeHandler(nil), // nil for redirection.
			ReadTimeout:       su.Server.ReadTimeout,
			ReadHeaderTimeout: su.Server.ReadHeaderTimeout,
			WriteTimeout:      su.Server.WriteTimeout,
			IdleTimeout:       su.Server.IdleTimeout,
			MaxHeaderBytes:    su.Server.MaxHeaderBytes,
		}

		if su.Fallback == nil {
			if !su.manuallyTLS && su.disableHTTP1ToHTTP2Redirection {
				// automatic redirection was disabled but Fallback was not registered.
				return fmt.Errorf("autotls: use iris.AutoTLSNoRedirect instead")
			}
			go http1Server.ListenAndServe()
		} else {
			// if it's manual TLS still can have its own Fallback server here,
			// the handler will be the redirect one, the difference is that it can run on any port.
			srv := su.Fallback(challengeHandler)
			if srv == nil {
				if !su.manuallyTLS {
					return fmt.Errorf("autotls: relies on an HTTP/1.1 server")
				}
				// for any case the end-developer decided to return nil here,
				// we proceed with the automatic redirection.
				srv = http1Server
				go srv.ListenAndServe()
			} else {
				if srv.Addr == "" {
					srv.Addr = ":http"
				}
				// } else if !su.manuallyTLS && srv.Addr != ":80" && srv.Addr != ":http" {
				// 	hostname, _, _ := net.SplitHostPort(su.Server.Addr)
				// 	return fmt.Errorf("autotls: The HTTP-01 challenge relies on http://%s:80/.well-known/acme-challenge/", hostname)
				// }

				if srv.Handler == nil {
					// handler was nil, caller wanted to change the server's options like read/write timeout.
					srv.Handler = http1Server.Handler
					go srv.ListenAndServe() // automatically start it, we assume the above ^
				}
				http1Server = srv // to register the shutdown event.
			}
		}

		su.RegisterOnShutdown(func() {
			timeout := 10 * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			http1Server.Shutdown(ctx)
		})
	}

	if su.Server.TLSConfig == nil {
		// If tls.Config is NOT configured manually through a host configurator,
		// then create it.
		su.Server.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			GetCertificate:           getCertificate,
			PreferServerCipherSuites: true,
			NextProtos:               []string{"h2", "http/1.1"},
			CurvePreferences: []tls.CurveID{
				tls.CurveP521,
				tls.CurveP384,
				tls.CurveP256,
			},
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				// tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, G402: TLS Bad Cipher Suite
				0xC028, /* TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384 */
			},
		}
	}

	ln, err := netutil.TCP(su.Server.Addr, su.SocketSharding)
	if err != nil {
		return err
	}

	return su.supervise(func() error { return su.Server.ServeTLS(ln, "", "") })
}

// RegisterOnShutdown registers a function to call on Shutdown.
// This can be used to gracefully shutdown connections that have
// undergone NPN/ALPN protocol upgrade or that have been hijacked.
// This function should start protocol-specific graceful shutdown,
// but should not wait for shutdown to complete.
//
// Callbacks will run as separate go routines.
func (su *Supervisor) RegisterOnShutdown(cb func()) {
	su.Server.RegisterOnShutdown(cb)
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// then the context's error is returned.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired.
func (su *Supervisor) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return su.Server.Shutdown(ctx)
}

func (su *Supervisor) shutdownOnInterrupt(ctx context.Context) {
	atomic.StoreUint32(&su.closedByInterruptHandler, 1)
	su.Shutdown(ctx)
}

// fileExists tries to report whether a local physical file of "filename" exists.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
