package iris

import (
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
	// Errors the meaning of these is that the developer can change the default handlers for http errors
	Errors *HTTPErrors

	// Debug Setting to true you enable the go profiling tool
	// Default Debug path (can changed via server.DebugPath = "/path/to/debug")
	// Memory profile (http://localhost:PORT/debug/pprof/heap)
	// CPU profile (http://localhost:PORT/debug/pprof/profile)
	// Goroutine blocking profile (http://localhost:PORT/debug/pprof/block)
	//
	// Used in the server.go file when starting to the server and initialize the Mux.
	debugEnabled bool
	// DebugPath set this to change the default http paths for the profiler
	debugPath   string
	router      IRouter
	listenerTCP *tcpKeepAliveListener
	isRunning   bool
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

// Starts the http server ,tcp listening to the config's host and port
func (s *Server) start(fulladdr string) error {
	mux := http.NewServeMux() //we use the http's ServeMux for now as the top- middleware of the server, for now.

	mux.Handle("/", s)
	if s.debugEnabled {
		attachProfiler(s.router, s.debugPath)
	}

	//return http.ListenAndServe(s.config.Host+strconv.Itoa(s.config.Port), mux)

	listener, err := net.Listen("tcp", fulladdr)

	if err != nil {
		panic("Cannot run the server [problem with tcp listener on host:port]: " + fulladdr)
	}

	s.listenerTCP = &tcpKeepAliveListener{listener.(*net.TCPListener)} //we need this because I think that we have to 'have' a Close() method on the server instance
	//defer s.listenerTCP.Close()
	err = http.Serve(s.listenerTCP, mux)

	s.isRunning = true
	s.listenerTCP.Close()
	return err
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) Listen(fullHostOrPort interface{}) error {
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

	return s.start(addr)

}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func Listen(fullHostOrPort interface{}) error {

	return DefaultServer.Listen(fullHostOrPort)
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//I thing it's better to keep the main serve to the server, this is the meaning of the Server struct .so delete: s.router.ServeHTTP(res, req)
	if route := s.router.Find(req); route != nil {
		route.ServeHTTP(res, req)
	} else {
		s.Errors.NotFound(res)
	}

	//route, errCode := s.router.find(req)
	/*	if errCode == http.StatusOK {
			route.ServeHTTP(res, req)
		} else {
			//get the handler for this error
			errHandler := s.Errors.getByCode(errCode)
			if errHandler == nil {
				//if not a handler for this error exists, then just:
				http.Error(res, "An unexcpecting error occurs ("+strconv.Itoa(errCode)+")", errCode)
			} else {
				errHandler.handler.ServeHTTP(res, req)
			}
		}*/
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	DefaultServer.ServeHTTP(res, req)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func (s *Server) Close() {
	if s.isRunning && s.listenerTCP != nil {
		s.listenerTCP.Close()
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
