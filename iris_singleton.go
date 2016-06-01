package iris

import (
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/mail"
	"github.com/kataras/iris/render/rest"
	"github.com/kataras/iris/render/template"
	"github.com/kataras/iris/server"
	"github.com/kataras/iris/websocket"
)

// DefaultIris in order to use iris.Get(...,...) we need a default Iris on the package level
var DefaultIris *Iris = New()

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It panics on error if you need a func to return an error use the ListenWithErr
// ex: iris.Listen(":8080")
func Listen(addr string) {
	DefaultIris.Listen(addr)
}

// ListenWithErr starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// if you need a func to panic on error use the Listen
// ex: log.Fatal(iris.ListenWithErr(":8080"))
func ListenWithErr(addr string) error {
	return DefaultIris.ListenWithErr(addr)
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It panics on error if you need a func to return an error use the ListenTLSWithErr
// ex: iris.ListenTLS(":8080","yourfile.cert","yourfile.key")
func ListenTLS(addr string, certFile, keyFile string) {
	DefaultIris.ListenTLS(addr, certFile, keyFile)
}

// ListenTLSWithErr Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// if you need a func to panic on error use the ListenTLS
// ex: log.Fatal(iris.ListenTLSWithErr(":8080","yourfile.cert","yourfile.key"))
func ListenTLSWithErr(addr string, certFile, keyFile string) error {
	return DefaultIris.ListenTLSWithErr(addr, certFile, keyFile)
}

// Close is used to close the net.Listener of the standalone http server which has already running via .Listen
func Close() { DefaultIris.Close() }

// Router implementation

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func Party(path string, handlersFn ...HandlerFunc) IParty {
	return DefaultIris.Party(path, handlersFn...)
}

// Handle registers a route to the server's router
// if empty method is passed then registers handler(s) for all methods, same as .Any
func Handle(method string, registedPath string, handlers ...Handler) {
	DefaultIris.Handle(method, registedPath, handlers...)
}

// HandleFunc registers a route with a method, path string, and a handler
func HandleFunc(method string, path string, handlersFn ...HandlerFunc) {
	DefaultIris.HandleFunc(method, path, handlersFn...)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
//
// HandleAnnotated will be deprecated until the final v3 !
func HandleAnnotated(irisHandler Handler) error {
	return DefaultIris.HandleAnnotated(irisHandler)
}

// API converts & registers a custom struct to the router
// receives three parameters
// first is the request path (string)
// second is the custom struct (interface{}) which can be anything that has a *iris.Context as field
// third are the common middlewares, is optional parameter
//
// Recommend to use when you retrieve data from an external database,
// and the router-performance is not the (only) thing which slows the server's overall performance.
//
// This is a slow method, if you care about router-performance use the Handle/HandleFunc/Get/Post/Put/Delete/Trace/Options... instead
//
// Usage:
// All the below methods are optional except the *iris.Context field,
// example with /users :
/*

package main

import (
	"github.com/kataras/iris"
)

type UserAPI struct {
	*iris.Context
}

// GET /users
func (u UserAPI) Get() {
	u.Write("Get from /users")
	// u.JSON(iris.StatusOK,myDb.AllUsers())
}

// GET /:param1 which its value passed to the id argument
func (u UserAPI) GetBy(id string) { // id equals to u.Param("param1")
	u.Write("Get from /users/%s", id)
	// u.JSON(iris.StatusOK, myDb.GetUserById(id))

}

// PUT /users
func (u UserAPI) Put() {
	name := u.FormValue("name")
	// myDb.InsertUser(...)
	println(string(name))
	println("Put from /users")
}

// POST /users/:param1
func (u UserAPI) PostBy(id string) {
	name := u.FormValue("name") // you can still use the whole Context's features!
	// myDb.UpdateUser(...)
	println(string(name))
	println("Post from /users/" + id)
}

// DELETE /users/:param1
func (u UserAPI) DeleteBy(id string) {
	// myDb.DeleteUser(id)
	println("Delete from /" + id)
}

func main() {
	iris.API("/users", UserAPI{})
	iris.Listen(":80")
}
*/
func API(registedPath string, controller HandlerAPI, middlewares ...HandlerFunc) error {
	return DefaultIris.API(registedPath, controller, middlewares...)
}

// Use appends a middleware to the route or to the router if it's called from router
func Use(handlers ...Handler) {
	DefaultIris.Use(handlers...)
}

// UseFunc same as Use but it accepts/receives ...HandlerFunc instead of ...Handler
// form of acceptable: func(c *iris.Context){//first middleware}, func(c *iris.Context){//second middleware}
func UseFunc(handlersFn ...HandlerFunc) {
	DefaultIris.UseFunc(handlersFn...)
}

// Get registers a route for the Get http method
func Get(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Get(path, handlersFn...)
}

// Post registers a route for the Post http method
func Post(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Post(path, handlersFn...)
}

// Put registers a route for the Put http method
func Put(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Put(path, handlersFn...)
}

// Delete registers a route for the Delete http method
func Delete(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Delete(path, handlersFn...)
}

// Connect registers a route for the Connect http method
func Connect(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Connect(path, handlersFn...)
}

// Head registers a route for the Head http method
func Head(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Head(path, handlersFn...)
}

// Options registers a route for the Options http method
func Options(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Options(path, handlersFn...)
}

// Patch registers a route for the Patch http method
func Patch(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Patch(path, handlersFn...)
}

// Trace registers a route for the Trace http methodd
func Trace(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Trace(path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(path string, handlersFn ...HandlerFunc) {
	DefaultIris.Any(path, handlersFn...)
}

// StaticHandlerFunc returns a HandlerFunc to serve static system directory
// Accepts 5 parameters
//
// first is the systemPath (string)
// Path to the root directory to serve files from.
//
// second is the  stripSlashes (int) level
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
//
// third is the compress (bool)
// Transparently compresses responses if set to true.
//
// The server tries minimizing CPU usage by caching compressed files.
// It adds FSCompressedFileSuffix suffix to the original file name and
// tries saving the resulting compressed file under the new file name.
// So it is advisable to give the server write access to Root
// and to all inner folders in order to minimze CPU usage when serving
// compressed responses.
//
// fourth is the generateIndexPages (bool)
// Index pages for directories without files matching IndexNames
// are automatically generated if set.
//
// Directory index generation may be quite slow for directories
// with many files (more than 1K), so it is discouraged enabling
// index pages' generation for such directories.
//
// fifth is the indexNames ([]string)
// List of index file names to try opening during directory access.
//
// For example:
//
//     * index.html
//     * index.htm
//     * my-super-index.xml
//
func StaticHandlerFunc(systemPath string, stripSlashes int, compress bool, generateIndexPages bool, indexNames []string) HandlerFunc {
	return DefaultIris.StaticHandlerFunc(systemPath, stripSlashes, compress, generateIndexPages, indexNames)
}

// Static registers a route which serves a system directory
// this doesn't generates an index page which list all files
// no compression is used also, for these features look at StaticFS func
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func Static(reqPath string, systemPath string, stripSlashes int) {
	DefaultIris.Static(reqPath, systemPath, stripSlashes)
}

// StaticFS registers a route which serves a system directory
// generates an index page which list all files
// uses compression which file cache, if you use this method it will generate compressed files also
// think this function as small fileserver with http
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func StaticFS(reqPath string, systemPath string, stripSlashes int) {
	DefaultIris.StaticFS(reqPath, systemPath, stripSlashes)
}

// StaticWeb same as Static but if index.html exists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func StaticWeb(reqPath string, systemPath string, stripSlashes int) {
	DefaultIris.StaticWeb(reqPath, systemPath, stripSlashes)
}

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath ( the same path will be used to register the GET&HEAD routes)
// if second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache)
func StaticServe(systemPath string, requestPath ...string) {
	DefaultIris.StaticServe(systemPath, requestPath...)
}

// Favicon serves static favicon
// accepts 2 parameters, second is optionally
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
// Note that you have to call it on every favicon you have to serve automatically (dekstop, mobile and so on)
//
// returns an error if something goes bad
func Favicon(favPath string, requestPath ...string) error {
	return DefaultIris.Favicon(favPath)
}

// StaticContent serves bytes, memory cached, on the reqPath
func StaticContent(reqPath string, contentType string, content []byte) {
	DefaultIris.StaticContent(reqPath, contentType, content)
}

// OnError Registers a handler for a specific http error status
func OnError(httpStatus int, handler HandlerFunc) {
	DefaultIris.OnError(httpStatus, handler)
}

// EmitError executes the handler of the given error http status code
func EmitError(httpStatus int, ctx *Context) {
	DefaultIris.EmitError(httpStatus, ctx)
}

// OnNotFound sets the handler for http status 404,
// default is a response with text: 'Not Found' and status: 404
func OnNotFound(handlerFunc HandlerFunc) {
	DefaultIris.OnNotFound(handlerFunc)
}

// OnPanic sets the handler for http status 500,
// default is a response with text: The server encountered an unexpected condition which prevented it from fulfilling the request. and status: 500
func OnPanic(handlerFunc HandlerFunc) {
	DefaultIris.OnPanic(handlerFunc)
}

// ***********************
// Export DefaultIris's  exported properties
// ***********************

// Server returns the server
func Server() *server.Server {
	return DefaultIris.Server()
}

// Plugins returns the plugin container
func Plugins() *PluginContainer {
	return DefaultIris.Plugins()
}

// Config returns the configs
func Config() *config.Iris {
	return DefaultIris.Config()
}

// Logger returns the logger
func Logger() *logger.Logger {
	return DefaultIris.Logger()
}

// Rest returns the rest render
func Rest() *rest.Render {
	return DefaultIris.Rest()
}

// Templates returns the template render
func Templates() *template.Template {
	return DefaultIris.Templates()
}

// Websocket returns the websocket server
func Websocket() websocket.Server {
	return DefaultIris.Websocket()
}

// Mail returns the mail sender service
func Mail() mail.Service {
	return DefaultIris.Mail()
}
