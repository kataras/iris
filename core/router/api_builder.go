package router

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/mvc/activator"
)

const (
	// MethodNone is a Virtual method
	// to store the "offline" routes.
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
	// the api builder global errors, can be filled by the Subdomain, WildcardSubdomain, Handle...
	// the list of possible errors that can be
	// collected on the build state to log
	// to the end-user.
	reporter *errors.Reporter

	// the per-party handlers, order
	// of handlers registration matters.
	middleware context.Handlers
	// the global middleware handlers, order of call doesn't matters, order
	// of handlers registration matters. We need a secondary field for this
	// because `UseGlobal` registers handlers that should be executed
	// even before the `middleware` handlers, and in the same time keep the order
	// of handlers registration, so the same type of handlers are being called in order.
	beginGlobalHandlers context.Handlers
	// the per-party routes registry (useful for `Done` and `UseGlobal` only)
	apiRoutes []*Route
	// the per-party done handlers, order
	// of handlers registration matters.
	doneGlobalHandlers context.Handlers
	// the per-party
	relativePath string
}

var _ Party = &APIBuilder{}
var _ RoutesProvider = &APIBuilder{} // passed to the default request handler (routerHandler)

// NewAPIBuilder creates & returns a new builder
// which is responsible to build the API and the router handler.
func NewAPIBuilder() *APIBuilder {
	api := &APIBuilder{
		macros:            defaultMacros(),
		errorCodeHandlers: defaultErrorCodeHandlers(),
		reporter:          errors.NewReporter(),
		relativePath:      "/",
		routes:            new(repository),
	}

	return api
}

// GetReport returns an error may caused by party's methods.
func (api *APIBuilder) GetReport() error {
	return api.reporter.Return()
}

// GetReporter returns the reporter for adding errors
func (api *APIBuilder) GetReporter() *errors.Reporter {
	return api.reporter
}

// Handle registers a route to the server's api.
// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
//
// Returns a *Route, app will throw any errors later on.
func (api *APIBuilder) Handle(method string, relativePath string, handlers ...context.Handler) *Route {
	// if relativePath[0] != '/' {
	// 	return nil, errors.New("path should start with slash and should not be empty")
	// }

	if method == "" || method == "ALL" || method == "ANY" { // then use like it was .Any
		return api.Any(relativePath, handlers...)[0]
	}

	// no clean path yet because of subdomain indicator/separator which contains a dot.
	// but remove the first slash if the relative has already ending with a slash
	// it's not needed because later on we do normalize/clean the path, but better do it here too
	// for any future updates.
	if api.relativePath[len(api.relativePath)-1] == '/' {
		if relativePath[0] == '/' {
			relativePath = relativePath[1:]
		}
	}

	fullpath := api.relativePath + relativePath // for now, keep the last "/" if any,  "/xyz/"

	// global begin handlers -> middleware that are registered before route registration
	// -> handlers that are passed to this Handle function.
	routeHandlers := joinHandlers(append(api.beginGlobalHandlers, api.middleware...), handlers)
	// -> done handlers after all
	if len(api.doneGlobalHandlers) > 0 {
		routeHandlers = append(routeHandlers, api.doneGlobalHandlers...) // register the done middleware, if any
	}

	// here we separate the subdomain and relative path
	subdomain, path := splitSubdomainAndPath(fullpath)

	r, err := NewRoute(method, subdomain, path, routeHandlers, api.macros)
	if err != nil { // template path parser errors:
		api.reporter.Add("%v -> %s:%s:%s", err, method, subdomain, path)
		return nil
	}

	// global
	api.routes.register(r)

	// per -party, used for done handlers
	api.apiRoutes = append(api.apiRoutes, r)

	return r
}

// HandleMany works like `Handle` but can receive more than one
// paths separated by spaces and returns always a slice of *Route instead of a single instance of Route.
//
// It's useful only if the same handler can handle more than one request paths,
// otherwise use `Party` which can handle many paths with different handlers and middlewares.
//
// Usage:
// 	app.HandleMany(iris.MethodGet, "/user /user/{id:int} /user/me", userHandler)
// At the other side, with `Handle` we've had to write:
// 	app.Handle(iris.MethodGet, "/user", userHandler)
// 	app.Handle(iris.MethodGet, "/user/{id:int}", userHandler)
// 	app.Handle(iris.MethodGet, "/user/me", userHandler)
//
// This method is used behind the scenes at the `Controller` function
// in order to handle more than one paths for the same controller instance.
func (api *APIBuilder) HandleMany(method string, relativePath string, handlers ...context.Handler) (routes []*Route) {
	trimmedPath := strings.Trim(relativePath, " ")
	// at least slash
	// a space
	// at least one other slash for the next path
	// app.Controller("/user /user{id}", new(UserController))
	paths := strings.Split(trimmedPath, " ")
	for _, p := range paths {
		if p != "" {
			if method == "ANY" || method == "ALL" {
				routes = append(routes, api.Any(p, handlers...)...)
				continue
			}
			routes = append(routes, api.Handle(method, p, handlers...))
		}
	}
	return
}

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
func (api *APIBuilder) Party(relativePath string, handlers ...context.Handler) Party {
	parentPath := api.relativePath
	dot := string(SubdomainPrefix[0])
	if len(parentPath) > 0 && parentPath[0] == '/' && strings.HasSuffix(relativePath, dot) {
		// if ends with . , i.e admin., it's subdomain->
		parentPath = parentPath[1:] // remove first slash
	}

	// this is checked later on but for easier debug is better to do it here:
	if api.relativePath[len(api.relativePath)-1] == '/' && relativePath[0] == '/' {
		relativePath = relativePath[1:] // remove first slash if parent ended with / and new one started with /.
	}

	// if it's subdomain then it has priority, i.e:
	// api.relativePath == "admin."
	// relativePath == "panel."
	// then it should be panel.admin.
	// instead of admin.panel.
	if hasSubdomain(parentPath) && hasSubdomain(relativePath) {
		relativePath = relativePath + parentPath
		parentPath = ""
	}

	fullpath := parentPath + relativePath
	// append the parent's + child's handlers
	middleware := joinHandlers(api.middleware, handlers)

	return &APIBuilder{
		// global/api builder
		macros:              api.macros,
		routes:              api.routes,
		errorCodeHandlers:   api.errorCodeHandlers,
		beginGlobalHandlers: api.beginGlobalHandlers,
		doneGlobalHandlers:  api.doneGlobalHandlers,
		reporter:            api.reporter,
		// per-party/children
		middleware:   middleware,
		relativePath: fullpath,
	}
}

// PartyFunc same as `Party`, groups routes that share a base path or/and same handlers.
// However this function accepts a function that receives this created Party instead.
// Returns the Party in order the caller to be able to use this created Party to continue the
// top-bottom routes "tree".
//
// Note: `iris#Party` and `core/router#Party` describes the exactly same interface.
//
// Usage:
// app.PartyFunc("/users", func(u iris.Party){
//	u.Use(authMiddleware, logMiddleware)
//	u.Get("/", getAllUsers)
//	u.Post("/", createOrUpdateUser)
//	u.Delete("/", deleteUser)
// })
//
// Look `Party` for more.
func (api *APIBuilder) PartyFunc(relativePath string, partyBuilderFunc func(p Party)) Party {
	p := api.Party(relativePath)
	partyBuilderFunc(p)
	return p
}

// Subdomain returns a new party which is responsible to register routes to
// this specific "subdomain".
//
// If called from a child party then the subdomain will be prepended to the path instead of appended.
// So if app.Subdomain("admin.").Subdomain("panel.") then the result is: "panel.admin.".
func (api *APIBuilder) Subdomain(subdomain string, middleware ...context.Handler) Party {
	if api.relativePath == SubdomainWildcardIndicator {
		// cannot concat wildcard subdomain with something else
		api.reporter.Add("cannot concat parent wildcard subdomain with anything else ->  %s , %s",
			api.relativePath, subdomain)
		return api
	}
	return api.Party(subdomain, middleware...)
}

// WildcardSubdomain returns a new party which is responsible to register routes to
// a dynamic, wildcard(ed) subdomain. A dynamic subdomain is a subdomain which
// can reply to any subdomain requests. Server will accept any subdomain
// (if not static subdomain found) and it will search and execute the handlers of this party.
func (api *APIBuilder) WildcardSubdomain(middleware ...context.Handler) Party {
	if hasSubdomain(api.relativePath) {
		// cannot concat static subdomain with a dynamic one, wildcard should be at the root level
		api.reporter.Add("cannot concat static subdomain with a dynamic one. Dynamic subdomains should be at the root level -> %s",
			api.relativePath)
		return api
	}
	return api.Subdomain(SubdomainWildcardIndicator, middleware...)
}

// Macros returns the macro map which is responsible
// to register custom macro functions for all routes.
//
// Learn more at:  https://github.com/kataras/iris/tree/master/_examples/routing/dynamic-path
func (api *APIBuilder) Macros() *macro.Map {
	return api.macros
}

// GetRoutes returns the routes information,
// some of them can be changed at runtime some others not.
//
// Needs refresh of the router to Method or Path or Handlers changes to take place.
func (api *APIBuilder) GetRoutes() []*Route {
	return api.routes.getAll()
}

// GetRoute returns the registered route based on its name, otherwise nil.
// One note: "routeName" should be case-sensitive.
func (api *APIBuilder) GetRoute(routeName string) *Route {
	return api.routes.get(routeName)
}

// GetRouteReadOnly returns the registered "read-only" route based on its name, otherwise nil.
// One note: "routeName" should be case-sensitive. Used by the context to get the current route.
// It returns an interface instead to reduce wrong usage and to keep the decoupled design between
// the context and the routes.
//
// Look `GetRoute` for more.
func (api *APIBuilder) GetRouteReadOnly(routeName string) context.RouteReadOnly {
	r := api.GetRoute(routeName)
	if r == nil {
		return nil
	}
	return routeReadOnlyWrapper{r}
}

// Use appends Handler(s) to the current Party's routes and child routes.
// If the current Party is the root, then it registers the middleware to all child Parties' routes too.
//
// Call order matters, it should be called right before the routes that they care about these handlers.
//
// If it's called after the routes then these handlers will never be executed.
// Use `UseGlobal` if you want to register begin handlers(middleware)
// that should be always run before all application's routes.
func (api *APIBuilder) Use(handlers ...context.Handler) {
	api.middleware = append(api.middleware, handlers...)
}

// Done appends to the very end, Handler(s) to the current Party's routes and child routes
// The difference from .Use is that this/or these Handler(s) are being always running last.
func (api *APIBuilder) Done(handlers ...context.Handler) {
	for _, r := range api.routes.routes {
		r.done(handlers) // append the handlers to the existing routes
	}
	// set as done handlers for the next routes as well.
	api.doneGlobalHandlers = append(api.doneGlobalHandlers, handlers...)
}

// UseGlobal registers handlers that should run before all routes,
// including all parties, subdomains
// and other middleware that were registered before or will be after.
// It doesn't care about call order, it will prepend the handlers to all
// existing routes and the future routes that may being registered.
//
// It's always a good practise to call it right before the `Application#Run` function.
func (api *APIBuilder) UseGlobal(handlers ...context.Handler) {
	for _, r := range api.routes.routes {
		r.use(handlers) // prepend the handlers to the existing routes
	}
	// set as begin handlers for the next routes as well.
	api.beginGlobalHandlers = append(api.beginGlobalHandlers, handlers...)
}

// None registers an "offline" route
// see context.ExecRoute(routeName) and
// party.Routes().Online(handleResultRouteInfo, "GET") and
// Offline(handleResultRouteInfo)
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) None(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(MethodNone, relativePath, handlers...)
}

// Get registers a route for the Get http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Get(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodGet, relativePath, handlers...)
}

// Post registers a route for the Post http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Post(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPost, relativePath, handlers...)
}

// Put registers a route for the Put http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Put(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPut, relativePath, handlers...)
}

// Delete registers a route for the Delete http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Delete(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodDelete, relativePath, handlers...)
}

// Connect registers a route for the Connect http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Connect(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodConnect, relativePath, handlers...)
}

// Head registers a route for the Head http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Head(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodHead, relativePath, handlers...)
}

// Options registers a route for the Options http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Options(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodOptions, relativePath, handlers...)
}

// Patch registers a route for the Patch http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Patch(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPatch, relativePath, handlers...)
}

// Trace registers a route for the Trace http method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Trace(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodTrace, relativePath, handlers...)
}

// Any registers a route for ALL of the http methods
// (Get,Post,Put,Head,Patch,Options,Connect,Delete).
func (api *APIBuilder) Any(relativePath string, handlers ...context.Handler) (routes []*Route) {
	for _, m := range AllMethods {
		r := api.HandleMany(m, relativePath, handlers...)
		routes = append(routes, r...)
	}

	return
}

// Controller registers a `Controller` instance and returns the registered Routes.
// The "controller" receiver should embed a field of `Controller` in order
// to be compatible Iris `Controller`.
//
// It's just an alternative way of building an API for a specific
// path, the controller can register all type of http methods.
//
// Keep note that controllers are bit slow
// because of the reflection use however it's as fast as possible because
// it does preparation before the serve-time handler but still
// remains slower than the low-level handlers
// such as `Handle, Get, Post, Put, Delete, Connect, Head, Trace, Patch`.
//
//
// All fields that are tagged with iris:"persistence"` or binded
// are being persistence and kept the same between the different requests.
//
// An Example Controller can be:
//
// type IndexController struct {
// 	Controller
// }
//
// func (c *IndexController) Get() {
// 	c.Tmpl = "index.html"
// 	c.Data["title"] = "Index page"
// 	c.Data["message"] = "Hello world!"
// }
//
// Usage: app.Controller("/", new(IndexController))
//
//
// Another example with bind:
//
// type UserController struct {
// 	Controller
//
// 	DB        *DB
// 	CreatedAt time.Time
//
// }
//
// // Get serves using the User controller when HTTP Method is "GET".
// func (c *UserController) Get() {
// 	c.Tmpl = "user/index.html"
// 	c.Data["title"] = "User Page"
// 	c.Data["username"] = "kataras " + c.Params.Get("userid")
// 	c.Data["connstring"] = c.DB.Connstring
// 	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()
// }
//
// Usage: app.Controller("/user/{id:int}", new(UserController), db, time.Now())
// Note: Binded values of context.Handler type are being recognised as middlewares by the router.
//
// Read more at `/mvc#Controller`.
func (api *APIBuilder) Controller(relativePath string, controller activator.BaseController,
	bindValues ...interface{}) (routes []*Route) {

	registerFunc := func(ifRelPath string, method string, handlers ...context.Handler) {
		relPath := relativePath + ifRelPath
		r := api.HandleMany(method, relPath, handlers...)
		routes = append(routes, r...)
	}

	// bind any values to the controller's relative fields
	// and set them on each new request controller,
	// binder is an alternative method
	// of the persistence data control which requires the
	// user already set the values manually to controller's fields
	// and tag them with `iris:"persistence"`.
	//
	// don't worry it will never be handled if empty values.
	if err := activator.Register(controller, bindValues, registerFunc); err != nil {
		api.reporter.Add("%v for path: '%s'", err, relativePath)
	}

	return
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

func (api *APIBuilder) registerResourceRoute(reqPath string, h context.Handler) *Route {
	api.Head(reqPath, h)
	return api.Get(reqPath, h)
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
func (api *APIBuilder) StaticHandler(systemPath string, showList bool, gzip bool) context.Handler {
	// Note: this doesn't need to be here but we'll keep it for consistently
	return StaticHandler(systemPath, showList, gzip)
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
func (api *APIBuilder) StaticServe(systemPath string, requestPath ...string) *Route {
	var reqPath string

	if len(requestPath) == 0 {
		reqPath = strings.Replace(systemPath, string(os.PathSeparator), "/", -1) // replaces any \ to /
		reqPath = strings.Replace(reqPath, "//", "/", -1)                        // for any case, replaces // to /
		reqPath = strings.Replace(reqPath, ".", "", -1)                          // replace any dots (./mypath -> /mypath)
	} else {
		reqPath = requestPath[0]
	}

	return api.Get(joinPath(reqPath, WildcardParam("file")), func(ctx context.Context) {
		filepath := ctx.Params().Get("file")

		spath := strings.Replace(filepath, "/", string(os.PathSeparator), -1)
		spath = path.Join(systemPath, spath)

		if !DirectoryExists(spath) {
			ctx.NotFound()
			return
		}

		if err := ctx.ServeFile(spath, true); err != nil {
			ctx.Application().Logger().Warnf("while trying to serve static file: '%v' on IP: '%s'", err, ctx.RemoteAddr())
			ctx.StatusCode(http.StatusInternalServerError)
		}
	})
}

// StaticContent registers a GET and HEAD method routes to the requestPath
// that are ready to serve raw static bytes, memory cached.
//
// Returns the GET *Route.
func (api *APIBuilder) StaticContent(reqPath string, cType string, content []byte) *Route {
	modtime := time.Now()
	h := func(ctx context.Context) {
		ctx.ContentType(cType)
		if _, err := ctx.WriteWithExpiration(content, modtime); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			// ctx.Application().Logger().Infof("error while serving []byte via StaticContent: %s", err.Error())
		}
	}

	return api.registerResourceRoute(reqPath, h)
}

// StaticEmbeddedHandler returns a Handler which can serve
// embedded into executable files.
//
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/file-server
func (api *APIBuilder) StaticEmbeddedHandler(vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) context.Handler {
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
// Examples: https://github.com/kataras/iris/tree/master/_examples/file-server
func (api *APIBuilder) StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) *Route {
	fullpath := joinPath(api.relativePath, requestPath)
	requestPath = joinPath(fullpath, WildcardParam("file"))

	h := StripPrefix(fullpath, api.StaticEmbeddedHandler(vdir, assetFn, namesFn))
	return api.registerResourceRoute(requestPath, h)
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
func (api *APIBuilder) Favicon(favPath string, requestPath ...string) *Route {
	favPath = Abs(favPath)

	f, err := os.Open(favPath)
	if err != nil {
		api.reporter.AddErr(errDirectoryFileNotFound.Format(favPath, err.Error()))
		return nil
	}

	// ignore error f.Close()
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			return api.Favicon(path.Join(favPath, "favicon.png"))
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
		api.reporter.AddErr(errDirectoryFileNotFound.
			Format(favPath, "favicon: couldn't read the data bytes for file: "+err.Error()))
		return nil
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
			// ctx.Application().Logger().Infof("error while trying to serve the favicon: %s", err.Error())
			ctx.StatusCode(http.StatusInternalServerError)
		}
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	if len(requestPath) > 0 && requestPath[0] != "" {
		reqPath = requestPath[0]
	}

	return api.registerResourceRoute(reqPath, h)
}

// StaticWeb returns a handler that serves HTTP requests
// with the contents of the file system rooted at directory.
//
// first parameter: the route path
// second parameter: the system directory
//
// for more options look router.StaticHandler.
//
//     api.StaticWeb("/static", "./static")
//
// As a special case, the returned file server redirects any request
// ending in "/index.html" to the same path, without the final
// "index.html".
//
// StaticWeb calls the `StripPrefix(fullpath, NewStaticHandlerBuilder(systemPath).Listing(false).Build())`.
//
// Returns the GET *Route.
func (api *APIBuilder) StaticWeb(requestPath string, systemPath string) *Route {

	paramName := "file"

	fullpath := joinPath(api.relativePath, requestPath)

	h := StripPrefix(fullpath, NewStaticHandlerBuilder(systemPath).Listing(false).Build())

	handler := func(ctx context.Context) {
		h(ctx)
		if ctx.GetStatusCode() >= 200 && ctx.GetStatusCode() < 400 {
			// re-check the content type here for any case,
			// although the new code does it automatically but it's good to have it here.
			if _, exists := ctx.ResponseWriter().Header()["Content-Type"]; !exists {
				if fname := ctx.Params().Get(paramName); fname != "" {
					cType := TypeByFilename(fname)
					ctx.ContentType(cType)
				}
			}
		}
	}

	requestPath = joinPath(requestPath, WildcardParam(paramName))
	return api.registerResourceRoute(requestPath, handler)
}

// OnErrorCode registers an error http status code
// based on the "statusCode" >= 400.
// The handler is being wrapepd by a generic
// handler which will try to reset
// the body if recorder was enabled
// and/or disable the gzip if gzip response recorder
// was active.
func (api *APIBuilder) OnErrorCode(statusCode int, handlers ...context.Handler) {
	api.errorCodeHandlers.Register(statusCode, handlers...)
}

// OnAnyErrorCode registers a handler which called when error status code written.
// Same as `OnErrorCode` but registers all http error codes.
// See: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
func (api *APIBuilder) OnAnyErrorCode(handlers ...context.Handler) {
	// we could register all >=400 and <=511 but this way
	// could override custom status codes that iris developers can register for their
	//  web apps whenever needed.
	// There fore these are the hard coded http error statuses:
	var errStatusCodes = []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusProxyAuthRequired,
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusGone,
		http.StatusLengthRequired,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusExpectationFailed,
		http.StatusTeapot,
		http.StatusUnprocessableEntity,
		http.StatusLocked,
		http.StatusFailedDependency,
		http.StatusUpgradeRequired,
		http.StatusPreconditionRequired,
		http.StatusTooManyRequests,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnavailableForLegalReasons,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		http.StatusInsufficientStorage,
		http.StatusLoopDetected,
		http.StatusNotExtended,
		http.StatusNetworkAuthenticationRequired}

	for _, statusCode := range errStatusCodes {
		api.OnErrorCode(statusCode, handlers...)
	}
}

// FireErrorCode executes an error http status code handler
// based on the context's status code.
//
// If a handler is not already registered,
// then it creates & registers a new trivial handler on the-fly.
func (api *APIBuilder) FireErrorCode(ctx context.Context) {
	api.errorCodeHandlers.Fire(ctx)
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
func (api *APIBuilder) Layout(tmplLayoutFile string) Party {
	api.Use(func(ctx context.Context) {
		ctx.ViewLayout(tmplLayoutFile)
		ctx.Next()
	})

	return api
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
