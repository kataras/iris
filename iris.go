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
	DefaultStation = New()
}

// defaultOptions returns the default options for the Station
func defaultOptions() StationOptions {
	return StationOptions{
		Profile:            false,
		ProfilePath:        DefaultProfilePath,
		Cache:              true,
		CacheMaxItems:      0,
		CacheResetDuration: 5 * time.Minute,
	}
}

// New creates and returns a new iris Station with recommented options
func New() *Station {
	defaultOptions := defaultOptions()
	return newStation(defaultOptions)
}

// Custom is used for iris-experienced developers
// creates and returns a new iris Station with custom StationOptions
//
// Note that if an option doesn't exist then the default value will be used instead
func Custom(options StationOptions) *Station {
	opt := defaultOptions()
	//check the given options one by one
	if options.Profile != opt.Profile {
		opt.Profile = options.Profile
	}
	if options.ProfilePath != "" {
		opt.ProfilePath = options.ProfilePath
	}
	if options.Cache != opt.Cache {
		opt.Cache = options.Cache
	}
	opt.CacheMaxItems = options.CacheMaxItems
	if options.CacheResetDuration > 30*time.Second { // 30 secs is the minimum value
		opt.CacheResetDuration = options.CacheResetDuration
	}

	return newStation(opt)
}

// Plugin activates the plugins and if succeed then adds it to the activated plugins list
func Plugin(plugin IPlugin) error {
	return DefaultStation.Plugin(plugin)
}

// Build is executed before the Listen automatically, if .Listen is not used then you should
// call .Build manually.
func Build() {
	DefaultStation.Build()
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port or empty, the default is 127.0.0.1:8080
func Listen(fullHostOrPort ...string) error {
	return DefaultStation.Listen(fullHostOrPort...)
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port
func ListenTLS(fullAddress string, certFile, keyFile string) error {
	return DefaultStation.ListenTLS(fullAddress, certFile, keyFile)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { DefaultStation.Close() }

// Router implementation

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func Party(rootPath string) IParty {
	return DefaultStation.Party(rootPath)
}

// Handle registers a route to the server's router
func Handle(method string, registedPath string, handler Handler) *Route {
	return DefaultStation.Handle(method, registedPath, handler)
}

// HandleFunc registers a route with a method, path string, and a handler
func HandleFunc(method string, path string, handler HandlerFunc) *Route {
	return DefaultStation.HandleFunc(method, path, handler)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
func HandleAnnotated(irisHandler Handler) (*Route, error) {
	return DefaultStation.HandleAnnotated(irisHandler)
}

// Use registers a a custom handler, with next, as a global middleware
func Use(handler MiddlewareHandler) {
	DefaultStation.Use(handler)
}

// UseFunc registers a function which is a handler, with next, as a global middleware
func UseFunc(handlerFunc func(ctx *Context, next Handler)) {
	DefaultStation.UseFunc(handlerFunc)
}

// Get registers a route for the Get http method
func Get(path string, handler HandlerFunc) *Route {
	return DefaultStation.Get(path, handler)
}

// Post registers a route for the Post http method
func Post(path string, handler HandlerFunc) *Route {
	return DefaultStation.Post(path, handler)
}

// Put registers a route for the Put http method
func Put(path string, handler HandlerFunc) *Route {
	return DefaultStation.Put(path, handler)
}

// Delete registers a route for the Delete http method
func Delete(path string, handler HandlerFunc) *Route {
	return DefaultStation.Delete(path, handler)
}

// Connect registers a route for the Connect http method
func Connect(path string, handler HandlerFunc) *Route {
	return DefaultStation.Connect(path, handler)
}

// Head registers a route for the Head http method
func Head(path string, handler HandlerFunc) *Route {
	return DefaultStation.Head(path, handler)
}

// Options registers a route for the Options http method
func Options(path string, handler HandlerFunc) *Route {
	return DefaultStation.Options(path, handler)
}

// Patch registers a route for the Patch http method
func Patch(path string, handler HandlerFunc) *Route {
	return DefaultStation.Patch(path, handler)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handler HandlerFunc) *Route {
	return DefaultStation.Trace(path, handler)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handler HandlerFunc) *Route {
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

// Templates sets the templates glob path for the web app
func Templates(pathGlob string) {
	DefaultStation.Templates(pathGlob)
}
