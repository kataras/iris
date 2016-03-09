package iris

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/pprof" // for profiling purpose
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultDebugPath = "/debug/pprof"
)

// Add paths for debug manually linked (alternative of using DefaultServeMux)
func attachProfiler(r IRouter, debugPath string) {
	if len(debugPath) == 0 {
		debugPath = DefaultDebugPath
	}
	r.HandleFunc(debugPath+"/", HandlerFunc(pprof.Index), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/cmdline", HandlerFunc(pprof.Cmdline), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/profile", HandlerFunc(pprof.Profile), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/symbol", HandlerFunc(pprof.Symbol), HTTPMethods.GET)

	r.HandleFunc(debugPath+"/goroutine", HandlerFunc(pprof.Handler("goroutine")), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/heap", HandlerFunc(pprof.Handler("heap")), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/threadcreate", HandlerFunc(pprof.Handler("threadcreate")), HTTPMethods.GET)
	r.HandleFunc(debugPath+"/pprof/block", HandlerFunc(pprof.Handler("block")), HTTPMethods.GET)

}

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
	// Debug Setting to true you enable the go profiling tool
	// Default Debug path (can changed via server.DebugPath = "/path/to/debug")
	// Memory profile (http://localhost:PORT/debug/pprof/heap)
	// CPU profile (http://localhost:PORT/debug/pprof/profile)
	// Goroutine blocking profile (http://localhost:PORT/debug/pprof/block)
	//
	// Used in the server.go file when starting to the server and initialize the Mux.
	debugEnabled bool
	// DebugPath set this to change the default http paths for the profiler
	debugPath string
	router    IRouter
	listener  net.Listener
	isRunning bool
	// isSecure true if ListenTLS (https/http2)
	isSecure bool
}

// Debug Setting to true you enable the go profiling tool
// Default Debug path (can changed via server.DebugPath = "/path/to/debug")
// Memory profile (http://localhost:PORT/debug/pprof/heap)
// CPU profile (http://localhost:PORT/debug/pprof/profile)
// Goroutine blocking profile (http://localhost:PORT/debug/pprof/block)
//
// Second parameter is the DebugPath set this to change the default http paths for the profiler
//
// Used in the server.go file when starting to the server and initialize the Mux.
func (s *Server) Debug(val bool, customPath ...string) {
	s.debugEnabled = val
	if customPath != nil && len(customPath) == 1 {
		s.debugPath = customPath[0]
	}
}

// Debug Setting to true you enable the go profiling tool
// Default Debug path (can changed via server.DebugPath = "/path/to/debug")
// Memory profile (http://localhost:PORT/debug/pprof/heap)
// CPU profile (http://localhost:PORT/debug/pprof/profile)
// Goroutine blocking profile (http://localhost:PORT/debug/pprof/block)
//
// Second parameter is the DebugPath set this to change the default http paths for the profiler
//
// Used in the server.go file when starting to the server and initialize the Mux.
func Debug(val bool, customPath ...string) {
	DefaultServer.Debug(val)
}

// Errors the meaning of these is that the developer can change the default handlers for http errors
func (s *Server) Errors() *HTTPErrors {
	return s.router.GetErrors()
}

func Errors() *HTTPErrors {
	return DefaultServer.router.GetErrors()
}

func parseAddr(fullHostOrPort interface{}) string {
	addr := "127.0.0.1:8080"
	if fullHostOrPort != nil {

		switch reflect.ValueOf(fullHostOrPort).Interface().(type) {
		case string:
			config := strings.Split(fullHostOrPort.(string), ":")

			if config[0] != "" {
				addr = config[0]
			}

			if len(config) > 1 {
				addr += config[1]
			} else {
				addr += ":8080"
			}
		case int:
			addr = "127.0.0.1:" + strconv.Itoa(fullHostOrPort.(int))
		}
	}
	return addr
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) Listen(fullHostOrPort interface{}) error {
	fulladdr := parseAddr(fullHostOrPort)
	mux := http.NewServeMux() //we use the http's ServeMux for now as the top- middleware of the server, for now.

	mux.Handle("/", s)
	if s.debugEnabled {
		attachProfiler(s.router, s.debugPath)
	}

	//return http.ListenAndServe(s.config.Host+strconv.Itoa(s.config.Port), mux)
	listener, err := net.Listen("tcp", fulladdr)

	if err != nil {
		//panic("Cannot run the server [problem with tcp listener on host:port]: " + fulladdr + " err:" + err.Error())
		return err
	}
	s.listener = &tcpKeepAliveListener{listener.(*net.TCPListener)}
	err = http.Serve(s.listener, mux)
	if err == nil {
		s.isRunning = true
		s.isSecure = false
	}
	listener.Close()
	//s.listener.Close()
	return err
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) ListenTLS(fullHostOrPort interface{}, certFile, keyFile string) error {
	var err error
	fulladdr := parseAddr(fullHostOrPort)
	httpServer := http.Server{
		Addr:    fulladdr,
		Handler: s,
	}
	if s.debugEnabled {
		attachProfiler(s.router, s.debugPath)
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

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func Listen(fullHostOrPort interface{}) error {
	return DefaultServer.Listen(fullHostOrPort)
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func ListenTLS(fullHostOrPort interface{}, certFile, keyFile string) error {
	return DefaultServer.ListenTLS(fullHostOrPort, certFile, keyFile)
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//I thing it's better to keep the main serve to the server, this is the meaning of the Server struct .so delete: s.router.ServeHTTP(res, req)

	//the Server is HTTPS/HTTP2 but the request is 'http'
	//redirect the url to the https version
	//doesn't work because of line 1406 of the net/http/server.go
	//the tls http.Serve is handle this via low-level connection, it logs an error on the console and returns
	//and doesn't continue here to the ServeHTTP

	///TODO: I must find another way to do something like that
	/*if s.isSecure && req.TLS == nil {
		//req.URL.Scheme = "https://"
		http.Redirect(res, req, "https://"+req.Host+req.URL.Path, http.StatusOK)
		return
	}*/

	s.router.ServeHTTP(res, req)
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	DefaultServer.ServeHTTP(res, req)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func (s *Server) Close() {
	if s.isRunning && s.listener != nil {
		s.listener.Close()
		s.isRunning = false
	}
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { DefaultServer.Close() }

///////////////////////////////////////////////////////////////////////////////////////////
//expose some methods as public on the Server struct, most of them are from server's router
///////////////////////////////////////////////////////////////////////////////////////////

// Get registers a route for the Get http method
func (s *Server) Get(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.GET)
}

// Get registers a route for the Get http method
func Get(path string, handler interface{}) *Route {
	return DefaultServer.Get(path, handler)
}

// Post registers a route for the Post http method
func (s *Server) Post(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.POST)
}

// Post registers a route for the Post http method
func Post(path string, handler interface{}) *Route {
	return DefaultServer.Post(path, handler)
}

// Put registers a route for the Put http method
func (s *Server) Put(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.PUT)
}

// Put registers a route for the Put http method
func Put(path string, handler interface{}) *Route {
	return DefaultServer.Put(path, handler)
}

// Delete registers a route for the Delete http method
func (s *Server) Delete(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.DELETE)
}

// Delete registers a route for the Delete http method
func Delete(path string, handler interface{}) *Route {
	return DefaultServer.Delete(path, handler)
}

// Connect registers a route for the Connect http method
func (s *Server) Connect(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.CONNECT)
}

// Connect registers a route for the Connect http method
func Connect(path string, handler interface{}) *Route {
	return DefaultServer.Connect(path, handler)
}

// Head registers a route for the Head http method
func (s *Server) Head(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.HEAD)
}

// Head registers a route for the Head http method
func Head(path string, handler interface{}) *Route {
	return DefaultServer.Head(path, handler)
}

// Options registers a route for the Options http method
func (s *Server) Options(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.OPTIONS)
}

// Options registers a route for the Options http method
func Options(path string, handler interface{}) *Route {
	return DefaultServer.Options(path, handler)
}

// Patch registers a route for the Patch http method
func (s *Server) Patch(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.PATCH)
}

// Patch registers a route for the Patch http method
func Patch(path string, handler interface{}) *Route {
	return DefaultServer.Patch(path, handler)
}

// Trace registers a route for the Trace http method
func (s *Server) Trace(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), HTTPMethods.TRACE)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handler interface{}) *Route {
	return DefaultServer.Trace(path, handler)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (s *Server) Any(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler), "")
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handler interface{}) *Route {
	return DefaultServer.Any(path, handler)
}

// Handle registers a route to the server's router, pass a struct -implements iris.Annotated as parameter
// Or pass just a http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
func (s *Server) Handle(params ...interface{}) *Route {
	paramsLen := len(params)
	if paramsLen == 0 {
		panic("No arguments given to the Handle function, please refer to docs")
	}

	if reflect.TypeOf(params[0]).Kind() == reflect.String {
		//means first parameter is the path, wich means it is a simple path with handler -> HandleFunc and method
		// means: http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
		return s.HandleFunc(params[0].(string), convertToHandler(params[1]), params[2].(string))
	} else {
		//means it's a struct which implements the iris.Annotated and have a Handle func inside it -> handleAnnotated
		r, err := s.HandleAnnotated(params[0].(Annotated))
		if err != nil {
			panic(err.Error())
		}
		return r
	}
}

// Handle registers a route to the server's router, pass a struct -implements iris.Annotated as parameter
// Or pass just a http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
func Handle(params ...interface{}) *Route {
	return DefaultServer.Handle(params...)
}
