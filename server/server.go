package server

import (
	"net"
	"os"
	"strings"

	"github.com/kataras/iris/config"
	"github.com/valyala/fasthttp"
)

// Server is the IServer's implementation, holds the fasthttp's Server, a net.Listener, the ServerOptions, and the handler
// handler is registed at the Station/Iris level
type Server struct {
	*fasthttp.Server
	listener net.Listener
	Config   config.Server
	started  bool
	tls      bool
	handler  fasthttp.RequestHandler
}

// New returns a pointer to a Server object, and set it's options if any,  nothing more
func New(cfg ...config.Server) *Server {
	c := config.DefaultServer().Merge(cfg)
	s := &Server{Server: &fasthttp.Server{Name: config.ServerName}, Config: c}

	s.Config.ListeningAddr = parseAddr(s.Config.ListeningAddr)

	return s
}

// SetHandler sets the handler in order to listen on new requests, this is done at the Station/Iris level
func (s *Server) SetHandler(h fasthttp.RequestHandler) {
	s.handler = h
	if s.Server != nil {
		s.Server.Handler = s.handler
	}
}

// Handler returns the fasthttp.RequestHandler which is registed to the Server
func (s *Server) Handler() fasthttp.RequestHandler {
	return s.handler
}

// IsListening returns true if server is listening/started, otherwise false
func (s *Server) IsListening() bool {
	return s.started
}

// IsSecure returns true if server uses TLS, otherwise false
func (s *Server) IsSecure() bool {
	return s.tls
}

// Listener returns the net.Listener which this server (is) listening to
func (s *Server) Listener() net.Listener {
	return s.listener
}

// Host returns the Listener().Addr().String(), if server is not listening it returns the config.ListeningAddr
func (s *Server) Host() (host string) {
	if s.IsListening() {
		return s.Listener().Addr().String()
	} else {
		return s.Config.ListeningAddr
	}
}

// VirtualHost returns the s.Config.ListeningAddr, if host provided else returns the Listener's (Host())
//
// Note: currently this is used only on iris/route.ParseURI.
//
func (s *Server) VirtualHost() (host string) {
	// we always have at least the :PORT because  of parseAddr, so we just
	// check if we have anything before PORT
	a := s.Config.ListeningAddr
	if len(a[0:strings.IndexByte(a, ':')]) > 0 {
		return a
	} else {
		return s.Host()
	}
}

// Hostname returns the hostname part only, if host == 0.0.0.0:8080 it will return the 0.0.0.0
// if server is not listening it returns the config.ListeningAddr's hostname part
func (s *Server) Hostname() (hostname string) {
	if s.IsListening() {
		fullhost := s.Listener().Addr().String()
		hostname = fullhost[0:strings.IndexByte(fullhost, ':')] // no the port
	} else {
		fullhost := s.Config.ListeningAddr
		if idx := strings.IndexByte(fullhost, ':'); idx > 1 { // at least after second char
			hostname = hostname[0:idx]
		} else {
			hostname = "0.0.0.0"
		}

	}
	return
}

//Serve just serves a listener, it is a blocking action, plugin.PostListen is not fired here.
func (s *Server) Serve(l net.Listener) error {
	s.listener = l
	return s.Server.Serve(l)
}

// listen starts the process of listening to the new requests
func (s *Server) listen() (err error) {

	if s.started {
		err = ErrServerAlreadyStarted.Return()
		return
	}
	s.listener, err = net.Listen("tcp4", s.Config.ListeningAddr)

	if err != nil {
		err = ErrServerPortAlreadyUsed.Return()
		return
	}

	//Non-block way here because I want the plugin's PostListen ability...
	go s.Server.Serve(s.listener)

	s.started = true
	s.tls = false

	return
}

// listenTLS starts the process of listening to the new requests using TLS, keyfile and certfile are given before this method fires
func (s *Server) listenTLS() (err error) {

	if s.started {
		err = ErrServerAlreadyStarted.Return()
		return
	}

	if s.Config.CertFile == "" || s.Config.KeyFile == "" {
		err = ErrServerTLSOptionsMissing.Return()
		return
	}

	s.listener, err = net.Listen("tcp4", s.Config.ListeningAddr)

	if err != nil {
		err = ErrServerPortAlreadyUsed.Return()
		return
	}

	go s.Server.ServeTLS(s.listener, s.Config.CertFile, s.Config.KeyFile)

	s.started = true
	s.tls = true

	return
}

// listenUnix  starts the process of listening to the new requests using a 'socket file', this works only on unix
func (s *Server) listenUnix() (err error) {

	if s.started {
		err = ErrServerAlreadyStarted.Return()
		return
	}

	mode := s.Config.Mode

	//this code is from fasthttp ListenAndServeUNIX, I extracted it because we need the tcp.Listener
	if errOs := os.Remove(s.Config.ListeningAddr); errOs != nil && !os.IsNotExist(errOs) {
		err = ErrServerRemoveUnix.Format(s.Config.ListeningAddr, errOs.Error())
		return
	}
	//
	s.listener, err = net.Listen("unix", s.Config.ListeningAddr)

	if err != nil {
		err = ErrServerPortAlreadyUsed.Return()
		return
	}

	if err = os.Chmod(s.Config.ListeningAddr, mode); err != nil {
		err = ErrServerChmod.Format(mode, s.Config.ListeningAddr, err.Error())
		return
	}

	s.Server.Handler = s.handler
	go s.Server.Serve(s.listener)

	s.started = true
	s.tls = false

	return

}

// OpenServer opens/starts/runs/listens (to) the server, listenTLS if Cert && Key is registed, listenUnix if Mode is registed, otherwise listen
// instead of return an error this is panics on any server's error
func (s *Server) OpenServer() (err error) {
	if s.Config.CertFile != "" && s.Config.KeyFile != "" {
		err = s.listenTLS()
	} else if s.Config.Mode > 0 {
		err = s.listenUnix()
	} else {
		err = s.listen()
	}

	return
}

// CloseServer closes the server
func (s *Server) CloseServer() error {

	if !s.started {
		return ErrServerIsClosed.Return()
	}

	if s.listener != nil {
		s.started = false
		return s.listener.Close()
	}
	return nil
}

// parseAddr gets a slice of string and returns the address of which the Iris' server can listen
func parseAddr(fullHostOrPort ...string) string {

	if len(fullHostOrPort) > 1 {
		fullHostOrPort = fullHostOrPort[0:1]
	}
	addr := config.DefaultServerAddr // default address
	// if nothing passed, then use environment's port (if any) or just :8080
	if len(fullHostOrPort) == 0 {
		if envPort := os.Getenv("PORT"); len(envPort) > 0 {
			addr = ":" + envPort
		}

	} else if len(fullHostOrPort) == 1 {
		addr = fullHostOrPort[0]
		if strings.IndexRune(addr, ':') == -1 {
			//: doesn't found on the given address, so maybe it's only a port
			addr = ":" + addr
		}
	}

	return addr
}
