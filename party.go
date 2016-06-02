package iris

import (
	"path"
	"reflect"
	"strconv"
	"strings"

	"os"

	"time"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

type (
	// IParty is the interface which implements the whole Party of routes
	IParty interface {
		Handle(string, string, ...Handler) IRoute
		HandleFunc(string, string, ...HandlerFunc) IRoute
		API(path string, controller HandlerAPI, middlewares ...HandlerFunc) error
		Get(string, ...HandlerFunc) RouteNameFunc
		Post(string, ...HandlerFunc) RouteNameFunc
		Put(string, ...HandlerFunc) RouteNameFunc
		Delete(string, ...HandlerFunc) RouteNameFunc
		Connect(string, ...HandlerFunc) RouteNameFunc
		Head(string, ...HandlerFunc) RouteNameFunc
		Options(string, ...HandlerFunc) RouteNameFunc
		Patch(string, ...HandlerFunc) RouteNameFunc
		Trace(string, ...HandlerFunc) RouteNameFunc
		Any(string, ...HandlerFunc) []IRoute
		Use(...Handler)
		UseFunc(...HandlerFunc)
		StaticHandlerFunc(systemPath string, stripSlashes int, compress bool, generateIndexPages bool, indexNames []string) HandlerFunc
		Static(string, string, int)
		StaticFS(string, string, int)
		StaticWeb(relative string, systemPath string, stripSlashes int)
		StaticServe(systemPath string, requestPath ...string)
		Party(string, ...HandlerFunc) IParty // Each party can have a party too
		IsRoot() bool
	}

	// GardenParty  is the struct which makes all the job for registering routes and middlewares
	GardenParty struct {
		relativePath string
		station      *Iris // this station is where the party is happening, this station's Garden is the same for all Parties per Station & Router instance
		middleware   Middleware
		root         bool
	}
)

var _ IParty = &GardenParty{}

// IsRoot returns true if this is the root party ("/")
func (p *GardenParty) IsRoot() bool {
	return p.root
}

// Handle registers a route to the server's router
// if empty method is passed then registers handler(s) for all methods, same as .Any, but returns nil as result
func (p *GardenParty) Handle(method string, registedPath string, handlers ...Handler) IRoute {
	if method == "" { // then use like it was .Any
		for _, k := range AllMethods {
			p.Handle(k, registedPath, handlers...)
		}
		return nil
	}
	path := fixPath(p.relativePath + registedPath) // keep the last "/" as default ex: "/xyz/"
	if !p.station.config.DisablePathCorrection {
		// if we have path correction remove it with absPath
		path = fixPath(absPath(p.relativePath, registedPath)) // "/xyz"
	}
	middleware := JoinMiddleware(p.middleware, handlers)
	route := NewRoute(method, path, middleware)
	p.station.plugins.DoPreHandle(route)
	p.station.addRoute(route)
	p.station.plugins.DoPostHandle(route)
	return route
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.ToHandlerFunc(func(res,req){})... or just use func(c *iris.Context)
func (p *GardenParty) HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) IRoute {
	return p.Handle(method, registedPath, ConvertToHandlers(handlersFn)...)
}

// API converts & registers a custom struct to the router
// receives two parameters
// first is the request path (string)
// second is the custom struct (interface{}) which can be anything that has a *iris.Context as field.
// third are the common middlewares, is optional parameter
//
// Note that API's routes have their default-name to the full registed path,
// no need to give a special name for it, because it's not supposed to be used inside your templates.
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
func (p *GardenParty) API(path string, controller HandlerAPI, middlewares ...HandlerFunc) error {
	// here we need to find the registed methods and convert them to handler funcs
	// methods are collected by method naming:  Get(),GetBy(...), Post(),PostBy(...), Put() and so on

	typ := reflect.ValueOf(controller).Type()
	contextField, found := typ.FieldByName("Context")
	if !found {
		return ErrControllerContextNotFound.Return()
	}

	// check & register the Get(),Post(),Put(),Delete() and so on
	for _, methodName := range AllMethods {

		methodCapitalName := strings.Title(strings.ToLower(methodName))

		if method, found := typ.MethodByName(methodCapitalName); found {
			methodFunc := method.Func
			if !methodFunc.IsValid() || methodFunc.Type().NumIn() > 1 { // for any case
				continue
			}

			func(path string, typ reflect.Type, contextField reflect.StructField, methodFunc reflect.Value, method string) {
				handlersFn := make([]HandlerFunc, 0)
				handlersFn = append(handlersFn, middlewares...)
				handlersFn = append(handlersFn, func(ctx *Context) {
					newController := reflect.New(typ).Elem()
					newController.FieldByName("Context").Set(reflect.ValueOf(ctx))
					methodFunc.Call([]reflect.Value{newController})
				})
				// register route
				p.HandleFunc(method, path, handlersFn...)
			}(path, typ, contextField, methodFunc, methodName)

		}

	}

	// check for GetBy/PostBy(id string, something_else string) , these must be requested by the same order.
	// (we could do this in the same top loop but I don't want)
	// GET, DELETE -> with url named parameters (/users/:id/:secondArgumentIfExists)
	// POST, PUT -> with post values (form)
	// all other with URL Parameters (?something=this&else=other
	//
	// or no, I changed my mind, let all be named parameters and let users to decide what info they need,
	// using the Context to take more values (post form,url params and so on).-

	for _, methodName := range AllMethods {
		methodWithBy := strings.Title(strings.ToLower(methodName)) + "By"
		if method, found := typ.MethodByName(methodWithBy); found {
			methodFunc := method.Func
			if !methodFunc.IsValid() || methodFunc.Type().NumIn() < 2 { //it's By but it has not receive any arguments so its not api's
				continue
			}
			methodFuncType := methodFunc.Type()
			numInLen := methodFuncType.NumIn() // how much data we should receive from the request
			registedPath := path

			for i := 1; i < numInLen; i++ { // from 1 because the first is the 'object'
				if registedPath[len(registedPath)-1] == SlashByte {
					registedPath += ":param" + strconv.Itoa(i)
				} else {
					registedPath += "/:param" + strconv.Itoa(i)
				}
			}

			func(registedPath string, typ reflect.Type, contextField reflect.StructField, methodFunc reflect.Value, paramsLen int, method string) {
				handlersFn := make([]HandlerFunc, 0)
				handlersFn = append(handlersFn, middlewares...)
				handlersFn = append(handlersFn, func(ctx *Context) {
					newController := reflect.New(typ).Elem()
					newController.FieldByName("Context").Set(reflect.ValueOf(ctx))
					args := make([]reflect.Value, paramsLen+1, paramsLen+1)
					args[0] = newController
					for i := 0; i < paramsLen; i++ {
						args[i+1] = reflect.ValueOf(ctx.Params[i].Value)
					}
					methodFunc.Call(args)
				})
				// register route
				p.HandleFunc(method, registedPath, handlersFn...)
			}(registedPath, typ, contextField, methodFunc, numInLen-1, methodName)

		}

	}

	return nil
}

// Get registers a route for the Get http method
func (p *GardenParty) Get(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodGet, path, handlersFn...).Name
}

// Post registers a route for the Post http method
func (p *GardenParty) Post(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodPost, path, handlersFn...).Name
}

// Put registers a route for the Put http method
func (p *GardenParty) Put(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodPut, path, handlersFn...).Name
}

// Delete registers a route for the Delete http method
func (p *GardenParty) Delete(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodDelete, path, handlersFn...).Name
}

// Connect registers a route for the Connect http method
func (p *GardenParty) Connect(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodConnect, path, handlersFn...).Name
}

// Head registers a route for the Head http method
func (p *GardenParty) Head(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodHead, path, handlersFn...).Name
}

// Options registers a route for the Options http method
func (p *GardenParty) Options(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodOptions, path, handlersFn...).Name
}

// Patch registers a route for the Patch http method
func (p *GardenParty) Patch(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodPatch, path, handlersFn...).Name
}

// Trace registers a route for the Trace http method
func (p *GardenParty) Trace(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return p.HandleFunc(MethodTrace, path, handlersFn...).Name
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (p *GardenParty) Any(registedPath string, handlersFn ...HandlerFunc) []IRoute {
	theRoutes := make([]IRoute, len(AllMethods), len(AllMethods))
	for idx, k := range AllMethods {
		r := p.HandleFunc(k, registedPath, handlersFn...)
		theRoutes[idx] = r
	}

	return theRoutes
}

// H_ is used to convert a context.IContext handler func to iris.HandlerFunc, is used only inside iris internal package to avoid import cycles
func (p *GardenParty) H_(method string, registedPath string, fn func(context.IContext)) {
	p.HandleFunc(method, registedPath, func(ctx *Context) {
		fn(ctx)
	})
}

// Use registers a Handler middleware
func (p *GardenParty) Use(handlers ...Handler) {
	p.middleware = append(p.middleware, handlers...)
}

// UseFunc registers a HandlerFunc middleware
func (p *GardenParty) UseFunc(handlersFn ...HandlerFunc) {
	p.Use(ConvertToHandlers(handlersFn)...)
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
// It adds fasthttp.FSCompressedFileSuffix suffix to the original file name and
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
func (p *GardenParty) StaticHandlerFunc(systemPath string, stripSlashes int, compress bool, generateIndexPages bool, indexNames []string) HandlerFunc {
	if indexNames == nil {
		indexNames = []string{}
	}
	fs := &fasthttp.FS{
		// Path to directory to serve.
		Root:       systemPath,
		IndexNames: indexNames,
		// Generate index pages if client requests directory contents.
		GenerateIndexPages: generateIndexPages,

		// Enable transparent compression to save network traffic.
		Compress:             compress,
		CacheDuration:        config.StaticCacheDuration,
		CompressedFileSuffix: config.CompressedFileSuffix,
	}

	if stripSlashes > 0 {
		fs.PathRewrite = fasthttp.NewPathSlashesStripper(stripSlashes)
	}

	// Create request handler for serving static files.
	h := fs.NewRequestHandler()
	return func(ctx *Context) {
		h(ctx.RequestCtx)
		errCode := ctx.RequestCtx.Response.StatusCode()

		if errHandler := ctx.station.router.GetByCode(errCode); errHandler != nil {
			ctx.RequestCtx.Response.ResetBody()
			ctx.EmitError(errCode)
		}
		if ctx.pos < uint8(len(ctx.middleware))-1 {
			ctx.Next() // for any case
		}

	}
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
func (p *GardenParty) Static(relative string, systemPath string, stripSlashes int) {
	if relative[len(relative)-1] != SlashByte { // if / then /*filepath, if /something then /something/*filepath
		relative += "/"
	}

	h := p.StaticHandlerFunc(systemPath, stripSlashes, false, false, nil)

	p.Get(relative+"*filepath", h)
	p.Head(relative+"*filepath", h)
}

// StaticFS registers a route which serves a system directory
// this is the fastest method to serve static files
// generates an index page which list all files
// if you use this method it will generate compressed files also
// think this function as small fileserver with http
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func (p *GardenParty) StaticFS(reqPath string, systemPath string, stripSlashes int) {
	if reqPath[len(reqPath)-1] != SlashByte {
		reqPath += "/"
	}

	h := p.StaticHandlerFunc(systemPath, stripSlashes, true, true, nil)
	p.Get(reqPath+"*filepath", h)
	p.Head(reqPath+"*filepath", h)
}

// StaticWeb same as Static but if index.html exists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
// * if you don't know what to put on stripSlashes just 1

func (p *GardenParty) StaticWeb(reqPath string, systemPath string, stripSlashes int) {
	if reqPath[len(reqPath)-1] != SlashByte { // if / then /*filepath, if /something then /something/*filepath
		reqPath += "/"
	}

	hasIndex := utils.Exists(systemPath + utils.PathSeparator + "index.html")
	serveHandler := p.StaticHandlerFunc(systemPath, stripSlashes, false, !hasIndex, nil) // if not index.html exists then generate index.html which shows the list of files
	indexHandler := func(ctx *Context) {
		if len(ctx.Param("filepath")) < 2 && hasIndex {
			ctx.Request.SetRequestURI("index.html")
		}
		ctx.Next()

	}
	p.Get(reqPath+"*filepath", indexHandler, serveHandler)
	p.Head(reqPath+"*filepath", indexHandler, serveHandler)
}

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath ( the same path will be used to register the GET&HEAD routes)
// if second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache)
func (p *GardenParty) StaticServe(systemPath string, requestPath ...string) {
	var reqPath string

	if len(reqPath) > 0 {
		reqPath = requestPath[0]
	}

	reqPath = strings.Replace(systemPath, utils.PathSeparator, Slash, -1) // replaces any \ to /
	reqPath = strings.Replace(reqPath, "//", Slash, -1)                   // for any case, replaces // to /
	reqPath = strings.Replace(reqPath, ".", "", -1)                       // replace any dots (./mypath -> /mypath)

	p.Get(reqPath+"/*file", func(ctx *Context) {
		filepath := ctx.Param("file")

		path := strings.Replace(filepath, "/", utils.PathSeparator, -1)
		path = absPath(systemPath, path)

		if !utils.DirectoryExists(path) {
			ctx.NotFound()
			return
		}

		ctx.ServeFile(path, true)
	})
}

/* here in order to the subdomains be able to change favicon also */

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
func (p *GardenParty) Favicon(favPath string, requestPath ...string) error {
	f, err := os.Open(favPath)
	if err != nil {
		return ErrDirectoryFileNotFound.Format(favPath, err.Error())
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			return p.Favicon(path.Join(favPath, "favicon.png"))
		}
		favPath = fav
		fi, _ = f.Stat()
	}
	modtime := fi.ModTime().UTC().Format(TimeFormat)
	contentType := utils.TypeByExtension(favPath)
	// copy the bytes here in order to cache and not read the ico on each request.
	cacheFav := make([]byte, fi.Size())
	if _, err = f.Read(cacheFav); err != nil {
		return ErrDirectoryFileNotFound.Format(favPath, "Couldn't read the data bytes from ico: "+err.Error())
	}

	h := func(ctx *Context) {
		if t, err := time.Parse(TimeFormat, ctx.RequestHeader(IfModifiedSince)); err == nil && fi.ModTime().Before(t.Add(config.StaticCacheDuration)) {
			ctx.Response.Header.Del(ContentType)
			ctx.Response.Header.Del(ContentLength)
			ctx.SetStatusCode(StatusNotModified)
			return
		}

		ctx.Response.Header.Set(ContentType, contentType)
		ctx.Response.Header.Set(LastModified, modtime)
		ctx.SetStatusCode(StatusOK)
		ctx.Response.SetBody(cacheFav)
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	if len(requestPath) > 0 {
		reqPath = requestPath[0]
	}
	p.Get(reqPath, h)
	p.Head(reqPath, h)
	return nil
}

// StaticContent serves bytes, memory cached, on the reqPath
func (p *GardenParty) StaticContent(reqPath string, contentType string, content []byte) {
	modtime := time.Now()
	modtimeStr := modtime.UTC().Format(TimeFormat)

	h := func(ctx *Context) {
		if t, err := time.Parse(TimeFormat, ctx.RequestHeader(IfModifiedSince)); err == nil && modtime.Before(t.Add(config.StaticCacheDuration)) {
			ctx.Response.Header.Del(ContentType)
			ctx.Response.Header.Del(ContentLength)
			ctx.SetStatusCode(StatusNotModified)
			return
		}

		ctx.Response.Header.Set(ContentType, contentType)
		ctx.Response.Header.Set(LastModified, modtimeStr)
		ctx.SetStatusCode(StatusOK)
		ctx.Response.SetBody(content)
	}

	p.Get(reqPath, h)
	p.Head(reqPath, h)
}

/* */

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func (p *GardenParty) Party(path string, handlersFn ...HandlerFunc) IParty {
	middleware := ConvertToHandlers(handlersFn)
	if path[0] != SlashByte && strings.Contains(path, ".") {
		//it's domain so no handlers share (even the global ) or path, nothing.
	} else {
		// set path to parent+child
		path = absPath(p.relativePath, path)
		// append the parent's +child's handlers
		middleware = JoinMiddleware(p.middleware, middleware)
	}

	return &GardenParty{relativePath: path, station: p.station, middleware: middleware}
}

func absPath(rootPath string, relativePath string) (absPath string) {

	if relativePath == "" {
		absPath = rootPath
	} else {
		absPath = path.Join(rootPath, relativePath)
	}

	return
}

// fixPath fix the double slashes, (because of root,I just do that before the .Handle no need for anything else special)
func fixPath(str string) string {

	strafter := strings.Replace(str, "//", Slash, -1)

	if strafter[0] == SlashByte && strings.Count(strafter, ".") >= 2 {
		//it's domain, remove the first slash
		strafter = strafter[1:]
	}

	return strafter
}
