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
}

// DefaultServer returns the default configs for the server
func DefaultServer() Server {
	return Server{DefaultServerAddr, "", "", 0}
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
