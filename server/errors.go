package server

import "github.com/kataras/iris/errors"

var (
	// ErrServerPortAlreadyUsed returns an error with message: 'Server can't run, port is already used'
	ErrServerPortAlreadyUsed = errors.New("Server can't run, port is already used")
	// ErrServerAlreadyStarted returns an error with message: 'Server is already started and listening'
	ErrServerAlreadyStarted = errors.New("Server is already started and listening")
	// ErrServerOptionsMissing returns an error with message: 'You have to pass iris.ServerOptions'
	ErrServerOptionsMissing = errors.New("You have to pass iris.ServerOptions")
	// ErrServerTLSOptionsMissing returns an error with message: 'You have to set CertFile and KeyFile to iris.ServerOptions before ListenTLS'
	ErrServerTLSOptionsMissing = errors.New("You have to set CertFile and KeyFile to iris.ServerOptions before ListenTLS")
	// ErrServerIsClosed returns an error with message: 'Can't close the server, propably is already closed or never started'
	ErrServerIsClosed = errors.New("Can't close the server, propably is already closed or never started")
	// ErrServerUnknown returns an error with message: 'Unknown reason from Server, please report this as bug!'
	ErrServerUnknown = errors.New("Unknown reason from Server, please report this as bug!")
	// ErrParsedAddr returns an error with message: 'ListeningAddr error, for TCP and UDP, the syntax of ListeningAddr is host:port, like 127.0.0.1:8080.
	// If host is omitted, as in :8080, Listen listens on all available interfaces instead of just the interface with the given host address.
	// See Dial for more details about address syntax'
	ErrParsedAddr = errors.New("ListeningAddr error, for TCP and UDP, the syntax of ListeningAddr is host:port, like 127.0.0.1:8080. If host is omitted, as in :8080, Listen listens on all available interfaces instead of just the interface with the given host address. See Dial for more details about address syntax")
	// ErrServerRemoveUnix returns an error with message: 'Unexpected error when trying to remove unix socket file +filename: +specific error"'
	ErrServerRemoveUnix = errors.New("Unexpected error when trying to remove unix socket file. Addr: %s | Trace: %s")
	// ErrServerChmod returns an error with message: 'Cannot chmod +mode for +host:+specific error
	ErrServerChmod = errors.New("Cannot chmod %#o for %q: %s")
)
