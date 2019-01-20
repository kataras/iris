package router

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/macro"
)

// MethodNone is a Virtual method
// to store the "offline" routes.
const MethodNone = "NONE"

var (
	// AllMethods contains the valid http methods:
	// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD",
	// "PATCH", "OPTIONS", "TRACE".
	AllMethods = []string{
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
	for _, r := range r.routes {
		if r.String() == route.String() {
			return // do not register any duplicates, the sooner the better.
		}
	}

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
	macros *macro.Macros
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

	// the per-party done handlers, order matters.
	doneHandlers context.Handlers
	// global done handlers, order doesn't matter.
	doneGlobalHandlers context.Handlers

	// the per-party
	relativePath string
	// allowMethods are filled with the `AllowMethods` func.
	// They are used to create new routes
	// per any party's (and its children) routes registered
	// if the method "x" wasn't registered already via  the `Handle` (and its extensions like `Get`, `Post`...).
	allowMethods []string

	// the per-party (and its children) execution rules for begin, main and done handlers.
	handlerExecutionRules ExecutionRules
}

var _ Party = (*APIBuilder)(nil)
var _ RoutesProvider = (*APIBuilder)(nil) // passed to the default request handler (routerHandler)

// NewAPIBuilder creates & returns a new builder
// which is responsible to build the API and the router handler.
func NewAPIBuilder() *APIBuilder {
	api := &APIBuilder{
		macros:            macro.Defaults,
		errorCodeHandlers: defaultErrorCodeHandlers(),
		reporter:          errors.NewReporter(),
		relativePath:      "/",
		routes:            new(repository),
	}

	return api
}

// GetRelPath returns the current party's relative path.
// i.e:
// if r := app.Party("/users"), then the `r.GetRelPath()` is the "/users".
// if r := app.Party("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
func (api *APIBuilder) GetRelPath() string {
	return api.relativePath
}

// GetReport returns an error may caused by party's methods.
func (api *APIBuilder) GetReport() error {
	return api.reporter.Return()
}

// GetReporter returns the reporter for adding errors
func (api *APIBuilder) GetReporter() *errors.Reporter {
	return api.reporter
}

// AllowMethods will re-register the future routes that will be registered
// via `Handle`, `Get`, `Post`, ... to the given "methods" on that Party and its children "Parties",
// duplicates are not registered.
//
// Call of `AllowMethod` will override any previous allow methods.
func (api *APIBuilder) AllowMethods(methods ...string) Party {
	api.allowMethods = methods
	return api
}

// SetExecutionRules alters the execution flow of the route handlers outside of the handlers themselves.
//
// For example, if for some reason the desired result is the (done or all) handlers to be executed no matter what
// even if no `ctx.Next()` is called in the previous handlers, including the begin(`Use`),
// the main(`Handle`) and the done(`Done`) handlers themselves, then:
// Party#SetExecutionRules(iris.ExecutionRules {
//   Begin: iris.ExecutionOptions{Force: true},
//   Main:  iris.ExecutionOptions{Force: true},
//   Done:  iris.ExecutionOptions{Force: true},
// })
//
// Note that if : true then the only remained way to "break" the handler chain is by `ctx.StopExecution()` now that `ctx.Next()` does not matter.
//
// These rules are per-party, so if a `Party` creates a child one then the same rules will be applied to that as well.
// Reset of these rules (before `Party#Handle`) can be done with `Party#SetExecutionRules(iris.ExecutionRules{})`.
//
// The most common scenario for its use can be found inside Iris MVC Applications;
// when we want the `Done` handlers of that specific mvc app's `Party`
// to be executed but we don't want to add `ctx.Next()` on the `OurController#EndRequest`.
//
// Returns this Party.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/mvc/middleware/without-ctx-next
func (api *APIBuilder) SetExecutionRules(executionRules ExecutionRules) Party {
	api.handlerExecutionRules = executionRules
	return api
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
	if len(handlers) == 0 {
		api.reporter.Add("missing handlers for route %s: %s", method, fullpath)
		return nil
	}

	// note: this can not change the caller's handlers as they're but the entry values(handlers)
	// of `middleware`, `doneHandlers` and `handlers` can.
	// So if we just put `api.middleware` or `api.doneHandlers`
	// then the next `Party` will have those updated handlers
	// but dev may change the rules for that child Party, so we have to make clones of them here.
	var (
		beginHandlers = joinHandlers(api.middleware, context.Handlers{})
		doneHandlers  = joinHandlers(api.doneHandlers, context.Handlers{})
	)

	mainHandlers := context.Handlers(handlers)
	// before join the middleware + handlers + done handlers and apply the execution rules.
	possibleMainHandlerName := context.HandlerName(mainHandlers[0])

	// TODO: for UseGlobal/DoneGlobal that doesn't work.
	applyExecutionRules(api.handlerExecutionRules, &beginHandlers, &doneHandlers, &mainHandlers)

	// global begin handlers -> middleware that are registered before route registration
	// -> handlers that are passed to this Handle function.
	routeHandlers := joinHandlers(beginHandlers, mainHandlers)
	// -> done handlers
	routeHandlers = joinHandlers(routeHandlers, doneHandlers)

	// here we separate the subdomain and relative path
	subdomain, path := splitSubdomainAndPath(fullpath)

	// if allowMethods are empty, then simply register with the passed, main, method.
	methods := append(api.allowMethods, method)

	var (
		route *Route // the latest one is this route registered, see methods append.
		err   error  // not used outside of loop scope.
	)

	for _, m := range methods {
		route, err = NewRoute(m, subdomain, path, possibleMainHandlerName, routeHandlers, *api.macros)
		if err != nil { // template path parser errors:
			api.reporter.Add("%v -> %s:%s:%s", err, method, subdomain, path)
			return nil // fail on first error.
		}

		// Add UseGlobal & DoneGlobal Handlers
		route.use(api.beginGlobalHandlers)
		route.done(api.doneGlobalHandlers)

		// global
		api.routes.register(route)
	}

	return route
}

// HandleMany works like `Handle` but can receive more than one
// paths separated by spaces and returns always a slice of *Route instead of a single instance of Route.
//
// It's useful only if the same handler can handle more than one request paths,
// otherwise use `Party` which can handle many paths with different handlers and middlewares.
//
// Usage:
// 	app.HandleMany("GET", "/user /user/{id:uint64} /user/me", genericUserHandler)
// At the other side, with `Handle` we've had to write:
// 	app.Handle("GET", "/user", userHandler)
// 	app.Handle("GET", "/user/{id:uint64}", userByIDHandler)
// 	app.Handle("GET", "/user/me", userMeHandler)
//
// This method is used behind the scenes at the `Controller` function
// in order to handle more than one paths for the same controller instance.
func (api *APIBuilder) HandleMany(methodOrMulti string, relativePathorMulti string, handlers ...context.Handler) (routes []*Route) {
	// at least slash
	// a space
	// at least one other slash for the next path
	paths := splitPath(relativePathorMulti)
	methods := splitMethod(methodOrMulti)
	for _, p := range paths {
		if p != "" {
			for _, method := range methods {
				if method == "" {
					method = "ANY"
				}
				if method == "ANY" || method == "ALL" {
					routes = append(routes, api.Any(p, handlers...)...)
					continue
				}
				routes = append(routes, api.Handle(method, p, handlers...))
			}

		}
	}
	return
}

// Party groups routes which may have the same prefix and share same handlers,
// returns that new rich subrouter.
//
// You can even declare a subdomain with relativePath as "mysub." or see `Subdomain`.
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

	// the allow methods per party and its children.
	allowMethods := make([]string, len(api.allowMethods))
	copy(allowMethods, api.allowMethods)

	return &APIBuilder{
		// global/api builder
		macros:              api.macros,
		routes:              api.routes,
		errorCodeHandlers:   api.errorCodeHandlers,
		beginGlobalHandlers: api.beginGlobalHandlers,
		doneGlobalHandlers:  api.doneGlobalHandlers,
		reporter:            api.reporter,
		// per-party/children
		middleware:            middleware,
		doneHandlers:          api.doneHandlers[0:],
		relativePath:          fullpath,
		allowMethods:          allowMethods,
		handlerExecutionRules: api.handlerExecutionRules,
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
// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
func (api *APIBuilder) Subdomain(subdomain string, middleware ...context.Handler) Party {
	if api.relativePath == SubdomainWildcardIndicator {
		// cannot concat wildcard subdomain with something else
		api.reporter.Add("cannot concat parent wildcard subdomain with anything else ->  %s , %s",
			api.relativePath, subdomain)
		return api
	}
	if l := len(subdomain); l < 1 {
		return api
	} else if subdomain[l-1] != '.' {
		subdomain += "."
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

// Macros returns the macro collection that is responsible
// to register custom macros with their own parameter types and their macro functions for all routes.
//
// Learn more at:  https://github.com/kataras/iris/tree/master/_examples/routing/dynamic-path
func (api *APIBuilder) Macros() *macro.Macros {
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
// Look `GetRoutesReadOnly` to fetch a list of all registered routes.
//
// Look `GetRoute` for more.
func (api *APIBuilder) GetRouteReadOnly(routeName string) context.RouteReadOnly {
	r := api.GetRoute(routeName)
	if r == nil {
		return nil
	}
	return routeReadOnlyWrapper{r}
}

// GetRoutesReadOnly returns the registered routes with "read-only" access,
// you cannot and you should not change any of these routes' properties on request state,
// you can use the `GetRoutes()` for that instead.
//
// It returns interface-based slice instead of the real ones in order to apply
// safe fetch between context(request-state) and the builded application.
//
// Look `GetRouteReadOnly` too.
func (api *APIBuilder) GetRoutesReadOnly() []context.RouteReadOnly {
	routes := api.GetRoutes()
	readOnlyRoutes := make([]context.RouteReadOnly, len(routes))
	for i, r := range routes {
		readOnlyRoutes[i] = routeReadOnlyWrapper{r}
	}

	return readOnlyRoutes
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

// UseGlobal registers handlers that should run at the very beginning.
// It prepends those handler(s) to all routes,
// including all parties, subdomains.
// It doesn't care about call order, it will prepend the handlers to all
// existing routes and the future routes that may being registered.
//
// The difference from `.DoneGLobal` is that this/or these Handler(s) are being always running first.
// Use of `ctx.Next()` of those handler(s) is necessary to call the main handler or the next middleware.
// It's always a good practise to call it right before the `Application#Run` function.
func (api *APIBuilder) UseGlobal(handlers ...context.Handler) {
	for _, r := range api.routes.routes {
		r.use(handlers) // prepend the handlers to the existing routes
	}
	// set as begin handlers for the next routes as well.
	api.beginGlobalHandlers = append(api.beginGlobalHandlers, handlers...)
}

// Done appends to the very end, Handler(s) to the current Party's routes and child routes.
//
// Call order matters, it should be called right before the routes that they care about these handlers.
//
// The difference from .Use is that this/or these Handler(s) are being always running last.
func (api *APIBuilder) Done(handlers ...context.Handler) {
	api.doneHandlers = append(api.doneHandlers, handlers...)
}

// DoneGlobal registers handlers that should run at the very end.
// It appends those handler(s) to all routes,
// including all parties, subdomains.
// It doesn't care about call order, it will append the handlers to all
// existing routes and the future routes that may being registered.
//
// The difference from `.UseGlobal` is that this/or these Handler(s) are being always running last.
// Use of `ctx.Next()` at the previous handler is necessary.
// It's always a good practise to call it right before the `Application#Run` function.
func (api *APIBuilder) DoneGlobal(handlers ...context.Handler) {
	for _, r := range api.routes.routes {
		r.done(handlers) // append the handlers to the existing routes
	}
	// set as done handlers for the next routes as well.
	api.doneGlobalHandlers = append(api.doneGlobalHandlers, handlers...)
}

// Reset removes all the begin and done handlers that may derived from the parent party via `Use` & `Done`,
// and the execution rules.
// Note that the `Reset` will not reset the handlers that are registered via `UseGlobal` & `DoneGlobal`.
//
// Returns this Party.
func (api *APIBuilder) Reset() Party {
	api.middleware = api.middleware[0:0]
	api.doneHandlers = api.doneHandlers[0:0]
	api.handlerExecutionRules = ExecutionRules{}
	return api
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

// StaticServe serves a directory as web resource.
// Same as `StaticWeb`.
// DEPRECATED; use `StaticWeb` or `StaticHandler` (for more options) instead.
func (api *APIBuilder) StaticServe(systemPath string, requestPath ...string) *Route {
	var reqPath string

	if len(requestPath) == 0 {
		reqPath = strings.Replace(systemPath, string(os.PathSeparator), "/", -1) // replaces any \ to /
		reqPath = strings.Replace(reqPath, "//", "/", -1)                        // for any case, replaces // to /
		reqPath = strings.Replace(reqPath, ".", "", -1)                          // replace any dots (./mypath -> /mypath)
	} else {
		reqPath = requestPath[0]
	}

	return api.StaticWeb(reqPath, systemPath)
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

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets" (no trailing slash),
// Third parameter is the Asset function
// Forth parameter is the AssetNames function.
//
// Returns the GET *Route.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-files-into-app
func (api *APIBuilder) StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) *Route {
	return api.staticEmbedded(requestPath, vdir, assetFn, namesFn, false)
}

// StaticEmbeddedGzip registers a route which can serve embedded gziped files
// that are embedded using the https://github.com/kataras/bindata tool and only.
// It's 8 times faster than the `StaticEmbeddedHandler` with `go-bindata` but
// it sends gzip response only, so the client must be aware that is expecting a gzip body
// (browsers and most modern browsers do that, so you can use it without fair).
//
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets" (no trailing slash),
// Third parameter is the GzipAsset function
// Forth parameter is the GzipAssetNames function.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-gziped-files-into-app
func (api *APIBuilder) StaticEmbeddedGzip(requestPath string, vdir string, gzipAssetFn func(name string) ([]byte, error), gzipNamesFn func() []string) *Route {
	return api.staticEmbedded(requestPath, vdir, gzipAssetFn, gzipNamesFn, true)
}

// look fs.go#StaticEmbeddedHandler
func (api *APIBuilder) staticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string, assetsGziped bool) *Route {
	fullpath := joinPath(api.relativePath, requestPath)
	// if subdomain,
	// here we get the full path of the path only,
	// because a subdomain can have parties as well
	// and we need that path to call the `StripPrefix`.
	_, fullpath = splitSubdomainAndPath(fullpath)

	paramName := "file"
	requestPath = joinPath(requestPath, WildcardParam(paramName))

	h := StaticEmbeddedHandler(vdir, assetFn, namesFn, assetsGziped)

	if fullpath != "/" {
		h = StripPrefix(fullpath, h)
	}

	// it handles the subdomain(root Party) of this party as well, if any.
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

	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		return api.Favicon(path.Join(favPath, "favicon.ico"))
	}

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

	modtime := time.Now()
	cType := TypeByFilename(favPath)
	h := func(ctx context.Context) {
		ctx.ContentType(cType)
		if _, err := ctx.WriteWithExpiration(cacheFav, modtime); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.Application().Logger().Debugf("while trying to serve the favicon: %s", err.Error())
		}
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) // we could use the filename, but because standards is /favicon.ico
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
	fullpath := joinPath(api.relativePath, requestPath)

	// if subdomain,
	// here we get the full path of the path only,
	// because a subdomain can have parties as well
	// and we need that path to call the `StripPrefix`.
	_, fullpath = splitSubdomainAndPath(fullpath)

	paramName := "file"
	requestPath = joinPath(requestPath, WildcardParam(paramName))

	h := NewStaticHandlerBuilder(systemPath).Listing(false).Build()

	if fullpath != "/" {
		h = StripPrefix(fullpath, h)
	}

	// it handles the subdomain(root Party) of this party as well, if any.
	return api.registerResourceRoute(requestPath, h)
}

// OnErrorCode registers an error http status code
// based on the "statusCode" < 200 || >= 400 (came from `context.StatusCodeNotSuccessful`).
// The handler is being wrapepd by a generic
// handler which will try to reset
// the body if recorder was enabled
// and/or disable the gzip if gzip response recorder
// was active.
func (api *APIBuilder) OnErrorCode(statusCode int, handlers ...context.Handler) {
	if len(api.beginGlobalHandlers) > 0 {
		handlers = joinHandlers(api.beginGlobalHandlers, handlers)
	}

	api.errorCodeHandlers.Register(statusCode, handlers...)
}

// OnAnyErrorCode registers a handler which called when error status code written.
// Same as `OnErrorCode` but registers all http error codes based on the `context.StatusCodeNotSuccessful`
// which defaults to < 200 || >= 400 for an error code, any previous error code will be overridden,
// so call it first if you want to use any custom handler for a specific error status code.
//
// Read more at: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
func (api *APIBuilder) OnAnyErrorCode(handlers ...context.Handler) {
	for code := 100; code <= 511; code++ {
		if context.StatusCodeNotSuccessful(code) {
			api.OnErrorCode(code, handlers...)
		}
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

// Layout overrides the parent template layout with a more specific layout for this Party.
// It returns the current Party.
//
// The "tmplLayoutFile" should be a relative path to the templates dir.
// Usage:
//
// app := iris.New()
// app.RegisterView(iris.$VIEW_ENGINE("./views", ".$extension"))
// my := app.Party("/my").Layout("layouts/mylayout.html")
// 	my.Get("/", func(ctx iris.Context) {
// 		ctx.View("page1.html")
// 	})
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/view
func (api *APIBuilder) Layout(tmplLayoutFile string) Party {
	api.Use(func(ctx context.Context) {
		ctx.ViewLayout(tmplLayoutFile)
		ctx.Next()
	})

	return api
}

// joinHandlers uses to create a copy of all Handlers and return them in order to use inside the node
func joinHandlers(h1 context.Handlers, h2 context.Handlers) context.Handlers {
	nowLen := len(h1)
	totalLen := nowLen + len(h2)
	// create a new slice of Handlers in order to merge the "h1" and "h2"
	newHandlers := make(context.Handlers, totalLen)
	// copy the already Handlers to the just created
	copy(newHandlers, h1)
	// start from there we finish, and store the new Handlers too
	copy(newHandlers[nowLen:], h2)
	return newHandlers
}
