package iris

// This file's usage is just to expose the server and it's router functionality
// But the fact is that it has got the only one init func in the project also
import (
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
