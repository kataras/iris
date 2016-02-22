package iris

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"strconv"
	"strings"
)

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
	listenerTCP net.Listener
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
	mux := http.DefaultServeMux
	if !Debug {
		mux = http.NewServeMux()
	}

	mux.Handle("/", s)

	//return http.ListenAndServe(s.config.Host+strconv.Itoa(s.config.Port), mux)
	fullAddr := s.config.Host + ":" + strconv.Itoa(s.config.Port)
	listener, err := net.Listen("tcp", fullAddr)

	if err != nil {
		panic("Cannot run the server [problem with tcp listener on host:port]: " + fullAddr)
	}

	s.listenerTCP = listener //we need this because I think that we have to 'have' a Close() method on the server instance
	err = http.Serve(s.listenerTCP, mux)
	s.listenerTCP.Close()
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

// Handle registers a handler in the router and returns a Route
// Parameters (path string, handler HTTPHandler OR any struct implements the custom Iris Handler interface.)
func (s *Server) Handle(params ...interface{}) *Route {
	//poor, but means path, custom HTTPhandler
	if len(params) == 2 {
		return s.router.Handle(params[0].(string), params[1].(HTTPHandler))
	}
	route, err := s.RegisterHandler(params[0].(Handler))

	if err != nil {
		panic(err.Error())
	}

	return route

}

// Get registers a route for the Get http method
func (s *Server) Get(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.GET)
}

// Post registers a route for the Post http method
func (s *Server) Post(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.POST)
}

// Put registers a route for the Put http method
func (s *Server) Put(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.PUT)
}

// Delete registers a route for the Delete http method
func (s *Server) Delete(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.DELETE)
}

// Connect registers a route for the Connect http method
func (s *Server) Connect(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.CONNECT)
}

// Head registers a route for the Head http method
func (s *Server) Head(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.HEAD)
}

// Options registers a route for the Options http method
func (s *Server) Options(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.OPTIONS)
}

// Patch registers a route for the Patch http method
func (s *Server) Patch(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.PATCH)
}

// Trace registers a route for the Trace http method
func (s *Server) Trace(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.TRACE)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (s *Server) Any(path string, handler HTTPHandler) *Route {
	return s.router.Handle(path, handler, HTTPMethods.ALL...)
}

// RegisterHandler registers a route handler using a Struct
// implements Handle() function and has iris.Handler anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (s *Server) RegisterHandler(irisHandler Handler) (*Route, error) {
	return s.router.RegisterHandler(irisHandler)
}
