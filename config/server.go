package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	"github.com/valyala/fasthttp"
)

// Default values for base Server conf
const (
	// DefaultServerHostname returns the default hostname which is 0.0.0.0
	DefaultServerHostname = "0.0.0.0"
	// DefaultServerPort returns the default port which is 8080
	DefaultServerPort = 8080
	// DefaultMaxRequestBodySize is 8MB
	DefaultMaxRequestBodySize = 2 * fasthttp.DefaultMaxRequestBodySize

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is 8MB
	DefaultReadBufferSize = 8096

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is 8MB
	DefaultWriteBufferSize = 8096

	// DefaultServerName the response header of the 'Server' value when writes to the client
	DefaultServerName = "iris"
)

var (
	// DefaultServerAddr the default server addr which is: 0.0.0.0:8080
	DefaultServerAddr = DefaultServerHostname + ":" + strconv.Itoa(DefaultServerPort)
)

// Server used inside server for listening
type Server struct {
	// ListenningAddr the addr that server listens to
	ListeningAddr string
	CertFile      string
	KeyFile       string
	// Mode this is for unix only
	Mode os.FileMode
	// MaxRequestBodySize Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// By default request body size is 8MB.
	MaxRequestBodySize int

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	ReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	WriteBufferSize int

	// RedirectTo, defaults to empty, set it in order to override the station's handler and redirect all requests to this address which is of form(HOST:PORT or :PORT)
	//
	// NOTE: the http status is 'StatusMovedPermanently', means one-time-redirect(the browser remembers the new addr and goes to the new address without need to request something from this server
	// which means that if you want to change this address you have to clear your browser's cache in order this to be able to change to the new addr.
	//
	// example: https://github.com/iris-contrib/examples/tree/master/multiserver_listening2
	RedirectTo string
	// Virtual If this server is not really listens to a real host, it mostly used in order to achieve testing without system modifications
	Virtual bool
	// VListeningAddr, can be used for both virtual = true or false,
	// if it's setted to not empty, then the server's Host() will return this addr instead of the ListeningAddr.
	// server's Host() is used inside global template helper funcs
	// set it when you are sure you know what it does.
	//
	// Default is empty ""
	VListeningAddr string
	// Name the server's name, defaults to "iris".
	// You're free to change it, but I will trust you to don't, this is the only setting whose somebody, like me, can see if iris web framework is used
	Name string
}

// ServerParseAddr parses the listening addr and returns this
func ServerParseAddr(listeningAddr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := listeningAddr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		}

		if a == "" {
			a = DefaultServerAddr
		}

	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
		a = DefaultServerHostname + a
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx < 0 {
		// missing port part, add it
		a = a + ":80"
	}

	return a
}

// DefaultServer returns the default configs for the server
func DefaultServer() Server {
	return Server{
		ListeningAddr:      DefaultServerAddr,
		Name:               DefaultServerName,
		MaxRequestBodySize: DefaultMaxRequestBodySize,
		ReadBufferSize:     DefaultReadBufferSize,
		WriteBufferSize:    DefaultWriteBufferSize,
		RedirectTo:         "",
		Virtual:            false,
		VListeningAddr:     "",
	}
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

// MergeSingle merges the default with the given config and returns the result
func (c Server) MergeSingle(cfg Server) (config Server) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
