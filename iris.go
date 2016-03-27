// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"net/http"
	"time"
)

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
		PathCorrection:     true,
	}
}

// New creates and returns a new iris Station with recommented options
func New() *Station {
	defaultOptions := defaultOptions()
	return newStation(defaultOptions)
}

// Custom is used for iris-experienced developers
// creates and returns a new iris Station with custom StationOptions
func Custom(options StationOptions) *Station {

	if options.ProfilePath != "" {
		options.ProfilePath = DefaultProfilePath
	}

	if options.CacheResetDuration < 30*time.Second { // 30 secs is the minimum value, if not then set it to 5minutes
		options.CacheResetDuration = 5 * time.Minute
	}

	return newStation(options)
}

// Plugin activates the plugins and if succeed then adds it to the activated plugins list
func Plugin(plugin IPlugin) error {
	return DefaultStation.Plugin(plugin)
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
func Handle(method string, registedPath string, handlers ...Handler) {
	DefaultStation.Handle(method, registedPath, handlers...)
}

// HandleFunc registers a route with a method, path string, and a handler
func HandleFunc(method string, path string, handlersFn ...HandlerFunc) {
	DefaultStation.HandleFunc(method, path, handlersFn...)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
func HandleAnnotated(irisHandler Handler) error {
	return DefaultStation.HandleAnnotated(irisHandler)
}

// Use appends a middleware to the route or to the router if it's called from router
func Use(handlers ...Handler) {
	DefaultStation.Use(handlers...)
}

// UseFunc same as Use but it accepts/receives ...HandlerFunc instead of ...Handler
// form of acceptable: func(c *iris.Context){//first middleware}, func(c *iris.Context){//second middleware}
func UseFunc(handlersFn ...HandlerFunc) {
	DefaultStation.UseFunc(handlersFn...)
}

// Get registers a route for the Get http method
func Get(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Get(path, handlersFn...)
}

// Post registers a route for the Post http method
func Post(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Post(path, handlersFn...)
}

// Put registers a route for the Put http method
func Put(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Put(path, handlersFn...)
}

// Delete registers a route for the Delete http method
func Delete(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Delete(path, handlersFn...)
}

// Connect registers a route for the Connect http method
func Connect(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Connect(path, handlersFn...)
}

// Head registers a route for the Head http method
func Head(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Head(path, handlersFn...)
}

// Options registers a route for the Options http method
func Options(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Options(path, handlersFn...)
}

// Patch registers a route for the Patch http method
func Patch(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Patch(path, handlersFn...)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Trace(path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handlersFn ...HandlerFunc) {
	DefaultStation.Any(path, handlersFn...)
}

// Ws registers a websocket route
func Ws(path string, handler Handler) {
	DefaultStation.Ws(path, handler)
}

// ServeHTTP serves an http request,
// with this function iris can be used also as  a middleware into other already defined http server
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	DefaultStation.ServeHTTP(res, req)
}

// Error handling

// Errors returns the object which is resposible for the error(s) handler(s)
func Errors() IHTTPErrors {
	return DefaultStation.Errors()
}

func OnError(statusCode int, handlerFunc HandlerFunc) {
	DefaultStation.OnError(statusCode, handlerFunc)
}

// EmitError emits an error with it's http status code and the iris Context passed to the function
func EmitError(statusCode int, ctx *Context) {
	DefaultStation.EmitError(statusCode, ctx)
}

// OnNotFound sets the handler for http status 404,
// default is a response with text: 'Not Found' and status: 404
func OnNotFound(handlerFunc HandlerFunc) {
	DefaultStation.OnNotFound(handlerFunc)
}

// OnPanic sets the handler for http status 500,
// default is a response with text: The server encountered an unexpected condition which prevented it from fulfilling the request. and status: 500
func OnPanic(handlerFunc HandlerFunc) {
	DefaultStation.OnPanic(handlerFunc)
}

//
// Templates sets the templates glob path for the web app
func Templates(pathGlob string) {
	DefaultStation.Templates(pathGlob)
}

// OptimusPrime, YOU MUST RUN IT ONLY IF YOU DON'T USE iris.Listen or iris.Serve() method
func OptimusPrime() {
	DefaultStation.OptimusPrime()
}

// Serve is used instead of the iris.Listen
// eg  http.ListenAndServe(":80",iris.Serve()) if you don't want to use iris.Listen(":80") ( you can't use iris because its package variable it's golang limitation)
func Serve() http.Handler {
	OptimusPrime()
	return DefaultStation.IRouter
}
