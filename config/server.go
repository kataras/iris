package config

import (
	"os"
	"strconv"

	"github.com/imdario/mergo"
)

const (
	// DefaultServerHostname returns the default hostname which is 127.0.0.1
	DefaultServerHostname = "127.0.0.1"
	// DefaultServerPort returns the default port which is 8080
	DefaultServerPort = 8080
)

var (
	// DefaultServerAddr the default server addr which is: 127.0.0.1:8080
	DefaultServerAddr = DefaultServerHostname + ":" + strconv.Itoa(DefaultServerPort)
)

// ServerName the response header of the 'Server' value when writes to the client
const ServerName = "iris"

// Server used inside server for listening
type Server struct {
	// ListenningAddr the addr that server listens to
	ListeningAddr string
	CertFile      string
	KeyFile       string
	// Mode this is for unix only
	Mode os.FileMode
	// RedirectTo, defaults to empty, set it in order to override the station's handler and redirect all requests to this address which is of form(HOST:PORT or :PORT)
	//
	// NOTE: the http status is 'StatusMovedPermanently', means one-time-redirect(the browser remembers the new addr and goes to the new address without need to request something from this server
	// which means that if you want to change this address you have to clear your browser's cache in order this to be able to change to the new addr.
	//
	// example: https://github.com/iris-contrib/examples/tree/master/multiserver_listening2
	RedirectTo string
	// Virtual If this server is not really listens to a real host, it mostly used in order to achieve testing without system modifications
	Virtual bool
}

// DefaultServer returns the default configs for the server
func DefaultServer() Server {
	return Server{ListeningAddr: DefaultServerAddr}
}

// Merge merges the default with the given config and returns the result
func (c Server) Merge(cfg []Server) (config Server) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}
