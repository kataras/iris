package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14/internal/signalhandler"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// ServerOptions is the configuration for the server.
// It's used to configure the server's network address, timeouts, TLS, LetsEncrypt, Ngrok, and other options.
// See "NewServer" package-level function for more.
type ServerOptions struct {
	// The network must be "tcp", "tcp4", "tcp6", "unix" or "unixpacket".
	//
	// Defaults to "tcp".
	Network string `json:"network,omitempty" yaml:"Network,omitempty" toml:"Network,omitempty"`
	// The address has the form "host:port".
	// The host must be a literal IP address, or a host name that can be
	// resolved to IP addresses.
	// The port must be a literal port number or a service name.
	// If the host is a literal IPv6 address it must be enclosed in square
	// brackets, as in "[2001:db8::1]:80" or "[fe80::1%zone]:80".
	//
	// If Addr is an empty string, ":http" is used unless TLSConfig is provided, in which case
	// it will be ":https".
	Addr string `json:"addr,omitempty" yaml:"Addr,omitempty" toml:"Addr,omitempty"`
	// KeepAlive specifies the keep-alive period for network
	// connections accepted by this listener.
	// If zero, keep-alives are enabled if supported by the protocol
	// and operating system. Network protocols or operating systems
	// that do not support keep-alives ignore this field.
	// If negative, keep-alives are disabled.
	KeepAlive time.Duration `json:"keep_alive,omitzero" yaml:"KeepAlive" toml:"keep_alive,omitempty"`
	// SocketSharding enables SO_REUSEPORT (or SO_REUSEADDR on Windows).
	// Allowing for linear scaling of server performance on multi-CPU servers.
	// Defaults to false.
	SocketSharding bool `json:"socket_sharding,omitzero" yaml:"SocketSharding,omitempty" toml:"SocketSharding,omitempty"`

	// Hosts is the list of hosts that the certificate is valid for.
	// If this is empty, the Application's Host will be used, if that is empty too,
	// then the resolved host from Addr will be used instead.
	// Framework should provide the root host and all subdomains registered (+.$FRAMEWORK_HOST),
	// if a wildcard is used, then add (+.FRAMEWORK_HOST) it too.
	// The first entry is expected to be the root domain, will be used on services like letsencrypt.
	Hosts []string `json:"hosts,omitempty" yaml:"Hosts,omitempty" toml:"Hosts,omitempty"`
	// TLSConfig optionally provides a TLS configuration for use.
	// If this is set then LetsEncrypt and Ngrok and CertFile and KeyFile options SHOULD be nil.
	TLSConfig *tls.Config `json:"-" yaml:"-" toml:"-"`
	// CertFile and KeyFile are the paths to the TLS certificate and key files.
	// The file must contain PEM encoded data.
	CertFile string `json:"cert_file,omitempty" yaml:"CertFile,omitempty" toml:"CertFile,omitempty"`
	// KeyFile is the path to the TLS key file.
	// The file must contain PEM encoded data.
	KeyFile string `json:"key_file,omitempty" yaml:"KeyFile,omitempty" toml:"KeyFile,omitempty"`
	// Letsencrypt enables automatic HTTPS certificates from Let's Encrypt.
	// A host is validated with the tls-alpn-01 challenge, which this TLS listener answers itself:
	// Let's Encrypt connects to port 443 of the host, so that port must reach this server.
	// The HTTP-01 challenge is not served, so no handler on port 80 is involved.
	// If this is enabled but custom CertFile and KeyFile are provided then they are used as a fallback if Let's Encrypt fails,
	// useful for local testing. Nothing is logged when the fallback pair is served.
	Letsencrypt bool `json:"letsencrypt,omitzero" yaml:"Letsencrypt,omitempty" toml:"Letsencrypt,omitempty"`
	// Ngrok provides a tunnel to the local server. If this is set then TLSConfig, Letsencrypt, CertFile and KeyFile options SHOULD be nil.
	// It needs the blank import _ "github.com/kataras/iris/v14/core/server/ngrok" somewhere in the program: without it the server fails to start.
	Ngrok *NgrokOptions `json:"ngrok,omitempty" yaml:"Ngrok,omitempty" toml:"Ngrok,omitempty"`
	// SSHTunnel enables SSH reverse tunnel, if set then TLSConfig, Letsencrypt, CertFile and KeyFile and Ngrok options SHOULD be nil.
	SSHTunnel *SSHTunnelOptions `json:"ssh_tunnel,omitempty" yaml:"SSHTunnel,omitempty" toml:"SSHTunnel,omitempty"`

	// EnableHTTP1Fallback enables HTTP/1 fallback for Let's Encrypt challenges and Ngrok serves HTTP too.
	// EnableHTTP1Fallback bool `json:"enable_http1_fallback,omitempty" yaml:"EnableHTTP1Fallback,omitempty" toml:"EnableHTTP1Fallback,omitempty"`
	// Maybe in a later release as this is not critical, most users will run this behind a balancer or reverse proxy.

	// Timeout server and router(main handler) configuration.
	Timeout *TimeoutOptions `json:"timeout,omitempty" yaml:"Timeout,omitempty" toml:"Timeout,omitempty"`
	// Limits configures the underline net/http server's request limits.
	// Worth a look since Go 1.27: every TLS Iris server offers "h2", so RFC 9218
	// client-driven response prioritization is on by default (see DisableClientPriority),
	// and MaxHeaderValueCount guards the new repeated-header 400 default.
	Limits *LimitsOptions `json:"limits,omitempty" yaml:"Limits,omitempty" toml:"Limits,omitempty"`
	// OnServerCreated is called with the standard http.Server instance right after
	// it's created and the Timeout and Limits options are applied to it, and before
	// it starts serving. It is the escape hatch for any http.Server field Iris does
	// not mirror as an option — including fields added by future Go releases — and
	// for TLS knobs on its TLSConfig such as CurvePreferences (e.g. tls.MLKEM1024;
	// Go's default curve preferences already include the post-quantum hybrid, which
	// is why Iris deliberately leaves CurvePreferences nil).
	// If it returns a non-nil error the listener is closed and the server does not start.
	OnServerCreated func(*http.Server) error `json:"-" yaml:"-" toml:"-"`
	// LogWriter is the logger that the server should use when errors are propagated from the standard library's http.Server instance.
	// If nil, logging is done via the log package's standard logger. This only applies to errors that are not related to the request.
	LogWriter io.Writer `json:"-" yaml:"-" toml:"-"`

	// DisableStartupLog disables the startup log message when server starts.
	DisableStartupLog bool `json:"disable_startup_log,omitzero" yaml:"DisableStartupLog,omitempty" toml:"DisableStartupLog,omitempty"`
	// DisableInterruptHandler disables the interrupt handler, if true then the server will not wait for the interrupt signal to shutdown and cleanup.
	DisableInterruptHandler bool `json:"disable_interrupt_handler,omitzero" yaml:"DisableInterruptHandler,omitempty" toml:"DisableInterruptHandler,omitempty"`

	// hostPolicy is a function that provides a way to check if a host is allowed.
	// It is always built from the Hosts option by setHostPolicy.
	// Every time a new host added on PrependHostPatterns the host policy is re-built,
	// and once more on ListenAndServe, so the Hosts should NOT be modified at serve-time.
	hostPolicy autocert.HostPolicy
}

// func (opts *ServerOptions) Extend(addrHostPort string, rootHostPtr *string, subdomains []string, logWriter io.Writer) {
// 	var rootHost string
// 	if rootHostPtr != nil {
// 	}
// }

// IsTLS checks if the server is configured to use TLS.
func (opts *ServerOptions) IsTLS() bool {
	return opts.TLSConfig != nil || opts.Letsencrypt || opts.Ngrok != nil || opts.SSHTunnel != nil
}

// PrependHostPatterns prepends the host patterns to the server's hosts option.
// Has no effect after the server starts.
func (opts *ServerOptions) PrependHostPatterns(patterns ...string) {
	if len(patterns) == 0 {
		return
	}

	patterns = slices.DeleteFunc(patterns, func(pattern string) bool {
		return pattern == ""
	})

	oldN := len(opts.Hosts)
	for i, pattern := range patterns {
		if !slices.Contains(opts.Hosts, pattern) { // Add the Host to the server's hosts if not already exists.
			opts.Hosts = slices.Insert(opts.Hosts, i, pattern) // each subdomain (and wildcard) has its own host pattern if they don't exist (see Build method).
		}
	}

	if modified := oldN != len(opts.Hosts); modified {
		opts.setHostPolicy()
	}
}

func (opts *ServerOptions) setHostPolicy() {
	opts.hostPolicy = newHostPolicy(opts.Hosts)
}

// NgrokOptions provides configuration for ngrok.
//
// The tunnel code lives in its own package, so a program that sets these options
// also needs the blank import _ "github.com/kataras/iris/v14/core/server/ngrok".
//
// The tunnel's domain is the first entry of the ServerOptions.Hosts, if it exists and it is
// not a loopback address, otherwise ngrok picks a random one.
type NgrokOptions struct {
	// Region is the ngrok region to connect to, e.g. "eu".
	// It is not read by the server at the moment, the ngrok agent selects the region on its own.
	Region    string `json:"region,omitempty" yaml:"Region,omitempty" toml:"Region,omitempty"`
	AuthToken string `json:"auth_token,omitempty" yaml:"AuthToken,omitempty" toml:"AuthToken,omitempty"` // if this is empty ngrok.WithAuthtokenFromEnv() is used.
	// Compression enables compression streaming between ngrok, request body and server.
	// Do not enable it if you are already enabled compression inside Iris application.
	Compression bool              `json:"compression,omitzero" yaml:"Compression,omitempty" toml:"Compression,omitempty"`
	Oauth       map[string]string `json:"oauth,omitempty" yaml:"Oauth,omitempty" toml:"Oauth,omitempty"` // config.WithOAuth("google", config.WithAllowOAuthDomain("example.com")) // key -> value.
}

// SSHTunnelOptions provides configuration for establishing an SSH reverse tunnel
// using services like serveo.net, localhost.run, or any SSH server that supports -R.
//
// Example usage:
//
//	opts := SSHTunnelOptions{
//		RemotePort: 80,                // Remote port to expose (e.g. 80 for HTTP).
//		LocalHost:  "localhost",       // Local host to forward to.
//		LocalPort:  8080,              // Local port to forward to.
//		Domain:     "serveo.net",      // Remote SSH server (e.g. serveo.net, localhost.run).
//		User:       "",                // Optional: SSH username.
//		SSHKeyPath: "",                // Optional: path to SSH private key.
//		ExtraArgs:  []string{},        // Optional: additional SSH arguments.
//	}
//
// The resulting SSH command would look like:
//
//	ssh -T [-i <SSHKeyPath>] -R <RemotePort>:<LocalHost>:<LocalPort> [extra args] [user@]<domain>
type SSHTunnelOptions struct {
	// RemotePort is the port on the remote server to expose (e.g. 80 for HTTP).
	RemotePort int `json:"remote_port" yaml:"RemotePort"`
	// LocalHost is the local host to forward traffic to (e.g. "localhost").
	LocalHost string `json:"local_host" yaml:"LocalHost"`
	// LocalPort is the local port to forward traffic to (e.g. 8080).
	LocalPort int `json:"local_port" yaml:"LocalPort"`
	// Domain is the remote SSH server or custom subdomain/domain (e.g. "serveo.net" or "myapp.serveo.net").
	Domain string `json:"domain,omitempty" yaml:"Domain,omitempty"`
	// User is the SSH username (optional).
	User string `json:"user,omitempty" yaml:"User,omitempty"`
	// SSHKeyPath is the path to the SSH private key file (optional).
	SSHKeyPath string `json:"ssh_key_path,omitempty" yaml:"SSHKeyPath,omitempty"`
	// ExtraArgs allows specifying additional SSH command-line arguments.
	ExtraArgs []string `json:"extra_args,omitempty" yaml:"ExtraArgs,omitempty"`
}

// BuildCommand returns the SSH command string for the SSHTunnelOptions.
// Example output: ssh -T -R 80:localhost:8080 serveo.net.
func (opts *SSHTunnelOptions) BuildCommand() string {
	// For serveo.net and similar, do not add -T if you want the tunnel to actually serve HTTP(S).
	// To avoid terminal glitches, always capture and discard/capture both stdout and stderr, and do not attach stdin.
	args := []string{"ssh", "-T"}

	if opts.SSHKeyPath != "" {
		args = append(args, "-i", opts.SSHKeyPath)
	}

	args = append(args, "-R",
		fmt.Sprintf("%d:%s:%d", opts.RemotePort, opts.LocalHost, opts.LocalPort),
	)

	if len(opts.ExtraArgs) > 0 {
		args = append(args, opts.ExtraArgs...)
	}

	domain := opts.Domain
	if domain == "" {
		domain = "serveo.net"
	}

	if domain == "localhost.run" && opts.User == "" {
		opts.User = "nokey" // for free version.
	}

	if opts.User != "" {
		domain = opts.User + "@" + domain
	}
	args = append(args, domain)

	return strings.Join(args, " ")
}

// Server is a simple HTTP server.
// It's a rich-featured wrapper around the standard http.Server.
// It provides a way to gracefully shutdown the server through the `Shutdown` method
// and to force-close it through the `Close` method.
// Closers of other resources are not registered here, they belong to the Iris Application (see its AddCloser method).
// It provides letsencrypt, ngrok, tls, and other options to configure the server, automatically handles host whitelist.
// It supports socket sharding for linear scaling of server performance on multi-CPU servers and many more.
type Server struct {
	options       *ServerOptions // set on ListenAndServe.
	httpServer    *http.Server
	serverErrorMu sync.RWMutex
	serverError   error // cache the error from ListenAndServe.

	// tlsConfig is the tls.Config the listener was wrapped with (nil on plain HTTP,
	// ngrok and SSH tunnels). Set by newListener, copied to httpServer.TLSConfig by
	// startHTTPServer so that scheme resolvers (netutil.IsTLS) see the truth.
	tlsConfig *tls.Config

	// url is the parsed URL of the address or first host with the listener options (including tlsconfig, letsencrypt and ngrok).
	// It's the base URL of form http://ipaddr:port.
	// It's empty until the server is started. It's written by the serve
	// goroutine and read through the String method by any other goroutine,
	// so access goes through urlMu.
	urlMu sync.RWMutex
	url   string

	onListenerCreated func(ln net.Listener, url *url.URL) error

	// shutdownCh is closed when a graceful shutdown begins. Every request context
	// of this server carries it, so a handler that is streaming can learn that the
	// process is going away while its connection is still writable.
	//
	// It is created in startHTTPServer and closed once, through shutdownOnce.
	shutdownCh   chan struct{}
	shutdownOnce sync.Once

	// shutdowns counts the Shutdown and Close calls that are running and
	// shutdownsDone is closed when the last of them returns. ListenAndServe waits
	// on it, because http.Server.Serve returns when a shutdown begins, not when
	// it ends. Both are guarded by shutdownsMu.
	shutdownsMu   sync.Mutex
	shutdowns     int
	shutdownsDone chan struct{}
}

// signalShutdown closes the shutdown channel, exactly once. It only signals: the
// waiting is http.Server.Shutdown's job.
func (s *Server) signalShutdown() {
	s.shutdownOnce.Do(func() {
		if s.shutdownCh != nil {
			close(s.shutdownCh)
		}
	})
}

// beginShutdown counts a Shutdown or Close call that is about to run and returns
// the function that marks its end.
func (s *Server) beginShutdown() (end func()) {
	s.shutdownsMu.Lock()
	if s.shutdowns == 0 {
		s.shutdownsDone = make(chan struct{})
	}
	s.shutdowns++
	s.shutdownsMu.Unlock()

	return func() {
		s.shutdownsMu.Lock()
		if s.shutdowns--; s.shutdowns == 0 {
			close(s.shutdownsDone)
		}
		s.shutdownsMu.Unlock()
	}
}

// waitShutdown blocks until no Shutdown or Close call is running. It returns at
// once when there is none, which is the case of an http.Server that was shut
// down directly (OnServerCreated hands it out).
func (s *Server) waitShutdown() {
	s.shutdownsMu.Lock()
	done := s.shutdownsDone
	s.shutdownsMu.Unlock()

	if done != nil {
		<-done
	}
}

// NewServer returns a new server instance.
// It does not start the server.
// Use the `ListenAndServe` method to start the server.
// Use the `Shutdown` method to shutdown the server.
// Use the `Close` method to immediately close the server.
// Use the `String` method to get the server's URL.
func NewServer(options ServerOptions) *Server {
	options.Timeout = options.Timeout.Get()
	return &Server{
		options: &options,
	}
}

func (s *Server) allowHost(ctx context.Context, host string) error {
	fn := s.options.hostPolicy
	if fn == nil {
		return nil
	}

	return fn(ctx, host)
}

// OnListenerCreated sets a callback which is called when the server starts and the listener is created.
func (s *Server) OnListenerCreated(fn func(ln net.Listener, url *url.URL) error) *Server {
	if s.onListenerCreated != nil {
		prevFn := s.onListenerCreated
		newFn := fn
		fn = func(ln net.Listener, url *url.URL) error {
			if err := prevFn(ln, url); err != nil {
				return err
			}
			return newFn(ln, url)
		}
	}

	s.onListenerCreated = fn
	return s
}

// ListenAndServe listens on the TCP network address and then calls Serve to handle requests on incoming connections.
//
// It blocks until the server stops. After a Shutdown or a Close it returns nil,
// and only once that call has returned: a graceful shutdown has drained the
// requests in flight, or reached the deadline of its context, by then.
//
// This is the server's own shutdown. iris.Application.Run waits for more: the
// closers that iris.Application.Shutdown runs after it.
func (s *Server) ListenAndServe(handler http.Handler) error {
	err := s.listenAndServe(handler)
	s.setServerError(err) // set the error here, so any Wait call will return.
	if err == http.ErrServerClosed {
		// Serve returns the moment a shutdown begins. A caller that exits when this
		// method returns, a main that ends with it, would cut the requests the
		// shutdown is still draining.
		s.waitShutdown()
		return nil // ignore this error here, it's expected.
	}
	return err
}

func (s *Server) listenAndServe(handler http.Handler) error {
	s.options.setHostPolicy() // build the host policy with the hosts list we have so far.

	listener, err := s.newListener(15 * time.Second)
	if err != nil {
		return err
	}

	if s.onListenerCreated != nil {
		if parsedURL, err := url.Parse(s.String()); err == nil {
			if callbackErr := s.onListenerCreated(listener, parsedURL); callbackErr != nil {
				listener.Close() // close the listener if the callback failed. No need to close anywhere else, http server handles it automatically on later steps.
				return callbackErr
			}
		}
	}

	if !s.options.DisableStartupLog {
		printIPs(nil, s.options.Addr, s.String())
	}

	serverHandler := s.options.Timeout.TimeoutHandler(handler)
	return s.startHTTPServer(serverHandler, listener)
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
// It waits for the requests in flight until ctx is done, then closes the
// connections that are left. ListenAndServe returns once Shutdown has returned.
//
// A handler that stops its own server calls Shutdown on another goroutine, with
// a context of its own: go s.Shutdown(context.Background()), or one that
// carries a deadline. Called in place, Shutdown waits for the request that
// called it:
//
//   - with a context that has no deadline it does not return, and
//     ListenAndServe waits with it. The request's own context is one of
//     those: it ends only when the client goes away.
//   - with a deadline it returns at the deadline, after closing every
//     connection that is left, the caller's own included.
//
// The request's own context is no better on another goroutine: it is canceled
// when the handler returns, which closes the other connections at once.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.free(ctx, true)
}

// Close immediately closes all active connections.
// For a graceful shutdown, use the `Shutdown` method.
func (s *Server) Close() error {
	// immediately(almost) call server's Close.
	ctx, cancelFn := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFn()

	return s.free(ctx, false)
}

// String returns the server's URL.
// It's empty until the server is started.
// It's safe to call it concurrently with ListenAndServe.
func (s *Server) String() string {
	s.urlMu.RLock()
	u := s.url
	s.urlMu.RUnlock()
	return u
}

func (s *Server) setURL(u string) {
	s.urlMu.Lock()
	s.url = u
	s.urlMu.Unlock()
}

// errURLQuerySemicolon is logged by the standard net/http server when
// the request contains a semicolon (;) which, after go1.17, is not used as a key-value separator character.
var errURLQuerySemicolon = errors.New("http: URL query contains semicolon, which is no longer a supported separator; parts of the query may be stripped when parsed; see golang.org/issue/25192")

func (s *Server) startHTTPServer(handler http.Handler, listener net.Listener) error {
	if s.shutdownCh == nil {
		s.shutdownCh = make(chan struct{})
	}
	shutdownCh := (<-chan struct{})(s.shutdownCh)

	logger := newServerLogger(s.options.LogWriter, http.ErrServerClosed.Error(), errURLQuerySemicolon.Error())

	if s.options.Timeout != nil && s.options.Timeout.HandlerTimeout > 0 && logger != nil {
		// Worth one line at startup because the symptom is baffling otherwise:
		// http.TimeoutHandler's response writer implements neither http.Flusher
		// nor http.Hijacker, so under this one option every streaming response
		// buffers until the handler returns and websockets cannot upgrade.
		logger.Printf("Timeout.HandlerTimeout is set (%s): responses cannot stream or hijack under http.TimeoutHandler. "+
			"Use per-route deadlines instead if this server serves Server-Sent Events or websockets.", s.options.Timeout.HandlerTimeout)
	}

	httpServer := &http.Server{
		Addr:     s.options.Addr, // it's not required here as we use its Serve method using a predefined listener.
		Handler:  handler,
		ErrorLog: logger, // if nil, then it uses the standard log package's logger under the hood. Ignore http.ErrServerClosed.
		BaseContext: func(net.Listener) context.Context {
			return withShutdownSignal(context.Background(), shutdownCh)
		},
		// The listener is already wrapped with this tls.Config (see newListener),
		// so Serve does not use it for handshakes; it's set so that netutil.IsTLS
		// and ResolveSchemeFromServer (e.g. router's WithServer) report "https".
		// HTTP/2 is unaffected: Serve configures h2 when a non-nil TLSConfig lists
		// "h2" in its NextProtos (see net/http's shouldConfigureHTTP2ForServe and
		// Issue 15908), and every config built or normalized by newTLSConfig does.
		TLSConfig: s.tlsConfig,
	}
	s.options.Timeout.ApplyToServer(httpServer)
	s.options.Limits.ApplyToServer(httpServer)

	if fn := s.options.OnServerCreated; fn != nil {
		if err := fn(httpServer); err != nil {
			listener.Close() // the standard library never took ownership of the listener, close it here.
			return err
		}
	}

	s.httpServer = httpServer // bind the HTTP Server instance.

	if !s.options.DisableInterruptHandler {
		s.registerInterruptHandler()
	}

	return httpServer.Serve(listener) // start the server, blocking call. No need to Close the listener, it's handled inside the standard library.
}

// registerInterruptHandler wires the server's own shutdown to SIGINT and
// SIGTERM, and hands the signal handler the ShutdownOnInterruptTimeout option:
// it bounds the context that every callback of the signal receives, this
// server's own and the ones registered with iris.RegisterOnInterrupt alike.
func (s *Server) registerInterruptHandler() {
	signalhandler.RegisterWithTimeout(s.shutdownOnInterrupt, s.options.Timeout.ShutdownOnInterruptTimeout)
}

// shutdownOnInterrupt is what the interrupt signal runs: a graceful shutdown that
// may take up to the ShutdownOnInterruptTimeout option. A zero or negative value
// sets no deadline, as the option documents. Given to context.WithTimeout it is a
// context that has expired already, which closes every connection at once.
//
// It builds its own context, so this is the budget of the server's own drain
// whatever the signal handler gives it. The same option bounds the context of
// every other callback of the signal, see registerInterruptHandler.
func (s *Server) shutdownOnInterrupt() error {
	ctx := context.Background()
	if timeout := s.options.Timeout.ShutdownOnInterruptTimeout; timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return s.Shutdown(ctx)
}

// free cleans up the server's resources and unblocks the main routine (which was blocked by ListenAndServe).
func (s *Server) free(ctx context.Context, graceful bool) (joinedErr error) {
	defer s.beginShutdown()() // ListenAndServe returns after this call does.

	// Tell handlers first, and tell them whether the shutdown is graceful or not:
	// a stream that is watching this signal gets the chance to write its terminal
	// frame while the connection is still open, before Shutdown starts waiting
	// for it to finish.
	s.signalShutdown()

	if s.httpServer != nil { // make sure server is not nil (it may be nil when CTRL+C before server is started).
		s.setServerError(http.ErrServerClosed) // set the error to the server error, so we can return it on the Wait method if the end-dev called Close/Shutdown or CTRL+C signal before server started.
		if !graceful {
			// force shutdown on Close.
			joinedErr = s.httpServer.Close()
		} else { // means we want a graceful shutdown.
			joinedErr = s.httpServer.Shutdown(ctx) // call the underline's server Shutdown method instead in order to not call cleanup twice.
			if joinedErr != nil {
				serverCloseErr := s.httpServer.Close() // if graceful shutdown failed, then close the server immediately. Same rule here, use the underline's server Close method.
				if serverCloseErr != nil {
					joinedErr = fmt.Errorf("%v\n%v", joinedErr, serverCloseErr)
				} else {
					joinedErr = nil // if the server is closed successfully, then we don't need to return the Shutdown error.
				}
			}
		}
	}

	return
}

func (s *Server) newListener(connectTimeout time.Duration) (net.Listener, error) {
	if s.options.Ngrok != nil {
		if s.options.TLSConfig != nil || s.options.Letsencrypt || s.options.CertFile != "" || s.options.KeyFile != "" {
			panic("server: Ngrok cannot be used with TLSConfig or Letsencrypt or CertFile or KeyFile options")
		}

		// Create the context, it should not be canceled, as it will close the tunnel. on Shutdown this is closed automatically.
		ctx := context.Background()

		newNgrok := RegisteredNgrokListener()
		if newNgrok == nil {
			return nil, errNgrokNotLinked
		}

		// create a new tunnel listener.
		tunnel, err := newNgrok(ctx, s.options.Ngrok, s.options.Hosts, s.options.Addr)
		if err != nil {
			return nil, err
		}

		// Set the Address and URL (and Hosts[0] if not exist, just in-case).
		tunnelURL := tunnel.URL()
		s.setURL(tunnelURL.String())

		// Set primary host.
		s.options.PrependHostPatterns(tunnelURL.Hostname())

		return tunnel, nil
	}

	if s.options.SSHTunnel != nil {
		if s.options.TLSConfig != nil || s.options.Letsencrypt || s.options.CertFile != "" || s.options.KeyFile != "" || s.options.Ngrok != nil {
			panic("server: SSH Tunnel cannot be used with TLSConfig or Letsencrypt or CertFile or KeyFile or Ngrok options")
		}

		ctx := context.Background()
		listener, tunnelURL, err := newSSHTunnelListener(ctx, s.options.SSHTunnel, s.options.Addr, s.options.LogWriter)
		if err != nil {
			return nil, fmt.Errorf("server: failed to create SSH tunnel listener: %w", err)
		}

		// Set the URL and host for the SSH tunnel.
		if tunnelURL != "" {
			s.setURL(tunnelURL)
			if u, err := url.Parse(tunnelURL); err == nil {
				s.options.PrependHostPatterns(u.Hostname())
			}
		}

		return listener, nil
	}

	// Initialize a new tls config if needed.
	tlsConfig := s.newTLSConfig()
	s.tlsConfig = tlsConfig // retain it for startHTTPServer to expose on the http.Server.

	// Create a context with a timeout, this context is used only to create the network listener.
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	// Create a new listener with the provided options.
	listener, err := newNetworkListener(ctx, s.options.Network, s.options.Addr, s.options.KeepAlive, s.options.SocketSharding, tlsConfig)
	if err != nil {
		return nil, err
	}

	// Set the URL.
	scheme := "http"
	if tlsConfig != nil {
		scheme = "https"
	}
	hostname := ResolveHostname(s.options.Addr, s.options.Hosts...)
	parsedURL, err := url.Parse(scheme + "://" + hostname)
	if err != nil {
		return nil, err
	}
	s.setURL(parsedURL.String())

	// Set the allow host.
	s.options.PrependHostPatterns(hostname)

	return listener, nil
}

// ResolveHostname resolves the hostname from the provided address.
// Rules:
//  1. If the address is empty, it returns "localhost".
//  2. If the address is a port,
//     2.1 If the port is 80 or 443 it returns "localhost"
//     2.2 Otherwise it returns "localhost" + port.
//  3. If the address is a domain, it returns the domain.
//  4. If the address is a port and a domain, it returns the domain.
func ResolveHostname(addr string, hosts ...string) string {
	host := addr
	if len(hosts) > 0 {
		host = hosts[0]
	} else {
		// if host == "" {
		// 	host = "localhost" // if empty, use localhost.
		// }
		// Check if it's passed as ":80" or ":443" or ":http" or ":https and replace it with localhost.
		if host == ":http" {
			host = "localhost"
		} else if host == ":https" {
			host = "localhost"
		} else if host[0] == ':' { // if it's a port.
			if host == ":0" {
				panic("server: cannot resolve hostname for address ':0', please provide a valid, specific address")
			} else {
				if host == ":80" || host == ":443" {
					host = "localhost"
				} else {
					host = "localhost" + host
				}
			}
		}
		// No, let it be 127.0.0.1/192.168.1.100 ...
		// } else if strings.Count(host, ".") > 1 {
		// 	host = "localhost"
		// 	if idx := strings.LastIndex(addr, ":");idx>0 {
		// 		host += host[addr:]
		// 	}
		// }
	}

	return host
}

func (s *Server) newTLSConfig() *tls.Config {
	var (
		config            = s.options.TLSConfig
		enableLetsencrypt = s.options.Letsencrypt
		ngrokConfig       = s.options.Ngrok
		certFile, keyFile = s.options.CertFile, s.options.KeyFile
		hostPatterns      = s.options.Hosts
	)

	if config != nil {
		if enableLetsencrypt || ngrokConfig != nil || certFile != "" || keyFile != "" {
			panic("server: TLSConfig cannot be used with Letsencrypt or Ngrok or CertFile or KeyFile options")
		}

		config = config.Clone() // normalize a copy, the caller's value is never modified.
		if config.NextProtos == nil {
			// Without "h2" here net/http silently serves HTTP/1.1 only, a custom
			// tls.Config used to lose HTTP/2 that way. A non-nil NextProtos is
			// respected as-is (except for "http/1.1", ensured below), and
			// MinVersion is never touched: both are the user's decision.
			config.NextProtos = []string{"h2", "http/1.1"}
		}
	}

	var localCertificate *tls.Certificate
	if certFile != "" && keyFile != "" {
		// if config != nil {
		// 	panic("server: CertFile and KeyFile cannot be used with TLSConfig")
		// }

		if ngrokConfig != nil {
			panic("server: CertFile and KeyFile cannot be used with Ngrok")
		}

		// Read from CertFile and KeyFile.
		// First, try to load them as raw PEM bytes contents.
		certificate, err := tls.X509KeyPair([]byte(certFile), []byte(keyFile))
		if err != nil {
			// If not, then try from file and panic if fail.
			certificate, err = tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				panic(err)
			}
		}

		localCertificate = &certificate
	}

	var letsEncryptConfig *tls.Config
	if enableLetsencrypt {
		// if config != nil {
		// 	panic("server: Letsencrypt cannot be used with TLSConfig")
		// }
		if ngrokConfig != nil {
			panic("server: CertFile and KeyFile cannot be used with Ngrok")
		}

		if len(hostPatterns) == 0 { // try to resolve the hostname from the address.
			hostPatterns = []string{ResolveHostname(s.options.Addr)}
		}
		letsEncryptConfig = s.newLetsencryptTLSConfig()

		if localCertificate != nil {
			// modify the GetCertificate function to fallback to the local certificate.
			letsEncryptGetCertificate := letsEncryptConfig.GetCertificate
			letsEncryptConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				certificate, err := letsEncryptGetCertificate(hello)
				if err != nil {
					// The manager's error ends here: the handshake goes on with the
					// local certificate and the http server's logger gets nothing,
					// so no log line tells which of the two certificates was served.
					certificate = localCertificate
				}
				return certificate, nil
			}
		}
	}

	if letsEncryptConfig != nil {
		config = letsEncryptConfig
	} else if localCertificate != nil {
		// Create a new tls.Config based on the local certificate.
		config = &tls.Config{
			Certificates: []tls.Certificate{*localCertificate},
			ServerName:   ResolveHostname(s.options.Addr, hostPatterns...),
			MinVersion:   tls.VersionTLS13,
			NextProtos:   []string{"h2", "http/1.1"},
		}
	} else if config == nil {
		return nil // no TLS configured, unless ngrok (see later step of the newListener method).
	}

	if !slices.Contains(config.NextProtos, "http/1.1") {
		config.NextProtos = append(config.NextProtos, "http/1.1")
	}
	if config.ServerName == "" {
		config.ServerName = ResolveHostname(s.options.Addr, hostPatterns...)
	}
	return config
}

const cacheDir = "./letsencrypt"

func (s *Server) newLetsencryptTLSConfig() *tls.Config {
	// if len(hostPatterns) == 0 {
	// 	panic("newLetsencryptTLSConfig: hostPatterns should contain at least one element")
	// }

	m := autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		Cache:       autocert.DirCache(cacheDir),
		RenewBefore: 14 * 24 * time.Hour, // renew 14 days before expiration.
		HostPolicy:  s.allowHost,         // we need this to be based on the server's host patterns.
	}

	// The HTTP-01 challenge is not served: its handler would need a listener
	// on port 80, which this server does not open. Maybe in a later step.
	// m.HTTPHandler()

	tlsConfig := &tls.Config{ // tlsConfig := m.TLSConfig()
		GetCertificate: m.GetCertificate,
		MinVersion:     tls.VersionTLS13,
		// The ACME protocol goes last, as in m.TLSConfig(): the HTTP-01 handler above
		// is not served, so tls-alpn-01 is the one challenge this listener can answer,
		// and without its protocol here the CA's handshake fails at negotiation.
		NextProtos: []string{"h2", "http/1.1", acme.ALPNProto},
	}

	return tlsConfig
}

// newNetworkListener creates a new listener with the provided options.
// If TLSConfig is provided, it will create a TLS listener.
func newNetworkListener(ctx context.Context, network, address string, keepAlive time.Duration, socketSharding bool, tlsConfig *tls.Config) (net.Listener, error) {
	cfg := net.ListenConfig{
		KeepAlive: keepAlive,
	}
	if socketSharding {
		cfg.Control = Control
	}

	if network == "" {
		network = "tcp"
	}
	if address == "" {
		if tlsConfig == nil {
			address = ":http"
		} else {
			address = ":https"
		}
	}

	networkListener, err := cfg.Listen(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if tlsConfig == nil {
		return networkListener, nil
	}

	return tls.NewListener(networkListener, tlsConfig), nil
}

// newSSHTunnelListener starts an SSH reverse tunnel using the provided SSHTunnelOptions.
// It fills missing fields from the server configuration and falls back to "localhost" where appropriate.
// Returns a net.Listener that listens on the local forwarded port.
func newSSHTunnelListener(ctx context.Context, opts *SSHTunnelOptions, serverAddr string, logWriter io.Writer) (net.Listener, string, error) {
	// Fill missing fields from serverOpts or fallback to localhost/defaults.
	if opts.LocalHost == "" {
		opts.LocalHost = "localhost"
	}
	if opts.LocalPort == 0 {
		addr := serverAddr
		if addr == "" {
			addr = ":http"
		}
		host, port, err := net.SplitHostPort(addr)
		if err == nil && port != "" {
			opts.LocalPort = parsePort(port)
		} else {
			opts.LocalPort = 8080 // fallback
		}
		if opts.LocalHost == "" && host != "" {
			opts.LocalHost = host
		}
	}
	if opts.RemotePort == 0 {
		opts.RemotePort = 80
	}
	if opts.Domain == "" {
		opts.Domain = "serveo.net"
	}

	// Build the SSH command.
	cmdStr := opts.BuildCommand()
	cmdArgs := strings.Fields(cmdStr)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = nil
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("server: failed to create stdout pipe for SSH tunnel command: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("server: failed to start SSH tunnel: %w", err)
	}

	// Read output for the public URL.
	urlCh := make(chan string, 1)
	go func() {
		defer close(urlCh)   // Ensure the channel is closed when done.
		defer stdout.Close() // Ensure stdout is closed when done.

		const timeout = 30 * time.Second
		start := time.Now()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() && time.Since(start) < timeout {
			line := scanner.Text()
			if line == "" || line == "\n" {
				continue
			}

			if u := extractTunnelURL(line); u != "" {
				urlCh <- u
				return
			}
		}
		// The server still starts without the public URL, so a read error is
		// reported, not returned.
		if err := scanner.Err(); err != nil && logWriter != nil {
			fmt.Fprintf(logWriter, "server: reading SSH tunnel output: %v\n", err)
		}
		urlCh <- ""
	}()

	// read remaining output until the command exits or timeout occurs.
	// go func() {
	// 	for scanner := bufio.NewScanner(stdout); scanner.Scan(); {
	// 		line := scanner.Text()
	// 		if line == "" || line == "\n" {
	// 			continue
	// 		}
	// 		fmt.Println(line)
	// 	}
	// }()

	// Wait for the URL or timeout.
	var tunnelURL string
	select {
	case tunnelURL = <-urlCh:
	case <-ctx.Done():
		return nil, "", fmt.Errorf("server: SSH tunnel command context canceled before URL was received: %w", ctx.Err())
	}

	// Listen on the local forwarded port.
	// listenAddr := fmt.Sprintf("%s:%d", opts.LocalHost, opts.LocalPort)
	if serverAddr == "" {
		serverAddr = fmt.Sprintf("%s:%d", opts.LocalHost, opts.LocalPort)
	}
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		cmd.Process.Kill() // Kill the SSH process if we failed to listen on the local port.
		return nil, "", fmt.Errorf("server: failed to listen on local forwarded port: %w", err)
	}

	cmdListener := &commandListener{
		Listener: listener,
		cmd:      cmd,
	}

	return cmdListener, tunnelURL, nil
}

type commandListener struct {
	net.Listener
	cmd *exec.Cmd
}

// Close kills the SSH tunnel child process and closes the local listener, joining both errors.
// This is required as the ssh process does not exit when the local listener is closed,
// so without it an orphan ssh process would be left running under the hood.
func (w *commandListener) Close() error {
	procErr := w.cmd.Process.Kill()   // Kill the SSH process.
	listenerErr := w.Listener.Close() // Close the listener.

	return errors.Join(procErr, listenerErr)
}

// parsePort safely parses a port string to int, returns 8080 on failure.
func parsePort(port string) int {
	p, err := strconv.Atoi(port)
	if err != nil {
		return 8080
	}
	return p
}

// Helper to extract public URL from SSH tunnel output.
func extractTunnelURL(output string) string {
	re := regexp.MustCompile(`https://[^\s/"]+$`)
	matches := re.FindAllString(output, -1)
	for _, url := range matches {
		if !strings.Contains(url, "localhost") {
			return strings.TrimSpace(url)
		}
	}
	return ""
}
