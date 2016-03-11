package iris

import (
	"net/http"
	"time"
) // not very useful I can just pick the minute nanoseconds

// iris.go exposes the default global (iris.) public API from the New() default station
var (
	DefaultStation *Station
)

// The one and only init to the whole package
func init() {
	templatesDirectory = getCurrentDir()

	DefaultStation = New()
}

// New creates and returns a new iris Station with recommented options
func New() *Station {
	defaultOptions := StationOptions{
		Profile:            false,
		ProfilePath:        DefaultProfilePath,
		Cache:              true,
		CacheMaxItems:      0,
		CacheResetDuration: 5 * time.Minute,
	}
	return newStation(defaultOptions)
}

// Custom is used for iris-experienced developers
// creates and returns a new iris Station with custom StationOptions
func Custom(options StationOptions) *Station {
	return newStation(options)
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func Listen(fullHostOrPort interface{}) error {
	return DefaultStation.Listen(fullHostOrPort)
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func ListenTLS(fullHostOrPort interface{}, certFile, keyFile string) error {
	return DefaultStation.ListenTLS(fullHostOrPort, certFile, keyFile)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { DefaultStation.Close() }

// Router implementation

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party choosen because it has more fun
func Party(rootPath string) IParty {
	return DefaultStation.Party(rootPath)
}

// HandleFunc registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.HandlerFunc(func(res,req){})... or just use func(c iris.Context),func(r iris.Renderer), func(c Context,r Renderer) or func(res http.ResponseWriter, req *http.Request)
// method is the last parameter, pass the http method ex: "GET","POST".. iris.HTTPMethods.PUT, or empty string to match all methods
func HandleFunc(path string, handler Handler, method string) *Route {
	return DefaultStation.HandleFunc(path, handler, method)
}

// HandleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func HandleAnnotated(irisHandler Annotated) (*Route, error) {
	return DefaultStation.HandleAnnotated(irisHandler)
}

// Handle registers a route to the server's router, pass a struct -implements iris.Annotated as parameter
// Or pass just a http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
func Handle(params ...interface{}) *Route {
	return DefaultStation.Handle(params...)
}

// Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) {
	DefaultStation.Use(handler)
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	DefaultStation.UseFunc(handlerFunc)
}

// UseHandler registers a simple http.Handler as global middleware
func UseHandler(handler http.Handler) {
	DefaultStation.UseHandler(handler)
}

// Get registers a route for the Get http method
func Get(path string, handler interface{}) *Route {
	return DefaultStation.Get(path, handler)
}

// Post registers a route for the Post http method
func Post(path string, handler interface{}) *Route {
	return DefaultStation.Post(path, handler)
}

// Put registers a route for the Put http method
func Put(path string, handler interface{}) *Route {
	return DefaultStation.Put(path, handler)
}

// Delete registers a route for the Delete http method
func Delete(path string, handler interface{}) *Route {
	return DefaultStation.Delete(path, handler)
}

// Connect registers a route for the Connect http method
func Connect(path string, handler interface{}) *Route {
	return DefaultStation.Connect(path, handler)
}

// Head registers a route for the Head http method
func Head(path string, handler interface{}) *Route {
	return DefaultStation.Head(path, handler)
}

// Options registers a route for the Options http method
func Options(path string, handler interface{}) *Route {
	return DefaultStation.Options(path, handler)
}

// Patch registers a route for the Patch http method
func Patch(path string, handler interface{}) *Route {
	return DefaultStation.Patch(path, handler)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handler interface{}) *Route {
	return DefaultStation.Trace(path, handler)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handler interface{}) *Route {
	return DefaultStation.Any(path, handler)
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	DefaultStation.ServeHTTP(res, req)
}

// Errors sets and gets custom error handlers or responses
func Errors() *HTTPErrors {
	return DefaultStation.Errors()
}
