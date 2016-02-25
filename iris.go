package iris

// This file's usage is just to expose the server and it's router functionality
// But the fact is that it has got the only one init func in the project also
import (
	"net/http"
	"reflect"
	"strings"
)

var (
	avalaibleMethodsStr = strings.Join(HTTPMethods.ANY, ",")
	DefaultServer       *Server
)

// The one and only init to the whole package
// set the type of the Context,Renderer and set as default templatesDirectory to the CurrentDirectory of the sources(;)

func init() {
	//Context.go
	contextType = reflect.TypeOf(Context{})
	//Renderer.go
	rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	DefaultServer = New()
}

//Debug Setting to true you enable the go profiling tool
// Memory profile (http://localhost:PORT/debug/pprof/heap)
// CPU profile (http://localhost:PORT/debug/pprof/profile)
// Goroutine blocking profile (http://localhost:PORT/debug/pprof/block)
//
// Debug its the the only one option which is global and shared between multiple server instance will be the Debug
// Used in the server.go file when starting to the server and initialize the Mux.
var Debug = false

// New returns a new iris/server
func New() *Server {
	_server := new(Server)
	_server.config = DefaultServerConfig()
	_server.router = newRouter()
	_server.Errors = DefaultHTTPErrors()
	// the only usage:  server -> router -> route -> context -> context has directly access to emit http errors
	// like NotFound (no from Errors object because we are travel only the map of errors with their handlers)
	_server.router.errorHandlers = _server.Errors.errorHanders
	return _server
}

/////////////////////////////////
//for standalone instance of iris
/////////////////////////////////

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	DefaultServer.ServeHTTP(res, req)
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func Listen(fullHostOrPort interface{}) error {

	return DefaultServer.Listen(fullHostOrPort)
}

// Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) *Server {

	DefaultServer.router.Use(handler)
	return DefaultServer
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	DefaultServer.router.UseFunc(handlerFunc)
	return DefaultServer
}

// UseHandler registers a simple http.Handler as global middleware
func UseHandler(handler http.Handler) *Server {
	DefaultServer.router.UseHandler(handler)
	return DefaultServer
}

// Handle receives a struct ( look Router.HandleAnnotated) OR a path and an iris.Handler as second parameter ( look Router.HandleFunc)
func Handle(params ...interface{}) *Route {
	return DefaultServer.Handle(params...)
}

// Get registers a route for the Get http method
func Get(path string, handler interface{}) *Route {
	return DefaultServer.Get(path, handler)
}

// Post registers a route for the Post http method
func Post(path string, handler interface{}) *Route {
	return DefaultServer.Post(path, handler)
}

// Put registers a route for the Put http method
func Put(path string, handler interface{}) *Route {
	return DefaultServer.Put(path, handler)
}

// Delete registers a route for the Delete http method
func Delete(path string, handler interface{}) *Route {
	return DefaultServer.Delete(path, handler)
}

// Connect registers a route for the Connect http method
func Connect(path string, handler interface{}) *Route {
	return DefaultServer.Connect(path, handler)
}

// Head registers a route for the Head http method
func Head(path string, handler interface{}) *Route {
	return DefaultServer.Head(path, handler)
}

// Options registers a route for the Options http method
func Options(path string, handler interface{}) *Route {
	return DefaultServer.Options(path, handler)
}

// Patch registers a route for the Patch http method
func Patch(path string, handler interface{}) *Route {
	return DefaultServer.Patch(path, handler)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handler interface{}) *Route {
	return DefaultServer.Trace(path, handler)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handler interface{}) *Route {
	return DefaultServer.Any(path, handler)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { DefaultServer.Close() }
