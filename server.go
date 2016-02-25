package iris

import (
	"net"
	"net/http"
	"net/http/pprof" // for testing purpose
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	DebugPath = "/debug/pprof"
)

// Add paths for debug manually linked (alternative of using DefaultServeMux)
func attachProfiler(theMux *http.ServeMux) {
	theMux.HandleFunc(DebugPath+"/", pprof.Index)
	theMux.HandleFunc(DebugPath+"/cmdline", pprof.Cmdline)
	theMux.HandleFunc(DebugPath+"/profile", pprof.Profile)
	theMux.HandleFunc(DebugPath+"/symbol", pprof.Symbol)

	theMux.Handle(DebugPath+"/goroutine", pprof.Handler("goroutine"))
	theMux.Handle(DebugPath+"/heap", pprof.Handler("heap"))
	theMux.Handle(DebugPath+"/threadcreate", pprof.Handler("threadcreate"))
	theMux.Handle(DebugPath+"/pprof/block", pprof.Handler("block"))
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
	Errors      *HTTPErrors
	config      *ServerConfig
	router      *Router
	listenerTCP *tcpKeepAliveListener
	isRunning   bool
}

// Host usage is to set the "host" of the server
func (s *Server) Host(host string) *Server {
	s.config.Host = host
	return s
}

// Port usage is to set the port of the server
func (s *Server) Port(port int) *Server {
	s.config.Port = port
	return s
}

// Starts the http server ,tcp listening to the config's host and port
func (s *Server) start() error {
	//mux := http.NewServeMux()
	//mux := http.DefaultServeMux
	//if !Debug {
	//	mux = http.NewServeMux()
	//}
	mux := http.NewServeMux()

	mux.Handle("/", s)

	if Debug {
		attachProfiler(mux)
	}

	//return http.ListenAndServe(s.config.Host+strconv.Itoa(s.config.Port), mux)
	fullAddr := s.config.Host + ":" + strconv.Itoa(s.config.Port)
	listener, err := net.Listen("tcp", fullAddr)

	if err != nil {
		panic("Cannot run the server [problem with tcp listener on host:port]: " + fullAddr)
	}

	s.listenerTCP = &tcpKeepAliveListener{listener.(*net.TCPListener)} //we need this because I think that we have to 'have' a Close() method on the server instance
	defer s.listenerTCP.Close()
	err = http.Serve(s.listenerTCP, mux)

	s.isRunning = true
	return err
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func (s *Server) Close() {
	if s.isRunning && s.listenerTCP != nil {
		s.listenerTCP.Close()
		s.isRunning = false
	}
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Server) Listen(fullHostOrPort interface{}) error {

	switch reflect.ValueOf(fullHostOrPort).Interface().(type) {
	case string:
		config := strings.Split(fullHostOrPort.(string), ":")

		if strings.TrimSpace(config[0]) != "" {
			s.config.Host = config[0]
		}

		if len(config) > 1 {
			s.config.Port, _ = strconv.Atoi(config[1])
		}
	default:
		s.config.Port = fullHostOrPort.(int)
	}
	return s.start()

}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//I thing it's better to keep the main serve to the server, this is the meaning of the Server struct .so delete: s.router.ServeHTTP(res, req)
	route, errCode := s.router.find(req)

	if errCode == http.StatusOK {
		route.ServeHTTP(res, req)
	} else {
		//get the handler for this error
		errHandler := s.Errors.errorHanders[errCode]

		if errHandler != nil {
			errHandler.ServeHTTP(res, req)
		} else {
			//if not a handler for this error exists, then just:
			http.Error(res, "An unexcpecting error occurs ("+strconv.Itoa(errCode)+")", errCode)
		}
	}

}

///////////////////////////////////////////////////////////////////////////////////////////
//expose some methods as public on the Server struct, most of them are from server's router
///////////////////////////////////////////////////////////////////////////////////////////

/* global middleware */

// Use registers a a custom handler, with next, as a global middleware
func (s *Server) Use(handler MiddlewareHandler) *Server {
	s.router.Use(handler)
	return s
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func (s *Server) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	s.router.UseFunc(handlerFunc)
	return s
}

// UseHandler registers a simple http.Handler as global middleware
func (s *Server) UseHandler(handler http.Handler) *Server {
	s.router.UseHandler(handler)
	return s
}

// I KEEP THIS FOR FUTURE USE MAYBE.
// Handle registers a handler in the router and returns a Route
// Parameters (path string, handler HTTPHandler (or middlewares and the last is the handler ...HTTPHandler) OR any struct implements the custom Iris Handler interface.)
// This doesnt provide a way to predefine the httpmethod, we use the .Methods() to set methods after this Handle method returns the route
/*func (s *Server) Handle(params ...interface{}) *Route {
	//means custom struct for handler
	var route *Route
	var err error
	argsLen := len(params)
	if argsLen == 1 {
		route, err = s.RegisterHandler(params[0].(Handler))
	} else if argsLen >= 2 {
		//means that we have a path string and the handler or handlers (as slice) OR just multiple arguments if this .Handle called directly.
		var handlers []interface{}
		// CARE IF the .Handle called directly the second parameter maybe not be a standalone slice
		// it is always slice if get called by iris.Get,iris.Post and e.t.c but not if it's called directly, so we check it first.
		//because that I removed the HTTPHandler and replace it with just an interface{} we had runtime errors on conversation between the []HTTPHandler and []interface{} when run the tests (which are call the .Handle directly)
		// or I can make it no visible to the public ?
		if reflect.TypeOf(params[1]).Kind() == reflect.Slice {
			handlers = params[1].([]interface{})
		} else {
			//called directly so maybe have more than two params and the second parameter(params[1]) is not a slice
			handlers = params[1:argsLen]
		}
		handlersLen := len(handlers)

		if handlersLen == 1 {
			//params[1].([]HTTPHandler)[0] (or handlers[0] )because the second parameter will be a slice using the handlers... at the Get, Post e.t.c... so we have to take the first of this slice
			route = s.router.Handle(params[0].(string), handlers[0].(HTTPHandler))
		} else {
			//means that we have path string, some middlewares and the last of these is the handler
			theHandler := handlers[handlersLen-1].(HTTPHandler) // get the last function, which is the  actual handler of the route
			theMiddlewares := handlers[:handlersLen-1]          //get all except the last
			route = s.router.Handle(params[0].(string), theHandler)
			for _, theMiddleware := range theMiddlewares {
				route.UseFunc(theMiddleware.(func(http.ResponseWriter, *http.Request, http.HandlerFunc)))
			}

		}

	} else {
		err = errors.New("Not supported parameters passed to the Handle[Get,Post...] please refer to the docs")
	}
	if err != nil {
		panic(err.Error())
	}
	return route
}*/

//HandleFunc handle without methods, if not method given before the Listen then the http methods will be []{"GET"}
func (s *Server) HandleFunc(path string, handler Handler) *Route {
	return s.router.HandleFunc(path, handler)
}

// HandleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (s *Server) HandleAnnotated(irisHandler Annotated) (*Route, error) {
	return s.router.HandleAnnotated(irisHandler)
}

func (s *Server) Handle(params ...interface{}) *Route {
	paramsLen := len(params)
	if paramsLen == 0 {
		panic("No arguments given to the Handle function, please refer to docs")
	}

	if reflect.TypeOf(params[0]).Kind() == reflect.String {
		//menas first parameter is the path, wich means it is a simple path with handler -> HandleFunc
		return s.HandleFunc(params[0].(string), convertToHandler(params[1]))
	} else {
		//means it's a struct which implements the iris.Annotated and have a Handle func inside it -> HandleAnnotated
		r, err := s.HandleAnnotated(params[0].(Annotated))
		if err != nil {
			panic(err.Error())
		}
		return r
	}
}

// Get registers a route for the Get http method
func (s *Server) Get(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.GET)
}

// Post registers a route for the Post http method
func (s *Server) Post(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.POST)
}

// Put registers a route for the Put http method
func (s *Server) Put(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.PUT)
}

// Delete registers a route for the Delete http method
func (s *Server) Delete(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.DELETE)
}

// Connect registers a route for the Connect http method
func (s *Server) Connect(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.CONNECT)
}

// Head registers a route for the Head http method
func (s *Server) Head(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.HEAD)
}

// Options registers a route for the Options http method
func (s *Server) Options(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.OPTIONS)
}

// Patch registers a route for the Patch http method
func (s *Server) Patch(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.PATCH)
}

// Trace registers a route for the Trace http method
func (s *Server) Trace(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Method(HTTPMethods.TRACE)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (s *Server) Any(path string, handler interface{}) *Route {
	return s.HandleFunc(path, convertToHandler(handler)).Methods(HTTPMethods.ALL...)
}
