package iris

// This file's usage is just to expose the server and it's router functionality
// But the fact is that it has got the only one init func in the project also
import (
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
	//contextType = reflect.TypeOf(Context{})
	//Renderer.go
	//rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	DefaultServer = New()
}

type IrisOptions struct {
	Cache bool
}

// New returns a new iris/server
func New(options ...IrisOptions) *Server {
	_server := new(Server)

	if options != nil && len(options) > 0 {
		if options[0].Cache == false {
			_server.router = NewRouter()

		}
	} else {
		_server.router = NewMemoryRouter() //the default will be the memory router
	}

	_server.Errors = DefaultHTTPErrors()
	// the only usage:  server -> router -> route -> context -> context has directly access to emit http errors
	// like NotFound (no from Errors object because we are travel only the map of errors with their handlers)
	_server.router.SetErrors(_server.Errors)
	//it's funny If I totaly remove the httperrors  from server , router, route and context
	//then the allocs/op goes 1080 but if I keep them it's stays on 766 allocs/op wtf is going on
	return _server
}
