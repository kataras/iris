package iris

/*
This file's usage is just to expose the server and it's router functionality
But the fact is that it has got the only one init func in the project also
*/
import (
	"net/http"
	"reflect"
	"strings"
)

var (
	avalaibleMethodsStr = strings.Join(HTTPMethods.ANY, ",")
	mainIris            *Server
)

/*
The one and only init to the whole package
set the type of the Context,Renderer and set as default templatesDirectory to the CurrentDirectory of the sources(;)
*/
func init() {
	//Context.go
	contextType = reflect.TypeOf(Context{})
	//Renderer.go
	rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	//I don't want to store in the memory a New() Iris because user maybe wants to use the form of api := Iris.New(); api.Get... instead of Iris.Get... (yet)
	mainIris = nil
}

//New returns a new iris/server
func New() *Server {
	_server := new(Server)
	_server.config = DefaultServerConfig()
	_server.router = NewRouter()
	return _server
}

/////////////////////////////////
//for standalone instance of iris
/////////////////////////////////

func check() {
	if mainIris == nil {
		mainIris = New()
	}
}

/*
ServeHTTP serves an http request,
with this function iris can be used also as  a middleware into other already defined http server
*/
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	check()
	mainIris.ServeHTTP(res, req)
}

/*
Listen starts the standalone http server
which listens to the fullHostOrPort parameter which as the form of
host:port or just port
*/
func Listen(fullHostOrPort interface{}) error {
	check()
	return mainIris.Listen(fullHostOrPort)
}

//Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) *Server {
	check()
	mainIris.router.Use(handler)
	return mainIris
}

//UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	check()
	mainIris.router.UseFunc(handlerFunc)
	return mainIris
}

//UseHandler registers a simple http.Handler as global middleware
func UseHandler(handler http.Handler) *Server {
	check()
	mainIris.router.UseHandler(handler)
	return mainIris
}

/*
Handle registers a handler in the Router and returns a Route
Parameters (path string, handler HTTPHandler OR any struct implements the custom Iris Handler interface.)
*/
func Handle(params ...interface{}) *Route {
	check()
	return mainIris.Handle(params...)

}

//Get registers a route for the Get http method
func Get(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Get(path, handler)
}

//Post registers a route for the Post http method
func Post(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Post(path, handler)
}

//Put registers a route for the Put http method
func Put(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Put(path, handler)
}

//Delete registers a route for the Delete http method
func Delete(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Delete(path, handler)
}

//Connect registers a route for the Connect http method
func Connect(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Connect(path, handler)
}

//Head registers a route for the Head http method
func Head(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Head(path, handler)
}

//Options registers a route for the Options http method
func Options(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Options(path, handler)
}

//Patch registers a route for the Patch http method
func Patch(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Patch(path, handler)
}

//Trace registers a route for the Trace http methodd
func Trace(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Trace(path, handler)
}

//Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Any(path, handler)
}

/*
RegisterHandler registers a route handler using a Struct
implements Handle() function and has iris.Handler anonymous property
which it's metadata has the form of
`method:"path" template:"file.html"` and returns the route and an error if any occurs
*/
func RegisterHandler(irisHandler Handler) (*Route, error) {
	check()
	return mainIris.RegisterHandler(irisHandler)
}

//Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { mainIris.Close() }
