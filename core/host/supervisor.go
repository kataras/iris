package host

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/netutil"
)

// Configurator provides an easy way to modify
// the Supervisor.
//
// Look the `Configure` func for more.
type Configurator func(su *Supervisor)

// Supervisor is the wrapper and the manager for a compatible server
// and it's relative actions, called Tasks.
//
// Interfaces are separated to return relative functionality to them.
type Supervisor struct {
	Server         *http.Server
	closedManually int32 // future use, accessed atomically (non-zero means we've called the Shutdown)
	manuallyTLS    bool  // we need that in order to determinate what to output on the console before the server begin.
	shouldWait     int32 // non-zero means that the host should wait for unblocking
	unblockChan    chan struct{}

	mu sync.Mutex

	onServe []func(TaskHost)
	// IgnoreErrors should contains the errors that should be ignored
	// on both serve functions return statements and error handlers.
	//
	// i.e: http.ErrServerClosed.Error().
	//
	// Note that this will match the string value instead of the equality of the type's variables.
	//
	// Defaults to empty.
	IgnoredErrors []string
	onErr         []func(error)
	onShutdown    []func()
}

// New returns a new host supervisor
// based on a native net/http "srv".
//
// It contains all native net/http's Server methods.
// Plus you can add tasks on specific events.
// It has its own flow, which means that you can prevent
// to return and exit and restore the flow too.
func New(srv *http.Server) *Supervisor {
	return &Supervisor{
		Server:      srv,
		unblockChan: make(chan struct{}, 1),
	}
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

// DeferFlow defers the flow of the exeuction,
// i.e: when server should return error and exit
// from app, a DeferFlow call inside a Task
// can wait for a `RestoreFlow` to exit or not exit if
// host's server is "fixed".
//
// See `RestoreFlow` too.
func (su *Supervisor) DeferFlow() {
	atomic.StoreInt32(&su.shouldWait, 1)
}

// RestoreFlow restores the flow of the execution,
// if called without a `DeferFlow` call before
// then it does nothing.
// See tests to understand how that can be useful on specific cases.
//
// See `DeferFlow` too.
func (su *Supervisor) RestoreFlow() {
	if su.isWaiting() {
		atomic.StoreInt32(&su.shouldWait, 0)
		su.mu.Lock()
		su.unblockChan <- struct{}{}
		su.mu.Unlock()
	}
}

func (su *Supervisor) isWaiting() bool {
	return atomic.LoadInt32(&su.shouldWait) != 0
}

func (su *Supervisor) newListener() (net.Listener, error) {
	// this will not work on "unix" as network
	// because UNIX doesn't supports the kind of
	// restarts we may want for the server.
	//
	// User still be able to call .Serve instead.
	l, err := netutil.TCPKeepAlive(su.Server.Addr)
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
	err = su.validateErr(err)
	if err != nil {
		su.mu.Lock()
		for _, f := range su.onErr {
			go f(err)
		}
		su.mu.Unlock()
	}
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

// Remove all channels, do it with events
// or with channels but with a different channel on each task proc
// I don't know channels are not so safe, when go func and race risk..
// so better with callbacks....
func (su *Supervisor) supervise(blockFunc func() error) error {
	host := createTaskHost(su)

	su.notifyServe(host)

	err := blockFunc()
	su.notifyErr(err)

	if su.isWaiting() {
	blockStatement:
		for {
			select {
			case <-su.unblockChan:
				break blockStatement
			}
		}
	}

	return su.validateErr(err)
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
	return su.supervise(func() error { return su.Server.Serve(l) })
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

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (su *Supervisor) ListenAndServeTLS(certFile string, keyFile string) error {
	su.manuallyTLS = true

	if certFile != "" && keyFile != "" {
		cfg := new(tls.Config)
		var err error
		cfg.Certificates = make([]tls.Certificate, 1)
		if cfg.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			return err
		}

		// manually inserted as pre-go 1.9 for any case.
		cfg.NextProtos = []string{"h2", "http/1.1"}
		su.Server.TLSConfig = cfg

		// It does nothing more than the su.Server.ListenAndServeTLS anymore.
		// - no hurt if we let it as it is
		// - no problem if we remove it as well
		// but let's comment this as proposed, fewer code is better:
		// return su.ListenAndServe()
	}

	if su.Server.TLSConfig == nil {
		return errors.New("certFile or keyFile missing")
	}

	return su.supervise(func() error { return su.Server.ListenAndServeTLS("", "") })
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

	if domain != "" {
		domains := strings.Split(domain, " ")
		hostPolicy = autocert.HostWhitelist(domains...)
	}

	autoTLSManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Email:      email,
		Cache:      cache,
	}

	srv2 := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		Addr:         ":http",
		Handler:      autoTLSManager.HTTPHandler(nil), // nil for redirect.
	}

	// register a shutdown callback to this
	// supervisor in order to close the "secondary redirect server" as well.
	su.RegisterOnShutdown(func() {
		// give it some time to close itself...
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		srv2.Shutdown(ctx)
	})
	go srv2.ListenAndServe()

	su.Server.TLSConfig = &tls.Config{
		GetCertificate:           autoTLSManager.GetCertificate,
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
		},
	}
	return su.ListenAndServeTLS("", "")
}

// RegisterOnShutdown registers a function to call on Shutdown.
// This can be used to gracefully shutdown connections that have
// undergone NPN/ALPN protocol upgrade or that have been hijacked.
// This function should start protocol-specific graceful shutdown,
// but should not wait for shutdown to complete.
func (su *Supervisor) RegisterOnShutdown(cb func()) {
	// when go1.9: replace the following lines with su.Server.RegisterOnShutdown(f)
	su.mu.Lock()
	su.onShutdown = append(su.onShutdown, cb)
	su.mu.Unlock()
}

func (su *Supervisor) notifyShutdown() {
	// when go1.9: remove the lines below
	su.mu.Lock()
	for _, f := range su.onShutdown {
		go f()
	}
	su.mu.Unlock()
	// end
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
	atomic.AddInt32(&su.closedManually, 1) // future-use
	su.notifyShutdown()
	return su.Server.Shutdown(ctx)
}
