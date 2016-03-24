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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// this is excatcly a copy of the Go Source (net/http) server.go
// and it used only on non-tsl server (HTTP/1.1)
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// Server is the container of the tcp listener used to start an http server,
//
// it holds it's router and it's config,
// also a property named isRunning which can be used to see if the server is already running or not.
//
// Server's New() located at the iris.go file
type Server struct {
	// the handler which comes from the station which comes from the router.
	handler       http.Handler
	listener      net.Listener
	IsRunning     bool
	ListeningAddr string
	// IsSecure true if ListenTLS (https/http2)
	IsSecure          bool
	CertFile, KeyFile string
}

func ParseAddr(fullHostOrPort []string) string {
	//means only port is given
	hlen := len(fullHostOrPort)

	//wrong parameters
	if hlen > 1 {
		panic("Iris: Max parameters length is 2, pass a host:port or port")
	}
	addr := ":8080" // default address
	// if nothing passed, then use environment's port (if any) or just :8080
	if hlen == 0 {
		if envPort := os.Getenv("PORT"); len(envPort) > 0 {
			addr = ":" + envPort
		}

	} else if hlen == 1 {
		addr = fullHostOrPort[0]
		if strings.IndexRune(addr, ':') == -1 {
			//: doesn't found on the given address, so maybe it's only a port
			addr = ":" + addr
		}
	}

	return addr
}

// listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) listen(fullHostOrPort ...string) error {
	fulladdr := ParseAddr(fullHostOrPort)
	//mux := http.NewServeMux() //we use the http's ServeMux for now as the top- middleware of the server, for now.

	//mux.Handle("/", s.handler)

	//return http.ListenAndServe(s.config.Host+strconv.Itoa(s.config.Port), mux)
	listener, err := net.Listen("tcp", fulladdr)

	if err != nil {
		//panic("Cannot run the server [problem with tcp listener on host:port]: " + fulladdr + " err:" + err.Error())
		return err
	}
	s.listener = &tcpKeepAliveListener{listener.(*net.TCPListener)}
	err = http.Serve(s.listener, s.handler)

	if err == nil {
		s.ListeningAddr = fulladdr
		s.IsRunning = true
		s.IsSecure = false
		s.CertFile = ""
		s.KeyFile = ""
	}
	listener.Close()
	//s.listener.Close()
	return err
}

/*///TODO: MANUAL HOOK THE LISTENER, BECAUSE STANDAR NET/HTTP PACKAGE MAKES SO MANY CHECKS MAYBE I CAN DO IT WORKS EVEN FASTER
func (s *Server) accept(l net.Listener) error {
	for {
		rw, e := l.Accept()
		if e != nil {
			return e
		}

		tempDelay = 0
		c := s.newConn(rw)
		c.setState(c.rwc, StateNew) // before Serve can return
		go c.serve()
	}
}
*/
// listenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) listenTLS(fulladdr string, certFile, keyFile string) error {
	var err error
	httpServer := http.Server{
		Addr:    fulladdr,
		Handler: s.handler,
	}

	config := &tls.Config{}

	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert && certFile != "" && keyFile != "" {
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	}
	httpServer.TLSConfig = config
	s.listener, err = tls.Listen("tcp", fulladdr, httpServer.TLSConfig)
	if err != nil {
		panic("Cannot run the server [problem with tcp listener on host:port]: " + fulladdr + " err:" + err.Error())
	}

	err = httpServer.Serve(s.listener)

	if err == nil {
		s.IsRunning = true
		s.IsSecure = true
		s.ListeningAddr = fulladdr
		s.CertFile = certFile
		s.KeyFile = keyFile
	}
	//s.listener.Close()
	return err
}

// closeServer is used to close the net.Listener of the standalone http server which has already running via .Listen
func (s *Server) closeServer() {
	if s.IsRunning && s.listener != nil {
		s.listener.Close()
		s.IsRunning = false
	}
}
