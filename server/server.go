// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package server

import (
	"net"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
)

const (
	// DefaultServerAddr the default server addr
	DefaultServerAddr = ":8080"
	// DefaultServerName the response header of the 'Server' value when writes to the client
	DefaultServerName = "iris"
)

// Server is the IServer's implementation, holds the fasthttp's Server, a net.Listener, the ServerOptions, and the handler
// handler is registed at the Station/Iris level
type Server struct {
	*fasthttp.Server
	listener net.Listener
	Config   Config
	started  bool
	tls      bool
	handler  fasthttp.RequestHandler
}

// New returns a pointer to a Server object, and set it's options if any,  nothing more
func New(opt ...Config) *Server {
	s := &Server{Server: &fasthttp.Server{Name: DefaultServerName}}
	if opt != nil && len(opt) > 0 {
		s.Config = opt[0]
	}
	s.Config.ListeningAddr = parseAddr(s.Config.ListeningAddr)

	return s
}

// DefaultConfig returns the default options for the server
func DefaultConfig() Config {
	return Config{DefaultServerAddr, "", "", 0}
}

// DefaultServerSecureOptions does nothing now
///TODO: make it to generate self-signed certificate: CertFile,KeyFile options for ListenTLS
func DefaultServerSecureOptions() Config { return DefaultConfig() }

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
	s.listener, err = net.Listen("tcp", s.Config.ListeningAddr)

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

	s.listener, err = net.Listen("tcp", s.Config.ListeningAddr)

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
		return s.listener.Close()
	}
	return nil
}

// parseAddr gets a slice of string and returns the address of which the Iris' server can listen
func parseAddr(fullHostOrPort ...string) string {

	if len(fullHostOrPort) > 1 {
		fullHostOrPort = fullHostOrPort[0:1]
	}
	addr := DefaultServerAddr // default address
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
