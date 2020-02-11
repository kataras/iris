package router

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/errgroup"
	"github.com/kataras/iris/v12/macro"
	macroHandler "github.com/kataras/iris/v12/macro/handler"
)

// MethodNone is a Virtual method
// to store the "offline" routes.
const MethodNone = "NONE"

// AllMethods contains the valid http methods:
// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD",
// "PATCH", "OPTIONS", "TRACE".
var AllMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodHead,
	http.MethodPatch,
	http.MethodOptions,
	http.MethodTrace,
}

// repository passed to all parties(subrouters), it's the object witch keeps
// all the routes.
type repository struct {
	routes []*Route
	pos    map[string]int
}

func (repo *repository) get(routeName string) *Route {
	for _, r := range repo.routes {
		if r.Name == routeName {
			return r
		}
	}
	return nil
}

func (repo *repository) getRelative(r *Route) *Route {
	if r.tmpl.IsTrailing() || !macroHandler.CanMakeHandler(r.tmpl) {
		return nil
	}

	for _, route := range repo.routes {
		if r.Subdomain == route.Subdomain && r.Method == route.Method && r.FormattedPath == route.FormattedPath && !route.tmpl.IsTrailing() {
			return route
		}
	}

	return nil
}

func (repo *repository) getByPath(tmplPath string) *Route {
	if repo.pos != nil {
		if idx, ok := repo.pos[tmplPath]; ok {
			if len(repo.routes) > idx {
				return repo.routes[idx]
			}
		}
	}

	return nil
}

func (repo *repository) getAll() []*Route {
	return repo.routes
}

func (repo *repository) register(route *Route, rule RouteRegisterRule) (*Route, error) {
	for i, r := range repo.routes {
		// 14 August 2019 allow register same path pattern with different macro functions,
		// see #1058
		if route.DeepEqual(r) {
			if rule == RouteSkip {
				return r, nil
			} else if rule == RouteError {
				return nil, fmt.Errorf("new route: %s conflicts with an already registered one: %s route", route.String(), r.String())
			} else {
				// replace existing with the latest one, the default behavior.
				repo.routes = append(repo.routes[:i], repo.routes[i+1:]...)
			}

			continue
		}
	}

	repo.routes = append(repo.routes, route)
	if repo.pos == nil {
		repo.pos = make(map[string]int)
	}

	repo.pos[route.tmpl.Src] = len(repo.routes) - 1
	return route, nil
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

	// the api builder global errors, can be filled by the Subdomain, WildcardSubdomain, Handle...
	// the list of possible errors that can be
	// collected on the build state to log
	// to the end-user.
	errors *errgroup.Group

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
	// the per-party (and its children) route registration rule, see `SetRegisterRule`.
	routeRegisterRule RouteRegisterRule
}

var _ Party = (*APIBuilder)(nil)
var _ RoutesProvider = (*APIBuilder)(nil) // passed to the default request handler (routerHandler)

// NewAPIBuilder creates & returns a new builder
// which is responsible to build the API and the router handler.
func NewAPIBuilder() *APIBuilder {
	api := &APIBuilder{
		macros:            macro.Defaults,
		errorCodeHandlers: defaultErrorCodeHandlers(),
		errors:            errgroup.New("API Builder"),
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

// GetReporter returns the reporter for adding or receiving any errors caused when building the API.
func (api *APIBuilder) GetReporter() *errgroup.Group {
	return api.errors
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

// RouteRegisterRule is a type of uint8.
// Defines the register rule for new routes that already exists.
// Available values are: RouteOverride, RouteSkip and RouteError.
//
// See `Party#SetRegisterRule`.
type RouteRegisterRule uint8

const (
	// RouteOverride an existing route with the new one, the default rule.
	RouteOverride RouteRegisterRule = iota
	// RouteSkip registering a new route twice.
	RouteSkip
	// RouteError log when a route already exists, shown after the `Build` state,
	// server never starts.
	RouteError
)

// SetRegisterRule sets a `RouteRegisterRule` for this Party and its children.
// Available values are: RouteOverride (the default one), RouteSkip and RouteError.
func (api *APIBuilder) SetRegisterRule(rule RouteRegisterRule) Party {
	api.routeRegisterRule = rule
	return api
}

// CreateRoutes returns a list of Party-based Routes.
// It does NOT registers the route. Use `Handle, Get...` methods instead.
// This method can be used for third-parties Iris helpers packages and tools
// that want a more detailed view of Party-based Routes before take the decision to register them.
func (api *APIBuilder) CreateRoutes(methods []string, relativePath string, handlers ...context.Handler) []*Route {
	if len(methods) == 0 || methods[0] == "ALL" || methods[0] == "ANY" { // then use like it was .Any
		return api.Any(relativePath, handlers...)
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

	filename, line := getCaller()

	fullpath := api.relativePath + relativePath // for now, keep the last "/" if any,  "/xyz/"
	if len(handlers) == 0 {
		api.errors.Addf("missing handlers for route[%s:%d] %s: %s", filename, line, strings.Join(methods, ", "), fullpath)
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

	possibleMainHandlerName := context.MainHandlerName(mainHandlers)

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
	methods = append(api.allowMethods, methods...)

	routes := make([]*Route, len(methods))

	for i, m := range methods {
		route, err := NewRoute(m, subdomain, path, possibleMainHandlerName, routeHandlers, *api.macros)
		if err != nil { // template path parser errors:
			api.errors.Addf("[%s:%d] %v -> %s:%s:%s", filename, line, err, m, subdomain, path)
			continue
		}

		route.SourceFileName = filename
		route.SourceLineNumber = line

		// Add UseGlobal & DoneGlobal Handlers
		route.Use(api.beginGlobalHandlers...)
		route.Done(api.doneGlobalHandlers...)

		routes[i] = route
	}

	return routes
}

// https://golang.org/doc/go1.9#callersframes
func getCaller() (string, int) {
	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	wd, _ := os.Getwd()
	for {
		frame, more := frames.Next()
		file := frame.File

		if !strings.Contains(file, "/kataras/iris") || strings.Contains(file, "/kataras/iris/_examples") || strings.Contains(file, "iris-contrib/examples") {
			if relFile, err := filepath.Rel(wd, file); err == nil {
				file = "./" + relFile
			}

			return file, frame.Line
		}

		if !more {
			break
		}
	}

	return "???", 0
}

// Handle registers a route to the server's api.
// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
//
// Returns a *Route, app will throw any errors later on.
func (api *APIBuilder) Handle(method string, relativePath string, handlers ...context.Handler) *Route {
	routes := api.CreateRoutes([]string{method}, relativePath, handlers...)

	var route *Route // the last one is returned.
	var err error
	for _, route = range routes {
		if route == nil {
			break
		}
		// global

		route.topLink = api.routes.getRelative(route)
		if route, err = api.routes.register(route, api.routeRegisterRule); err != nil {
			api.errors.Add(err)
			break
		}
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
// app.HandleMany("GET POST", "/path", handler)
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

// HandleDir registers a handler that serves HTTP requests
// with the contents of a file system (physical or embedded).
//
// first parameter  : the route path
// second parameter : the system or the embedded directory that needs to be served
// third parameter  : not required, the directory options, set fields is optional.
//
// Alternatively, to get just the handler for that look the FileServer function instead.
//
//     api.HandleDir("/static", "./assets",  DirOptions {ShowList: true, Gzip: true, IndexName: "index.html"})
//
// Returns the GET *Route.
//
// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/file-server
func (api *APIBuilder) HandleDir(requestPath, directory string, opts ...DirOptions) (getRoute *Route) {
	options := getDirOptions(opts...)

	h := FileServer(directory, options)

	// if subdomain, we get the full path of the path only,
	// because a subdomain can have parties as well
	// and we need that path to call the `StripPrefix`.
	if _, fullpath := splitSubdomainAndPath(joinPath(api.relativePath, requestPath)); fullpath != "/" {
		h = StripPrefix(fullpath, h)
	}

	requestPath = joinPath(requestPath, WildcardFileParam())
	routes := api.CreateRoutes([]string{http.MethodGet, http.MethodHead}, requestPath, h)
	getRoute = routes[0]
	// we get all index, including sub directories even if those
	// are already managed by the static handler itself.
	staticSites := context.GetStaticSites(directory, getRoute.StaticPath(), options.IndexName)
	for _, s := range staticSites {
		// if the end-dev did manage that index route manually already
		// then skip the auto-registration.
		//
		// Also keep note that end-dev is still able to replace this route and manage by him/herself
		// later on by a simple `Handle/Get/` call, refer to `repository#register`.
		if api.GetRouteByPath(s.RequestPath) != nil {
			continue
		}

		if n := len(api.relativePath); n > 0 && api.relativePath[n-1] == SubdomainPrefix[0] {
			// this api is a subdomain-based.
			slashIdx := strings.IndexByte(s.RequestPath, '/')
			if slashIdx == -1 {
				slashIdx = 0
			}

			requestPath = s.RequestPath[slashIdx:]
		} else {
			requestPath = s.RequestPath[strings.Index(s.RequestPath, api.relativePath)+len(api.relativePath):]
		}

		if requestPath == "" {
			requestPath = "/"
		}

		routes = append(routes, api.CreateRoutes([]string{http.MethodGet}, requestPath, h)...)
		getRoute.StaticSites = append(getRoute.StaticSites, s)
	}

	for _, route := range routes {
		route.MainHandlerName = `HandleDir(directory: "` + directory + `")`
		if _, err := api.routes.register(route, api.routeRegisterRule); err != nil {
			api.errors.Add(err)
			break
		}
	}

	return getRoute
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
		errors:              api.errors,
		// per-party/children
		middleware:            middleware,
		doneHandlers:          api.doneHandlers[0:],
		relativePath:          fullpath,
		allowMethods:          allowMethods,
		handlerExecutionRules: api.handlerExecutionRules,
		routeRegisterRule:     api.routeRegisterRule,
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
		api.errors.Addf("cannot concat parent wildcard subdomain with anything else ->  %s , %s",
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
		api.errors.Addf("cannot concat static subdomain with a dynamic one. Dynamic subdomains should be at the root level -> %s",
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

// GetRouteByPath returns the registered route based on the template path (`Route.Tmpl().Src`).
func (api *APIBuilder) GetRouteByPath(tmplPath string) *Route {
	return api.routes.getByPath(tmplPath)
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

// GetRouteReadOnlyByPath returns the registered read-only route based on the template path (`Route.Tmpl().Src`).
func (api *APIBuilder) GetRouteReadOnlyByPath(tmplPath string) context.RouteReadOnly {
	r := api.GetRouteByPath(tmplPath)
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

// UseGlobal registers handlers that should run at the very beginning.
// It prepends those handler(s) to all routes,
// including all parties, subdomains.
// It doesn't care about call order, it will prepend the handlers to all
// existing routes and the future routes that may being registered.
//
// The difference from `.DoneGlobal` is that this/or these Handler(s) are being always running first.
// Use of `ctx.Next()` of those handler(s) is necessary to call the main handler or the next middleware.
// It's always a good practise to call it right before the `Application#Run` function.
func (api *APIBuilder) UseGlobal(handlers ...context.Handler) {
	for _, r := range api.routes.routes {
		r.Use(handlers...) // prepend the handlers to the existing routes
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
		r.Done(handlers...) // append the handlers to the existing routes
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
	api.routeRegisterRule = RouteOverride
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
		api.errors.Addf("favicon: file or directory %s not found: %w", favPath, err)
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
		api.errors.Addf("favicon: couldn't read the data bytes for %s: %w", favPath, err)
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

// OnErrorCode registers an error http status code
// based on the "statusCode" < 200 || >= 400 (came from `context.StatusCodeNotSuccessful`).
// The handler is being wrapped by a generic
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
// it creates and registers a new trivial handler on the-fly.
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
