package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables.
var runsInCI = os.Getenv("CI") == "true"

func skipTestError(err error) bool {
	if err != nil {
		if runsInCI && strings.Contains(err.Error(), "permission denied") {
			return true
		}
	}

	return false
}

// TestServerListenAndServe tests the Server's ListenAndServe method.
func TestServerListenAndServe(t *testing.T) {
	// Create a ServerOptions instance with the desired settings.
	options := ServerOptions{
		Network: "tcp",
		Addr:    ":http",
	}

	if runsInCI {
		options.Addr = ":8080"
	}

	// Create a new Server instance with the options.
	server := NewServer(options)

	// Define a simple handler that responds with 200 OK.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Start the server in a separate goroutine since it is blocking.
	go func() {
		if err := server.ListenAndServe(handler); err != nil /* err!=http.ErrServerClosed no need */ {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	}()

	// Wait for the server to start and assign a port.
	time.Sleep(3 * time.Second)

	// Make a request to the test server.
	resp, err := http.Get(server.String())
	if err != nil {
		t.Fatalf("Could not make GET request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is 200 OK.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %v", resp.StatusCode)
	}

	// Shutdown the server gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}

// TestServerNewListener tests the Server's newListener method for expected errors and URL parsing.
func TestServerNewListener(t *testing.T) {
	type args struct {
		name           string
		serverOptions  *ServerOptions
		connectTimeout time.Duration
		wantErr        bool
		wantURL        string
	}
	tests := []args{
		{
			name: "Valid HTTP Listener",
			serverOptions: &ServerOptions{
				Network: "tcp",
				Addr:    ":http",
			},
			connectTimeout: 5 * time.Second,
			wantErr:        false,
			wantURL:        "http://localhost",
		},
		{
			name: "Valid HTTPS Listener",
			serverOptions: &ServerOptions{
				Network:   "tcp",
				Addr:      ":https",
				TLSConfig: &tls.Config{},
			},
			connectTimeout: 5 * time.Second,
			wantErr:        false,
			wantURL:        "https://localhost",
		},
		{
			name: "Invalid Network",
			serverOptions: &ServerOptions{
				Network: "invalid",
				Addr:    ":http",
			},
			connectTimeout: 5 * time.Second,
			wantErr:        true,
			wantURL:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				options: tt.serverOptions,
			}

			listener, err := s.newListener(tt.connectTimeout)
			if listener != nil {
				listener.Close()
			}

			if (err != nil) != tt.wantErr {
				if skipTestError(err) {
					return
				}

				t.Errorf("newListener() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if the URL is as expected.
			if got := s.String(); got != tt.wantURL {
				t.Errorf("newListener() got URL = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

// TestNewNetworkListener tests the newNetworkListener function.
func TestNewNetworkListener(t *testing.T) {
	tests := []struct {
		name           string
		network        string
		address        string
		keepAlive      time.Duration
		socketSharding bool
		tlsConfig      *tls.Config
		wantErr        bool
	}{
		{"DefaultNetworkAndAddress", "", "", 0, false, nil, false},
		{"SocketShardingEnabled", "tcp", ":8080", 0, true, nil, false},
		{"WithTLSConfig", "tcp", ":443", 0, false, &tls.Config{}, false},
		{"InvalidNetwork", "invalid", "", 0, false, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with a timeout.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			listener, err := newNetworkListener(ctx, tt.network, tt.address, tt.keepAlive, tt.socketSharding, tt.tlsConfig)
			if listener != nil {
				listener.Close()
			}

			if (err != nil) != tt.wantErr {
				if skipTestError(err) {
					return
				}

				t.Errorf("newNetworkListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestNgrokConfiguredWithoutPackageFailsListen checks that a server configured
// for ngrok does not fall back to a local listener when the
// core/server/ngrok package is not linked in.
func TestNgrokConfiguredWithoutPackageFailsListen(t *testing.T) {
	s := &Server{
		options: &ServerOptions{
			Network: "tcp",
			Addr:    "127.0.0.1:0", // a free port, never :80.
			Ngrok:   &NgrokOptions{},
		},
	}

	listener, err := s.newListener(5 * time.Second)
	if listener != nil {
		listener.Close()
	}

	if err == nil {
		t.Fatal("newListener() returned no error for Ngrok options without a registered ngrok listener")
	}

	const importPath = `_ "github.com/kataras/iris/v14/core/server/ngrok"`
	if !strings.Contains(err.Error(), importPath) {
		t.Fatalf("newListener() error = %q, want it to contain the import %s", err, importPath)
	}
}

// fakeTunnelListener is a local listener that reports a public URL, the way
// the listener of the core/server/ngrok package does.
type fakeTunnelListener struct {
	net.Listener
	publicURL *url.URL
}

func (l *fakeTunnelListener) URL() *url.URL { return l.publicURL }

// TestNgrokRegisteredListenerSetsTheServerURL checks that the server takes its
// URL and its first allowed host from the listener a registered
// NgrokListenerFunc returns. The function's return type is
// NgrokTunnelListener, so a listener without a URL method does not compile.
func TestNgrokRegisteredListenerSetsTheServerURL(t *testing.T) {
	const publicHost = "demo.ngrok.app"

	RegisterNgrokListener(func(context.Context, *NgrokOptions, []string, string) (NgrokTunnelListener, error) {
		local, err := net.Listen("tcp", "127.0.0.1:0") // a free port, never :80.
		if err != nil {
			return nil, err
		}

		return &fakeTunnelListener{Listener: local, publicURL: &url.URL{Scheme: "https", Host: publicHost}}, nil
	})
	t.Cleanup(func() { RegisterNgrokListener(nil) })

	s := &Server{
		options: &ServerOptions{
			Network: "tcp",
			Addr:    "127.0.0.1:0",
			Ngrok:   &NgrokOptions{},
		},
	}

	listener, err := s.newListener(5 * time.Second)
	if err != nil {
		t.Fatalf("newListener() error = %v", err)
	}
	defer listener.Close()

	if got, want := s.String(), "https://"+publicHost; got != want {
		t.Fatalf("server URL = %q, want the tunnel's public URL %q", got, want)
	}

	if !slices.Contains(s.options.Hosts, publicHost) {
		t.Fatalf("server hosts = %q, want them to contain the tunnel's host %q", s.options.Hosts, publicHost)
	}
}
