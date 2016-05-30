package config

import (
	"os"

	"github.com/imdario/mergo"
)

const (
	// DefaultServerAddr the default server addr
	DefaultServerAddr = ":8080"
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
