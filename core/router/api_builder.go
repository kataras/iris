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
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/macro"
	macroHandler "github.com/kataras/iris/v12/macro/handler"

	"github.com/kataras/golog"
)

// MethodNone is a Virtual method
// to store the "offline" routes.
const MethodNone = "NONE"

// AllMethods contains the valid HTTP Methods:
// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD",
// "PATCH", "OPTIONS", "TRACE".
var AllMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPatch,
	http.MethodPut,
	http.MethodPost,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodConnect,
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
		if r.Subdomain == route.Subdomain && r.StatusCode == route.StatusCode && r.Method == route.Method && r.FormattedPath == route.FormattedPath && !route.tmpl.IsTrailing() {
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
			} else if rule == RouteOverlap {
				overlapRoute(r, route)
				return route, nil
			} else {
				// replace existing with the latest one, the default behavior.
				repo.routes = append(repo.routes[:i], repo.routes[i+1:]...)
			}

			continue
		}
	}

	// fmt.Printf("repo.routes append:\t%#+v\n\n", route)
	repo.routes = append(repo.routes, route)

	if route.StatusCode == 0 { // a common resource route, not a status code error handler.
		if repo.pos == nil {
			repo.pos = make(map[string]int)
		}
		repo.pos[route.tmpl.Src] = len(repo.routes) - 1
	}

	return route, nil
}

var defaultOverlapFilter = func(ctx *context.Context) bool {
	if ctx.IsStopped() {
		// It's stopped and the response can be overridden by a new handler.
		rs, ok := ctx.ResponseWriter().(context.ResponseWriterReseter)
		return ok && rs.Reset()
	}

	// It's not stopped, all OK no need to execute the alternative route.
	return false
}

func overlapRoute(r *Route, next *Route) {
	next.BuildHandlers()
	nextHandlers := next.Handlers[0:]

	decisionHandler := func(ctx *context.Context) {
		ctx.Next()

		if !defaultOverlapFilter(ctx) {
			return
		}

		ctx.SetErr(nil) // clear any stored error.
		// Set the route to the next one and execute it.
		ctx.SetCurrentRoute(next.ReadOnly)
		ctx.HandlerIndex(0)
		ctx.Do(nextHandlers)
	}

	// NOTE(@kataras): Any UseGlobal call will prepend to this, if they are
	// in the same Party then it's expected, otherwise not.
	r.beginHandlers = append(context.Handlers{decisionHandler}, r.beginHandlers...)
}

// APIBuilder the visible API for constructing the router
// and child routers.
type APIBuilder struct {
	// the application logger.
	logger *golog.Logger
	// parent is the creator of this Party.
	// It is nil on Root.
	parent *APIBuilder // currently it's used only on UseRouter feature.

	// the per-party APIBuilder with DI.
	apiBuilderDI *APIContainer

	// the api builder global macros registry
	macros *macro.Macros
	// the api builder global routes repository
	routes *repository

	// the api builder global errors, can be filled by the Subdomain, WildcardSubdomain, Handle...
	// the list of possible errors that can be
	// collected on the build state to log
	// to the end-user.
	errors *errgroup.Group

	// the per-party handlers, order
	// of handlers registration matters.
	middleware          context.Handlers
	middlewareErrorCode context.Handlers
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

	// the per-party relative path.
	relativePath string
	// allowMethods are filled with the `AllowMethods` method.
	// They are used to create new routes
	// per any party's (and its children) routes registered
	// if the method "x" wasn't registered already via  the `Handle` (and its extensions like `Get`, `Post`...).
	allowMethods []string

	// the per-party (and its children) execution rules for begin, main and done handlers.
	handlerExecutionRules ExecutionRules
	// the per-party (and its children) route registration rule, see `SetRegisterRule`.
	routeRegisterRule RouteRegisterRule

	// routerFilters field is shared across Parties. Each Party registers
	// one or more middlewares to run before the router itself using the `UseRouter` method.
	// Each Party calls the shared filter (`partyMatcher`) that decides if its `UseRouter` handlers
	// can be executed. By default it's based on party's static path and/or subdomain,
	// it can be modified through an `Application.SetPartyMatcher` call
	// once before or after routerFilters filled.
	//
	// The Key is the Party (instance of APIBuilder),
	// value wraps the partyFilter + the handlers registered through `UseRouter`.
	// See `GetRouterFilters` too.
	routerFilters map[Party]*Filter
	// partyMatcher field is shared across all Parties,
	// can be modified through the Application level only.
	//
	// It defaults to the internal, simple, "defaultPartyMatcher".
	// It applies when "routerFilters" are used.
	partyMatcher PartyMatcherFunc
}

var (
	_ Party          = (*APIBuilder)(nil)
	_ PartyMatcher   = (*APIBuilder)(nil)
	_ RoutesProvider = (*APIBuilder)(nil) // passed to the default request handler (routerHandler)
)

// NewAPIBuilder creates & returns a new builder
// which is responsible to build the API and the router handler.
func NewAPIBuilder(logger *golog.Logger) *APIBuilder {
	return &APIBuilder{
		logger:        logger,
		parent:        nil,
		macros:        macro.Defaults,
		errors:        errgroup.New("API Builder"),
		relativePath:  "/",
		routes:        new(repository),
		apiBuilderDI:  &APIContainer{Container: hero.New()},
		routerFilters: make(map[Party]*Filter),
		partyMatcher:  defaultPartyMatcher,
	}
}

// Logger returns the Application Logger.
func (api *APIBuilder) Logger() *golog.Logger {
	return api.logger
}

// IsRoot reports whether this Party is the root Application's one.
// It will return false on all children Parties, no exception.
func (api *APIBuilder) IsRoot() bool {
	return api.parent == nil
}

/* If requested:
// GetRoot returns the very first Party (the Application).
func (api *APIBuilder) GetRoot() *APIBuilder {
	root := api.parent
	for root != nil {
		root = api.parent
	}

	return root
}*/

// ConfigureContainer accepts one or more functions that can be used
// to configure dependency injection features of this Party
// such as register dependency and register handlers that will automatically inject any valid dependency.
// However, if the "builder" parameter is nil or not provided then it just returns the *APIContainer,
// which automatically initialized on Party allocation.
//
// It returns the same `APIBuilder` featured with Dependency Injection.
func (api *APIBuilder) ConfigureContainer(builder ...func(*APIContainer)) *APIContainer {
	if api.apiBuilderDI.Self == nil {
		api.apiBuilderDI.Self = api
	}

	for _, b := range builder {
		if b != nil {
			b(api.apiBuilderDI)
		}
	}

	return api.apiBuilderDI
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
	// RouteOverride replaces an existing route with the new one, the default rule.
	RouteOverride RouteRegisterRule = iota
	// RouteSkip keeps the original route and skips the new one.
	RouteSkip
	// RouteError log when a route already exists, shown after the `Build` state,
	// server never starts.
	RouteError
	// RouteOverlap will overlap the new route to the previous one.
	// If the route stopped and its response can be reset then the new route will be execute.
	RouteOverlap
)

// SetRegisterRule sets a `RouteRegisterRule` for this Party and its children.
// Available values are:
// * RouteOverride (the default one)
// * RouteSkip
// * RouteError
// * RouteOverlap.
func (api *APIBuilder) SetRegisterRule(rule RouteRegisterRule) Party {
	api.routeRegisterRule = rule
	return api
}

// Handle registers a route to this Party.
// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
//
// Returns a *Route, app will throw any errors later on.
func (api *APIBuilder) Handle(method string, relativePath string, handlers ...context.Handler) *Route {
	return api.handle(0, method, relativePath, handlers...)
}

// handle registers a full route to this Party.
// Use Handle or Get, Post, Put, Delete and et.c. instead.
func (api *APIBuilder) handle(errorCode int, method string, relativePath string, handlers ...context.Handler) *Route {
	routes := api.createRoutes(errorCode, []string{method}, relativePath, handlers...)

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
// second parameter : the file system needs to be served
// third parameter  : not required, the serve directory options.
//
// Alternatively, to get just the handler for that look the FileServer function instead.
//
//     api.HandleDir("/static", iris.Dir("./assets"), iris.DirOptions{IndexName: "/index.html", Compress: true})
//
// Returns all the registered routes, including GET index and path patterm and HEAD.
//
// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/file-server
func (api *APIBuilder) HandleDir(requestPath string, fs http.FileSystem, opts ...DirOptions) (routes []*Route) {
	options := DefaultDirOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	h := FileServer(fs, options)
	description := "file server"
	if d, ok := fs.(http.Dir); ok {
		description = string(d)
	}

	fileName, lineNumber := context.HandlerFileLine(h) // take those before StripPrefix.

	// if subdomain, we get the full path of the path only,
	// because a subdomain can have parties as well
	// and we need that path to call the `StripPrefix`.
	_, fullpath := splitSubdomainAndPath(joinPath(api.relativePath, requestPath))
	if fullpath != "/" {
		h = StripPrefix(fullpath, h)
	}

	if api.GetRouteByPath(fullpath) == nil {
		// register index if not registered by the end-developer.
		routes = api.CreateRoutes([]string{http.MethodGet, http.MethodHead}, requestPath, h)
	}

	requestPath = joinPath(requestPath, WildcardFileParam())

	routes = append(routes, api.CreateRoutes([]string{http.MethodGet, http.MethodHead}, requestPath, h)...)

	for _, route := range routes {
		if route.Method == http.MethodHead {
		} else {
			route.Describe(description)
			route.SetSourceLine(fileName, lineNumber)
		}

		if _, err := api.routes.register(route, api.routeRegisterRule); err != nil {
			api.errors.Add(err)
			break
		}
	}

	return routes
}

// CreateRoutes returns a list of Party-based Routes.
// It does NOT registers the route. Use `Handle, Get...` methods instead.
// This method can be used for third-parties Iris helpers packages and tools
// that want a more detailed view of Party-based Routes before take the decision to register them.
func (api *APIBuilder) CreateRoutes(methods []string, relativePath string, handlers ...context.Handler) []*Route {
	return api.createRoutes(0, methods, relativePath, handlers...)
}

func (api *APIBuilder) createRoutes(errorCode int, methods []string, relativePath string, handlers ...context.Handler) []*Route {
	if statusCodeSuccessful(errorCode) {
		errorCode = 0
	}

	if errorCode == 0 {
		if len(methods) == 0 || methods[0] == "ALL" || methods[0] == "ANY" { // then use like it was .Any
			return api.Any(relativePath, handlers...)
		}
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
		// global middleware to error handlers as well.
		beginHandlers = api.beginGlobalHandlers
		doneHandlers  = api.doneGlobalHandlers
	)

	if errorCode == 0 {
		beginHandlers = context.JoinHandlers(beginHandlers, api.middleware)
		doneHandlers = context.JoinHandlers(doneHandlers, api.doneHandlers)
	} else {
		beginHandlers = context.JoinHandlers(beginHandlers, api.middlewareErrorCode)
	}

	mainHandlers := context.Handlers(handlers)
	// before join the middleware + handlers + done handlers and apply the execution rules.

	mainHandlerName, mainHandlerIndex := context.MainHandlerName(mainHandlers)

	mainHandlerFileName, mainHandlerFileNumber := context.HandlerFileLineRel(handlers[mainHandlerIndex])

	// re-calculate mainHandlerIndex in favor of the middlewares.
	mainHandlerIndex = len(beginHandlers) + mainHandlerIndex

	// TODO: for UseGlobal/DoneGlobal that doesn't work.
	applyExecutionRules(api.handlerExecutionRules, &beginHandlers, &doneHandlers, &mainHandlers)

	// global begin handlers -> middleware that are registered before route registration
	// -> handlers that are passed to this Handle function.
	routeHandlers := context.JoinHandlers(beginHandlers, mainHandlers)
	// -> done handlers
	routeHandlers = context.JoinHandlers(routeHandlers, doneHandlers)

	// here we separate the subdomain and relative path
	subdomain, path := splitSubdomainAndPath(fullpath)

	// if allowMethods are empty, then simply register with the passed, main, method.
	methods = removeDuplicates(append(api.allowMethods, methods...))

	routes := make([]*Route, len(methods))

	for i, m := range methods { // single, empty method for error handlers.
		route, err := NewRoute(errorCode, m, subdomain, path, routeHandlers, *api.macros)
		if err != nil { // template path parser errors:
			api.errors.Addf("[%s:%d] %v -> %s:%s:%s", filename, line, err, m, subdomain, path)
			continue
		}

		// The caller tiself, if anonymous, it's the first line of `app.X("/path", here)`
		route.RegisterFileName = filename
		route.RegisterLineNumber = line

		route.MainHandlerName = mainHandlerName
		route.MainHandlerIndex = mainHandlerIndex

		// The main handler source, could be the same as the register's if anonymous.
		route.SourceFileName = mainHandlerFileName
		route.SourceLineNumber = mainHandlerFileNumber

		// Add UseGlobal & DoneGlobal Handlers
		// route.Use(api.beginGlobalHandlers...)
		// route.Done(api.doneGlobalHandlers...)

		routes[i] = route
	}

	return routes
}

func removeDuplicates(elements []string) (result []string) {
	seen := make(map[string]struct{})

	for v := range elements {
		val := elements[v]
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

// Party returns a new child Party which inherites its
// parent's options and middlewares.
// If "relativePath" matches the parent's one then it returns the current Party.
// A Party groups routes which may have the same prefix or subdomain and share same middlewares.
//
// To create a group of routes for subdomains
// use the `Subdomain` or `WildcardSubdomain` methods
// or pass a "relativePath" as "admin." or "*." respectfully.
func (api *APIBuilder) Party(relativePath string, handlers ...context.Handler) Party {
	// if app.Party("/"), root party or app.Party("/user") == app.Party("/user")
	// then just add the middlewares and return itself.
	if relativePath == "" || api.relativePath == relativePath {
		api.Use(handlers...)
		return api
	}

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
	middleware := context.JoinHandlers(api.middleware, handlers)

	// the allow methods per party and its children.
	allowMethods := make([]string, len(api.allowMethods))
	copy(allowMethods, api.allowMethods)

	childAPI := &APIBuilder{
		// global/api builder
		logger:              api.logger,
		macros:              api.macros,
		routes:              api.routes,
		beginGlobalHandlers: api.beginGlobalHandlers,
		doneGlobalHandlers:  api.doneGlobalHandlers,
		errors:              api.errors,
		routerFilters:       api.routerFilters, // shared.
		partyMatcher:        api.partyMatcher,  // shared.
		// per-party/children
		parent:                api,
		middleware:            middleware,
		middlewareErrorCode:   context.JoinHandlers(api.middlewareErrorCode, context.Handlers{}),
		doneHandlers:          api.doneHandlers[0:],
		relativePath:          fullpath,
		allowMethods:          allowMethods,
		handlerExecutionRules: api.handlerExecutionRules,
		routeRegisterRule:     api.routeRegisterRule,
		apiBuilderDI: &APIContainer{
			// attach a new Container with correct dynamic path parameter start index for input arguments
			// based on the fullpath.
			Container: api.apiBuilderDI.Container.Clone(),
		},
	}

	return childAPI
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
		readOnlyRoutes[i] = r.ReadOnly
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
	return r.ReadOnly
}

// GetRouteReadOnlyByPath returns the registered read-only route based on the template path (`Route.Tmpl().Src`).
func (api *APIBuilder) GetRouteReadOnlyByPath(tmplPath string) context.RouteReadOnly {
	r := api.GetRouteByPath(tmplPath)
	if r == nil {
		return nil
	}

	return r.ReadOnly
}

type (
	// PartyMatcherFunc used to build a filter which decides
	// if the given Party is responsible to fire its `UseRouter` handlers or not.
	// Can be customized through `SetPartyMatcher` method. See `Match` method too.
	PartyMatcherFunc func(Party, *context.Context) bool
	// PartyMatcher decides if `UseRouter` handlers should be executed or not.
	// A different interface becauwe we want to separate
	// the Party's public API from `UseRouter` internals.
	PartyMatcher interface {
		Match(ctx *context.Context) bool
	}
	// Filter is a wraper for a Router Filter contains information
	// for its Party's fullpath, subdomain the Party's
	// matcher and the associated handlers to be executed before main router's request handler.
	Filter struct {
		Party     Party        // the Party itself
		Matcher   PartyMatcher // it's a Party, for freedom that can be changed through a custom matcher which accepts the same filter.
		Subdomain string
		Path      string
		Handlers  context.Handlers
	}
)

// SetPartyMatcher accepts a function which runs against
// a Party and should report whether its `UseRouter` handlers should be executed.
// PartyMatchers are run through parent to children.
// It modifies the default Party filter that decides
// which `UseRouter` middlewares to run before the Router,
// each one of those middlewares can skip `Context.Next` or call `Context.StopXXX`
// to stop the main router from searching for a route match.
// Can be called before or after `UseRouter`, it doesn't matter.
func (api *APIBuilder) SetPartyMatcher(matcherFunc PartyMatcherFunc) {
	if matcherFunc == nil {
		matcherFunc = defaultPartyMatcher
	}
	api.partyMatcher = matcherFunc
}

// Match reports whether the `UseRouter` handlers should be executed.
// Calls its parent's Match if possible.
// Implements the `PartyMatcher` interface.
func (api *APIBuilder) Match(ctx *context.Context) bool {
	return api.partyMatcher(api, ctx)
}

func defaultPartyMatcher(p Party, ctx *context.Context) bool {
	subdomain, path := splitSubdomainAndPath(p.GetRelPath())
	staticPath := staticPath(path)
	hosts := subdomain != ""

	if p.IsRoot() {
		// ALWAYS executed first when registered
		// through an `Application.UseRouter` call.
		return true
	}

	if hosts {
		// Note(@kataras): do NOT try to implement something like party matcher for each party
		// separately. We will introduce a new problem with subdomain inside a subdomain:
		// they are not by prefix, so parenting calls will not help
		// e.g. admin. and control.admin, control.admin is a sub of the admin.
		if !canHandleSubdomain(ctx, subdomain) {
			return false
		}
	}

	// this is the longest static path.
	return strings.HasPrefix(ctx.Path(), staticPath)
}

// GetRouterFilters returns the global router filters.
// Read `UseRouter` for more.
// The map can be altered before router built.
// The router internally prioritized them by the subdomains and
// longest static path.
// Implements the `RoutesProvider` interface.
func (api *APIBuilder) GetRouterFilters() map[Party]*Filter {
	return api.routerFilters
}

// UseRouter upserts one or more handlers that will be fired
// right before the main router's request handler.
//
// Use this method to register handlers, that can ran
// independently of the incoming request's values,
// that they will be executed ALWAYS against ALL children incoming requests.
// Example of use-case: CORS.
//
// Note that because these are executed before the router itself
// the Context should not have access to the `GetCurrentRoute`
// as it is not decided yet which route is responsible to handle the incoming request.
// It's one level higher than the `WrapRouter`.
// The context SHOULD call its `Next` method in order to proceed to
// the next handler in the chain or the main request handler one.
func (api *APIBuilder) UseRouter(handlers ...context.Handler) {
	if len(handlers) == 0 {
		return
	}

	beginHandlers := context.Handlers(handlers)
	// respect any execution rules (begin).
	api.handlerExecutionRules.Begin.apply(&beginHandlers)

	if f := api.routerFilters[api]; f != nil && len(f.Handlers) > 0 { // exists.
		beginHandlers = context.UpsertHandlers(f.Handlers, beginHandlers) // remove dupls.
	} else {
		// Note(@kataras): we don't add the parent's filter handlers
		// on `Party` method because we need to know if a `UseRouter` call exist
		// before prepending the parent's ones and fill a new Filter on `routerFilters`,
		// that key should NOT exist on a Party without `UseRouter` handlers (see router.go).
		// That's the only reason we need the `parent` field.
		if api.parent != nil {
			// If it's not root, add the parent's handlers here.
			if root, ok := api.routerFilters[api.parent]; ok {
				beginHandlers = context.UpsertHandlers(root.Handlers, beginHandlers)
			}
		}
	}

	subdomain, path := splitSubdomainAndPath(api.relativePath)
	api.routerFilters[api] = &Filter{
		Matcher:   api,
		Subdomain: subdomain,
		Path:      path,
		Handlers:  beginHandlers,
	}
}

// GetDefaultErrorMiddleware returns the application's error pre handlers
// registered through `UseError` for the default error handlers.
// This is used when no matching error handlers registered
// for a specific status code but `UseError` is called to register a middleware,
// so the default error handler should make use of those middleware now.
func (api *APIBuilder) GetDefaultErrorMiddleware() context.Handlers {
	return api.middlewareErrorCode
}

// UseError upserts one or more handlers that will be fired,
// as middleware, before any error handler registered through `On(Any)ErrorCode`.
// See `OnErrorCode` too.
func (api *APIBuilder) UseError(handlers ...context.Handler) {
	api.middlewareErrorCode = context.UpsertHandlers(api.middlewareErrorCode, handlers)
}

// Use appends Handler(s) to the current Party's routes and child routes.
// If the current Party is the root, then it registers the middleware to all child Parties' routes too.
//
// Call order matters, it should be called right before the routes that they care about these handlers.
//
// If it's called after the routes then these handlers will never be executed.
// Use `UseGlobal` if you want to register begin handlers(middleware)
// that should be always run before all application's routes.
// To register a middleware for error handlers, look `UseError` method instead.
func (api *APIBuilder) Use(handlers ...context.Handler) {
	api.middleware = append(api.middleware, handlers...)
}

// UseOnce either inserts a middleware,
// or on the basis of the middleware already existing,
// replace that existing middleware instead.
// To register a middleware for error handlers, look `UseError` method instead.
func (api *APIBuilder) UseOnce(handlers ...context.Handler) {
	api.middleware = context.UpsertHandlers(api.middleware, handlers)
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
		// r.beginHandlers = append(handlers, r.beginHandlers...)
		// ^ this is correct but we act global begin handlers as one chain, so
		// if called last more than one time, after all routes registered, we must somehow
		// register them by order, so:
		r.Use(handlers...)
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
	api.middlewareErrorCode = api.middlewareErrorCode[0:0]
	api.doneHandlers = api.doneHandlers[0:0]
	api.handlerExecutionRules = ExecutionRules{}
	api.routeRegisterRule = RouteOverride
	// keep container as it's.
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

// Get registers a route for the Get HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Get(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodGet, relativePath, handlers...)
}

// Post registers a route for the Post HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Post(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPost, relativePath, handlers...)
}

// Put registers a route for the Put HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Put(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPut, relativePath, handlers...)
}

// Delete registers a route for the Delete HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Delete(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodDelete, relativePath, handlers...)
}

// Connect registers a route for the Connect HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Connect(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodConnect, relativePath, handlers...)
}

// Head registers a route for the Head HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Head(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodHead, relativePath, handlers...)
}

// Options registers a route for the Options HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Options(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodOptions, relativePath, handlers...)
}

// Patch registers a route for the Patch HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Patch(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodPatch, relativePath, handlers...)
}

// Trace registers a route for the Trace HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIBuilder) Trace(relativePath string, handlers ...context.Handler) *Route {
	return api.Handle(http.MethodTrace, relativePath, handlers...)
}

// Any registers a route for ALL of the HTTP methods:
// Get
// Post
// Put
// Delete
// Head
// Patch
// Options
// Connect
// Trace
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
	h := func(ctx *context.Context) {
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
	description := favPath
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
	h := func(ctx *context.Context) {
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

	return api.registerResourceRoute(reqPath, h).Describe(description)
}

// OnErrorCode registers a handlers chain for this `Party` for a specific HTTP status code.
// Read more at: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
// Look `UseError` and `OnAnyErrorCode` too.
func (api *APIBuilder) OnErrorCode(statusCode int, handlers ...context.Handler) (routes []*Route) {
	routes = append(routes, api.handle(statusCode, "", "/", handlers...))

	if api.relativePath != "/" {
		routes = append(routes, api.handle(statusCode, "", "/{tail:path}", handlers...))
	}

	return
}

// OnAnyErrorCode registers a handlers chain for all error codes
// (4xxx and 5xxx, change the `context.ClientErrorCodes` and `context.ServerErrorCodes` variables to modify those)
// Look `UseError` and `OnErrorCode` too.
func (api *APIBuilder) OnAnyErrorCode(handlers ...context.Handler) (routes []*Route) {
	for _, statusCode := range context.ClientAndServerErrorCodes {
		routes = append(routes, api.OnErrorCode(statusCode, handlers...)...)
	}

	if n := len(routes); n > 1 {
		for _, r := range routes[1:n] {
			r.NoLog = true
		}

		routes[0].Title = "ERR"
	}

	return
}

// RegisterView registers and loads a view engine middleware for that group of routes.
// It overrides any of the application's root registered view engines.
// To register a view engine per handler chain see the `Context.ViewEngine` instead.
// Read `Configuration.ViewEngineContextKey` documentation for more.
func (api *APIBuilder) RegisterView(viewEngine context.ViewEngine) {
	if err := viewEngine.Load(); err != nil {
		api.errors.Add(err)
		return
	}

	handler := func(ctx *context.Context) {
		ctx.ViewEngine(viewEngine)
		ctx.Next()
	}
	api.Use(handler)
	api.UseError(handler)
	// Note (@kataras): It does not return the Party in order
	// to keep the iris.Application a compatible Party.
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
	handler := func(ctx *context.Context) {
		ctx.ViewLayout(tmplLayoutFile)
		ctx.Next()
	}

	api.Use(handler)
	api.UseError(handler)

	return api
}

// https://golang.org/doc/go1.9#callersframes
func getCaller() (string, int) {
	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	wd, _ := os.Getwd()

	var (
		frame runtime.Frame
		more  = true
	)

	for {
		if !more {
			break
		}

		frame, more = frames.Next()
		file := filepath.ToSlash(frame.File)
		// fmt.Printf("%s:%d | %s\n", file, frame.Line, frame.Function)

		if strings.Contains(file, "go/src/runtime/") {
			continue
		}

		if !strings.Contains(file, "_test.go") {
			if strings.Contains(file, "/kataras/iris") &&
				!strings.Contains(file, "kataras/iris/_examples") &&
				!strings.Contains(file, "kataras/iris/middleware") &&
				!strings.Contains(file, "iris-contrib/examples") {
				continue
			}
		}

		if relFile, err := filepath.Rel(wd, file); err == nil {
			if !strings.HasPrefix(relFile, "..") {
				// Only if it's relative to this path, not parent.
				file = "./" + relFile
			}
		}

		return file, frame.Line
	}

	return "???", 0
}
