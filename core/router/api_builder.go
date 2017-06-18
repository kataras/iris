// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/errors"
	"github.com/cdren/iris/core/router/macro"
)

const (
	// MethodNone is a Virtual method
	// to store the "offline" routes
	MethodNone = "NONE"
)

var (
	// AllMethods contains the valid http methods:
	// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD",
	// "PATCH", "OPTIONS", "TRACE".
	AllMethods = [...]string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"CONNECT",
		"HEAD",
		"PATCH",
		"OPTIONS",
		"TRACE",
	}
)

// repository passed to all parties(subrouters), it's the object witch keeps
// all the routes.
type repository struct {
	routes []*Route
}

func (r *repository) register(route *Route) {
	r.routes = append(r.routes, route)
}

func (r *repository) get(routeName string) *Route {
	for _, r := range r.routes {
		if r.Name == routeName {
			return r
		}
	}
	return nil
}

func (r *repository) getAll() []*Route {
	return r.routes
}

// RoutesProvider should be implemented by
// iteral which contains the registered routes.
type RoutesProvider interface { // api builder
	GetRoutes() []*Route
	GetRoute(routeName string) *Route
}

// APIBuilder the visible API for constructing the router
// and child routers.
type APIBuilder struct {
	// the api builder global macros registry
	macros *macro.Map
	// the api builder global handlers per status code registry (used for custom http errors)
	errorCodeHandlers *ErrorCodeHandlers
	// the api builder global routes repository
	routes *repository
	// the api builder global route path reverser object
	// used by the view engine but it can be used anywhere.
	reverser *RoutePathReverser

	// the per-party middleware
	middleware context.Handlers
	// the per-party routes (useful only for done middleware)
	apiRoutes []*Route
	// the per-party done middleware
	doneHandlers context.Handlers
	// the per-party
	relativePath string
}

var _ Party = &APIBuilder{}
var _ RoutesProvider = &APIBuilder{} // passed to the default request handler (routerHandler)

// NewAPIBuilder creates & returns a new builder
// which is responsible to build the API and the router handler.
func NewAPIBuilder() *APIBuilder {
	rb := &APIBuilder{
		macros:            defaultMacros(),
		errorCodeHandlers: defaultErrorCodeHandlers(),
		relativePath:      "/",
		routes:            new(repository),
	}

	return rb
}

// Handle registers a route to the server's rb.
// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Handle(method string, registeredPath string, handlers ...context.Handler) (*Route, error) {
	// if registeredPath[0] != '/' {
	// 	return nil, errors.New("path should start with slash and should not be empty")
	// }

	if method == "" || method == "ALL" || method == "ANY" { // then use like it was .Any
		return nil, rb.Any(registeredPath, handlers...)
	}

	// no clean path yet because of subdomain indicator/separator which contains a dot.
	// but remove the first slash if the relative has already ending with a slash
	// it's not needed because later on we do normalize/clean the path, but better do it here too
	// for any future updates.
	if rb.relativePath[len(rb.relativePath)-1] == '/' {
		if registeredPath[0] == '/' {
			registeredPath = registeredPath[1:]
		}
	}

	fullpath := rb.relativePath + registeredPath // for now, keep the last "/" if any,  "/xyz/"

	routeHandlers := joinHandlers(rb.middleware, handlers)

	// here we separate the subdomain and relative path
	subdomain, path := exctractSubdomain(fullpath)
	if len(rb.doneHandlers) > 0 {
		routeHandlers = append(routeHandlers, rb.doneHandlers...) // register the done middleware, if any
	}

	r, err := NewRoute(method, subdomain, path, routeHandlers, rb.macros)
	if err != nil {
		return nil, err
	}
	// global
	rb.routes.register(r)

	// per -party
	rb.apiRoutes = append(rb.apiRoutes, r)
	// should we remove the rb.apiRoutes on the .Party (new children party) ?, No, because the user maybe use this party later
	// should we add to the 'inheritance tree' the rb.apiRoutes, No, these are for this specific party only, because the user propably, will have unexpected behavior when using Use/Use, Done/DoneFunc
	return r, nil
}

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
func (rb *APIBuilder) Party(relativePath string, handlers ...context.Handler) Party {
	parentPath := rb.relativePath
	dot := string(SubdomainIndicator[0])
	if len(parentPath) > 0 && parentPath[0] == '/' && strings.HasSuffix(relativePath, dot) { // if ends with . , example: admin., it's subdomain->
		parentPath = parentPath[1:] // remove first slash
	}

	// this is checked later on but for easier debug is better to do it here:
	if rb.relativePath[0] == '/' && relativePath[0] == '/' {
		parentPath = parentPath[1:] // remove  first slash if parent ended with / and new one started with /.
	}

	fullpath := parentPath + relativePath
	// append the parent's +child's handlers
	middleware := joinHandlers(rb.middleware, handlers)

	return &APIBuilder{
		// global/api builder
		macros:            rb.macros,
		routes:            rb.routes,
		errorCodeHandlers: rb.errorCodeHandlers,
		doneHandlers:      rb.doneHandlers,
		// per-party/children
		middleware:   middleware,
		relativePath: fullpath,
	}
}

// Macros returns the macro map which is responsible
// to register custom macro functions for all routes.
//
// Learn more at:  https://github.com/cdren/iris/tree/master/_examples/beginner/routing/dynamic-path
func (rb *APIBuilder) Macros() *macro.Map {
	return rb.macros
}

// GetRoutes returns the routes information,
// some of them can be changed at runtime some others not.
//
// Needs refresh of the router to Method or Path or Handlers changes to take place.
func (rb *APIBuilder) GetRoutes() []*Route {
	return rb.routes.getAll()
}

// GetRoute returns the registered route based on its name, otherwise nil.
// One note: "routeName" should be case-sensitive.
func (rb *APIBuilder) GetRoute(routeName string) *Route {
	return rb.routes.get(routeName)
}

// Use appends Handler(s) to the current Party's routes and child routes.
// If the current Party is the root, then it registers the middleware to all child Parties' routes too.
func (rb *APIBuilder) Use(handlers ...context.Handler) {
	rb.middleware = append(rb.middleware, handlers...)
}

// Done appends to the very end, Handler(s) to the current Party's routes and child routes
// The difference from .Use is that this/or these Handler(s) are being always running last.
func (rb *APIBuilder) Done(handlers ...context.Handler) {
	if len(rb.apiRoutes) > 0 { // register these middleware on previous-party-defined routes, it called after the party's route methods (Handle/HandleFunc/Get/Post/Put/Delete/...)
		for i, n := 0, len(rb.apiRoutes); i < n; i++ {
			routeInfo := rb.apiRoutes[i]
			routeInfo.Handlers = append(routeInfo.Handlers, handlers...)
		}
	} else {
		// register them on the doneHandlers, which will be used on Handle to append these middlweare as the last handler(s)
		rb.doneHandlers = append(rb.doneHandlers, handlers...)
	}
}

// UseGlobal registers Handler middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It should be called right before Listen functions
func (rb *APIBuilder) UseGlobal(handlers ...context.Handler) {
	for _, r := range rb.routes.routes {
		r.Handlers = append(handlers, r.Handlers...) // prepend the handlers
	}
	rb.middleware = append(handlers, rb.middleware...) // set as middleware on the next routes too
	// rb.Use(handlers...)
}

// None registers an "offline" route
// see context.ExecRoute(routeName) and
// party.Routes().Online(handleResultRouteInfo, "GET") and
// Offline(handleResultRouteInfo)
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) None(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(MethodNone, path, handlers...)
}

// Get registers a route for the Get http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Get(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodGet, path, handlers...)
}

// Post registers a route for the Post http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Post(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodPost, path, handlers...)
}

// Put registers a route for the Put http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Put(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodPut, path, handlers...)
}

// Delete registers a route for the Delete http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Delete(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodDelete, path, handlers...)
}

// Connect registers a route for the Connect http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Connect(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodConnect, path, handlers...)
}

// Head registers a route for the Head http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Head(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodHead, path, handlers...)
}

// Options registers a route for the Options http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Options(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodOptions, path, handlers...)
}

// Patch registers a route for the Patch http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Patch(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodPatch, path, handlers...)
}

// Trace registers a route for the Trace http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (rb *APIBuilder) Trace(path string, handlers ...context.Handler) (*Route, error) {
	return rb.Handle(http.MethodTrace, path, handlers...)
}

// Any registers a route for ALL of the http methods
// (Get,Post,Put,Head,Patch,Options,Connect,Delete).
func (rb *APIBuilder) Any(registeredPath string, handlers ...context.Handler) error {
	for _, k := range AllMethods {
		if _, err := rb.Handle(k, registeredPath, handlers...); err != nil {
			return err
		}
	}

	return nil
}

// StaticCacheDuration expiration duration for INACTIVE file handlers, it's the only one global configuration
// which can be changed.
var StaticCacheDuration = 20 * time.Second

const (
	lastModifiedHeaderKey       = "Last-Modified"
	ifModifiedSinceHeaderKey    = "If-Modified-Since"
	contentDispositionHeaderKey = "Content-Disposition"
	cacheControlHeaderKey       = "Cache-Control"
	contentEncodingHeaderKey    = "Content-Encoding"
	acceptEncodingHeaderKey     = "Accept-Encoding"
	// contentLengthHeaderKey represents the header["Content-Length"]
	contentLengthHeaderKey = "Content-Length"
	contentTypeHeaderKey   = "Content-Type"
	varyHeaderKey          = "Vary"
)

func (rb *APIBuilder) registerResourceRoute(reqPath string, h context.Handler) (*Route, error) {
	if _, err := rb.Head(reqPath, h); err != nil {
		return nil, err
	}
	return rb.Get(reqPath, h)
}

// StaticHandler returns a new Handler which is ready
// to serve all kind of static files.
//
// Note:
// The only difference from package-level `StaticHandler`
// is that this `StaticHandler`` receives a request path which
// is appended to the party's relative path and stripped here.
//
// Usage:
// app := iris.New()
// ...
// mySubdomainFsServer := app.Party("mysubdomain.")
// h := mySubdomainFsServer.StaticHandler("./static_files", false, false)
// /* http://mysubdomain.mydomain.com/static/css/style.css */
// mySubdomainFsServer.Get("/static", h)
// ...
//
func (rb *APIBuilder) StaticHandler(systemPath string, showList bool, enableGzip bool, exceptRoutes ...*Route) context.Handler {
	// Note: this doesn't need to be here but we'll keep it for consistently
	return StaticHandler(systemPath, showList, enableGzip)
}

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath,
// the same path will be used to register the GET and HEAD method routes.
// If second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache).
//
// Returns the GET *Route.
func (rb *APIBuilder) StaticServe(systemPath string, requestPath ...string) (*Route, error) {
	var reqPath string

	if len(requestPath) == 0 {
		reqPath = strings.Replace(systemPath, string(os.PathSeparator), "/", -1) // replaces any \ to /
		reqPath = strings.Replace(reqPath, "//", "/", -1)                        // for any case, replaces // to /
		reqPath = strings.Replace(reqPath, ".", "", -1)                          // replace any dots (./mypath -> /mypath)
	} else {
		reqPath = requestPath[0]
	}

	return rb.Get(joinPath(reqPath, WildcardParam("file")), func(ctx context.Context) {
		filepath := ctx.Params().Get("file")

		spath := strings.Replace(filepath, "/", string(os.PathSeparator), -1)
		spath = path.Join(systemPath, spath)

		if !DirectoryExists(spath) {
			ctx.NotFound()
			return
		}

		if err := ctx.ServeFile(spath, true); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
		}
	})
}

// StaticContent registers a GET and HEAD method routes to the requestPath
// that are ready to serve raw static bytes, memory cached.
//
// Returns the GET *Route.
func (rb *APIBuilder) StaticContent(reqPath string, cType string, content []byte) (*Route, error) {
	modtime := time.Now()
	h := func(ctx context.Context) {
		if err := ctx.WriteWithExpiration(content, cType, modtime); err != nil {
			ctx.NotFound()
			// ctx.Application().Log("error while serving []byte via StaticContent: %s", err.Error())
		}
	}

	return rb.registerResourceRoute(reqPath, h)
}

// StaticEmbeddedHandler returns a Handler which can serve
// embedded into executable files.
//
//
// Examples: https://github.com/cdren/iris/tree/master/_examples/file-server
func (rb *APIBuilder) StaticEmbeddedHandler(vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) context.Handler {
	// Notes:
	// This doesn't need to be APIBuilder's scope,
	// but we'll keep it here for consistently.
	return StaticEmbeddedHandler(vdir, assetFn, namesFn)
}

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function.
//
// Returns the GET *Route.
//
// Examples: https://github.com/cdren/iris/tree/master/_examples/file-server
func (rb *APIBuilder) StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) (*Route, error) {
	fullpath := joinPath(rb.relativePath, requestPath)
	requestPath = joinPath(fullpath, WildcardParam("file"))

	h := StripPrefix(fullpath, rb.StaticEmbeddedHandler(vdir, assetFn, namesFn))
	return rb.registerResourceRoute(requestPath, h)
}

// errDirectoryFileNotFound returns an error with message: 'Directory or file %s couldn't found. Trace: +error trace'
var errDirectoryFileNotFound = errors.New("Directory or file %s couldn't found. Trace: %s")

// Favicon serves static favicon
// accepts 2 parameters, second is optional
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico
// (nothing special that you can't handle by yourself).
// Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on).
//
// Returns the GET *Route.
func (rb *APIBuilder) Favicon(favPath string, requestPath ...string) (*Route, error) {
	favPath = Abs(favPath)

	f, err := os.Open(favPath)
	if err != nil {
		return nil, errDirectoryFileNotFound.Format(favPath, err.Error())
	}

	// ignore error f.Close()
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			return rb.Favicon(path.Join(favPath, "favicon.png"))
		}
		favPath = fav
		fi, _ = f.Stat()
	}

	cType := TypeByFilename(favPath)
	// copy the bytes here in order to cache and not read the ico on each request.
	cacheFav := make([]byte, fi.Size())
	if _, err = f.Read(cacheFav); err != nil {
		// Here we are before actually run the server.
		// So we could panic but we don't,
		// we just interrupt with a message
		// to the (user-defined) logger.
		return nil, errDirectoryFileNotFound.
			Format(favPath, "favicon: couldn't read the data bytes for file: "+err.Error())
	}
	modtime := ""
	h := func(ctx context.Context) {
		if modtime == "" {
			modtime = fi.ModTime().UTC().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
		}
		if t, err := time.Parse(ctx.Application().ConfigurationReadOnly().GetTimeFormat(), ctx.GetHeader(ifModifiedSinceHeaderKey)); err == nil && fi.ModTime().Before(t.Add(StaticCacheDuration)) {

			ctx.ResponseWriter().Header().Del(contentTypeHeaderKey)
			ctx.ResponseWriter().Header().Del(contentLengthHeaderKey)
			ctx.StatusCode(http.StatusNotModified)
			return
		}

		ctx.ResponseWriter().Header().Set(contentTypeHeaderKey, cType)
		ctx.ResponseWriter().Header().Set(lastModifiedHeaderKey, modtime)
		ctx.StatusCode(http.StatusOK)
		if _, err := ctx.Write(cacheFav); err != nil {
			// ctx.Application().Log("error while trying to serve the favicon: %s", err.Error())
			ctx.StatusCode(http.StatusInternalServerError)
		}
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	if len(requestPath) > 0 && requestPath[0] != "" {
		reqPath = requestPath[0]
	}

	return rb.registerResourceRoute(reqPath, h)
}

// StaticWeb returns a handler that serves HTTP requests
// with the contents of the file system rooted at directory.
//
// first parameter: the route path
// second parameter: the system directory
// third OPTIONAL parameter: the exception routes
//      (= give priority to these routes instead of the static handler)
// for more options look rb.StaticHandler.
//
//     rb.StaticWeb("/static", "./static")
//
// As a special case, the returned file server redirects any request
// ending in "/index.html" to the same path, without the final
// "index.html".
//
// StaticWeb calls the StaticHandler(systemPath, listingDirectories: false, gzip: false ).
//
// Returns the GET *Route.
func (rb *APIBuilder) StaticWeb(requestPath string, systemPath string, exceptRoutes ...*Route) (*Route, error) {

	paramName := "file"

	fullpath := joinPath(rb.relativePath, requestPath)

	h := StripPrefix(fullpath, rb.StaticHandler(systemPath, false, false, exceptRoutes...))

	handler := func(ctx context.Context) {
		h(ctx)
		// re-check the content type here for any case,
		// although the new code does it automatically but it's good to have it here.
		if ctx.GetStatusCode() >= 200 && ctx.GetStatusCode() < 400 {
			if fname := ctx.Params().Get(paramName); fname != "" {
				cType := TypeByFilename(fname)
				ctx.ContentType(cType)
			}
		}
	}
	requestPath = joinPath(fullpath, WildcardParam(paramName))
	return rb.registerResourceRoute(requestPath, handler)
}

// OnErrorCode registers an error http status code
// based on the "statusCode" >= 400.
// The handler is being wrapepd by a generic
// handler which will try to reset
// the body if recorder was enabled
// and/or disable the gzip if gzip response recorder
// was active.
func (rb *APIBuilder) OnErrorCode(statusCode int, handler context.Handler) {
	rb.errorCodeHandlers.Register(statusCode, handler)
}

// FireErrorCode executes an error http status code handler
// based on the context's status code.
//
// If a handler is not already registered,
// then it creates & registers a new trivial handler on the-fly.
func (rb *APIBuilder) FireErrorCode(ctx context.Context) {
	rb.errorCodeHandlers.Fire(ctx)
}

// Layout oerrides the parent template layout with a more specific layout for this Party
// returns this Party, to continue as normal
// Usage:
// app := iris.New()
// my := app.Party("/my").Layout("layouts/mylayout.html")
// 	{
// 		my.Get("/", func(ctx context.Context) {
// 			ctx.MustRender("page1.html", nil)
// 		})
// 	}
func (rb *APIBuilder) Layout(tmplLayoutFile string) Party {
	rb.Use(func(ctx context.Context) {
		ctx.ViewLayout(tmplLayoutFile)
		ctx.Next()
	})

	return rb
}

// joinHandlers uses to create a copy of all Handlers and return them in order to use inside the node
func joinHandlers(Handlers1 context.Handlers, Handlers2 context.Handlers) context.Handlers {
	nowLen := len(Handlers1)
	totalLen := nowLen + len(Handlers2)
	// create a new slice of Handlers in order to store all handlers, the already handlers(Handlers) and the new
	newHandlers := make(context.Handlers, totalLen)
	//copy the already Handlers to the just created
	copy(newHandlers, Handlers1)
	//start from there we finish, and store the new Handlers too
	copy(newHandlers[nowLen:], Handlers2)
	return newHandlers
}
