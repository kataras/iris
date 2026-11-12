package server

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
)

// NgrokListenerFunc creates the listener of a ngrok tunnel for the given
// options. The package github.com/kataras/iris/v14/core/server/ngrok provides
// the implementation and registers it with RegisterNgrokListener.
//
// It returns a NgrokTunnelListener and not a plain net.Listener, so the server
// always learns the public URL of the tunnel: it logs that URL at startup and
// adds its host to the allowed hosts.
type NgrokListenerFunc func(ctx context.Context, opts *NgrokOptions, hostPatterns []string, addr string) (NgrokTunnelListener, error)

// NgrokTunnelListener is the listener a NgrokListenerFunc returns. URL is the
// public URL of the tunnel and must not be nil.
type NgrokTunnelListener interface {
	net.Listener
	URL() *url.URL
}

var errNgrokNotLinked = errors.New(`server: ngrok is configured but its listener is not linked in: add the import _ "github.com/kataras/iris/v14/core/server/ngrok"`)

var (
	ngrokListenerMu sync.RWMutex
	ngrokListener   NgrokListenerFunc
)

// RegisterNgrokListener sets the function the server calls to open a ngrok
// tunnel when ServerOptions.Ngrok is set. The package
// github.com/kataras/iris/v14/core/server/ngrok calls it from its init
// function, applications import that package instead of calling this.
// A later call replaces the earlier one. A nil fn removes the registration.
func RegisterNgrokListener(fn NgrokListenerFunc) {
	ngrokListenerMu.Lock()
	ngrokListener = fn
	ngrokListenerMu.Unlock()
}

// RegisteredNgrokListener returns the function set by RegisterNgrokListener,
// or nil when the core/server/ngrok package is not linked in.
func RegisteredNgrokListener() NgrokListenerFunc {
	ngrokListenerMu.RLock()
	defer ngrokListenerMu.RUnlock()
	return ngrokListener
}
