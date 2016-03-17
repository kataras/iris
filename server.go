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
	handler   http.Handler
	listener  net.Listener
	isRunning bool
	// isSecure true if ListenTLS (https/http2)
	isSecure bool
}

func parseAddr(fullHostOrPort []string) string {
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
	fulladdr := parseAddr(fullHostOrPort)
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
		s.isRunning = true
		s.isSecure = false
	}
	listener.Close()
	//s.listener.Close()
	return err
}

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
		s.isRunning = true
		s.isSecure = true
	}
	//s.listener.Close()
	return err
}

// closeServer is used to close the net.Listener of the standalone http server which has already running via .Listen
func (s *Server) closeServer() {
	if s.isRunning && s.listener != nil {
		s.listener.Close()
		s.isRunning = false
	}
}
