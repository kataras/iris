package router

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
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

// RegisterMethods adds custom http methods to the "AllMethods" list.
// Use it on initialization of your program.
func RegisterMethods(newCustomHTTPVerbs ...string) {
	newMethods := append(AllMethods, newCustomHTTPVerbs...)
	AllMethods = removeDuplicates(newMethods)
}

// repository passed to all parties(subrouters), it's the object witch keeps
// all the routes.
type repository struct {
	routes []*Route
	paths  map[string]*Route // only the fullname path part, required at CreateRoutes for registering index page.
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
		if r.tmpl.Src == route.tmpl.Src { // No topLink on the same route syntax.
			// Fixes #2008, because of APIBuilder.handle, repo.getRelative and repo.register replacement but with a toplink of the old route.
			continue
		}

		if r.Subdomain == route.Subdomain && r.StatusCode == route.StatusCode && r.Method == route.Method &&
			r.FormattedPath == route.FormattedPath && !route.tmpl.IsTrailing() {
			return route
		}
	}

	return nil
}

func (repo *repository) getByPath(tmplPath string) *Route {
	if r, ok := repo.paths[tmplPath]; ok {
		return r
	}

	return nil
}

func (repo *repository) getAll() []*Route {
	return repo.routes
}

func (repo *repository) remove(routeName string) bool {
	for i, r := range repo.routes {
		if r.Name == routeName {
			lastIdx := len(repo.routes) - 1
			if lastIdx == i {
				repo.routes = repo.routes[0:lastIdx]
			} else {
				cp := make([]*Route, 0, lastIdx)
				cp = append(cp, repo.routes[:i]...)
				repo.routes = append(cp, repo.routes[i+1:]...)
			}

			delete(repo.paths, r.tmpl.Src)
			return true
		}
	}
	return false
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

			break // continue
		}
	}

	repo.routes = append(repo.routes, route)

	if route.StatusCode == 0 { // a common resource route, not a status code error handler.
		if repo.paths == nil {
			repo.paths = make(map[string]*Route)
		}
		repo.paths[route.tmpl.Src] = route
	}

	return route, nil
}

var defaultOverlapFilter = func(ctx *context.Context) bool {
	if ctx.IsStopped() {
		// It's stopped and the response can be overridden by a new handler.
		// An exception of compress writer, which does not implement Reseter (and it shouldn't):
		rs, ok := ctx.ResponseWriter().(context.ResponseWriterReseter)
		return ok && rs.Reset()
	}

	// It's not stopped, all OK no need to execute the alternative route.
	return false
}

func overlapRoute(r *Route, next *Route) {
	next.BuildHandlers()
	nextHandlers := next.Handlers[0:]

	isErrorRoutes := r.StatusCode > 0 && next.StatusCode > 0

	decisionHandler := func(ctx *context.Context) {
		ctx.Next()

		if isErrorRoutes { // fixes issue #1602.
			// If it's an error we don't need to reset (see defaultOverlapFilter)
			// its status code(!) and its body, we just check if it was proceed or not.
			if !ctx.IsStopped() {
				return
			}
		} else {
			prevStatusCode := ctx.GetStatusCode()

			if !defaultOverlapFilter(ctx) {
				return
			}
			// set the status code that it was stopped with.
			// useful for dependencies with StopWithStatus(XXX)
			// instead of raw ctx.StopExecution().
			// The func_result now also catch the last registered status code
			// of the chain, unless the controller returns an integer.
			// See _examples/authenticated-controller.
			if prevStatusCode > 0 {
				// An exception when stored error
				// exists and it's type of ErrNotFound.
				// Example:
				// Version was not found:
				//	 we need to be able to send the status on the last not found version
				//   but reset the status code if a next available matched version was found.
				//	 see the versioning package.
				if !errors.Is(ctx.GetErr(), context.ErrNotFound) {
					ctx.StatusCode(prevStatusCode)
				}
			}
		}

		ctx.SetErr(nil) // clear any stored error.
		// Set the route to the next one and execute it.
		ctx.SetCurrentRoute(next.ReadOnly)
		ctx.HandlerIndex(0)
		ctx.Do(nextHandlers)
	}

	r.builtinBeginHandlers = append(context.Handlers{decisionHandler}, r.builtinBeginHandlers...)
	r.overlappedLink = next
}

// APIBuilder the visible API for constructing the router
// and child routers.
type APIBuilder struct {
	// the application logger.
	logger *golog.Logger
	// parent is the creator of this Party.
	// It is nil on Root.
	parent *APIBuilder // currently it's not used anywhere.

	// the per-party APIBuilder with DI.
	apiBuilderDI *APIContainer

	// the api builder global macros registry
	macros *macro.Macros
	// the per-party (and its children) values map
	// that may help on building the API
	// when source code is splitted between projects.
	// Initialized on Properties method.
	properties context.Map
	// the api builder global routes repository
	routes *repository
	// disables the debug logging of routes under a per-party and its children.
	routesNoLog bool

	// the per-party handlers, order
	// of handlers registration matters,
	// inherited by children unless Reset is called.
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

	// routerFilterHandlers holds a reference
	// of the handlers used by the current and its parent Party's registered
	// router filters. Inherited by children unless `Reset` (see `UseRouter`),
	routerFilterHandlers context.Handlers
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
		relativePath:  "/",
		routes:        new(repository),
		apiBuilderDI:  &APIContainer{Container: hero.New().WithLogger(logger)},
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

// EnsureStaticBindings panics on struct handler (controller)
// if at least one input binding depends on the request and not in a static structure.
// Should be called before `RegisterDependency`.
func (api *APIBuilder) EnsureStaticBindings() Party {
	diContainer := api.ConfigureContainer()
	diContainer.Container.DisableStructDynamicBindings = true
	return api
}

// RegisterDependency calls the `ConfigureContainer.RegisterDependency` method
// with the provided value(s). See `HandleFunc` and `PartyConfigure` methods too.
func (api *APIBuilder) RegisterDependency(dependencies ...interface{}) {
	diContainer := api.ConfigureContainer()
	for i, dependency := range dependencies {
		if dependency == nil {
			api.logger.Warnf("Party: %s: nil dependency on position: %d", api.relativePath, i)
			continue
		}

		diContainer.RegisterDependency(dependency)
	}
}

// HandleFunc registers a route on HTTP verb "method" and relative, to this Party, path.
// It is like the `Handle` method but it accepts one or more "handlersFn" functions
// that each one of them can accept any input arguments as the HTTP request and
// output a result as the HTTP response. Specifically,
// the input of the "handlersFn" can be any registered dependency
// (see ConfigureContainer().RegisterDependency)
// or leave the framework to parse the request and fill the values accordingly.
// The output of the "handlersFn" can be any output result:
//
//	custom structs <T>, string, []byte, int, error,
//	a combination of the above, hero.Result(hero.View | hero.Response) and more.
//
// If more than one handler function is registered
// then the execution happens without the nessecity of the `Context.Next` method,
// simply, to stop the execution and not continue to the next "handlersFn" in chain
// you should return an `iris.ErrStopExecution`.
//
// Example Code:
//
// The client's request body and server's response body Go types.
// Could be any data structure.
//
//	type (
//		request struct {
//			Firstname string `json:"firstname"`
//			Lastname string `json:"lastname"`
//		}
//
//		response struct {
//			ID uint64 `json:"id"`
//			Message string `json:"message"`
//		}
//	)
//
// Register the route hander.
//
//	            HTTP VERB    ROUTE PATH       ROUTE HANDLER
//	app.HandleFunc("PUT", "/users/{id:uint64}", updateUser)
//
// Code the route handler function.
// Path parameters and request body are binded
// automatically.
// The "id" uint64 binds to "{id:uint64}" route path parameter and
// the "input" binds to client request data such as JSON.
//
//	func updateUser(id uint64, input request) response {
//		// [custom logic...]
//
//		return response{
//			ID:id,
//			Message: "User updated successfully",
//		}
//	}
//
// Simulate a client request which sends data
// to the server and prints out the response.
//
//	curl --request PUT -d '{"firstname":"John","lastname":"Doe"}' \
//	-H "Content-Type: application/json" \
//	http://localhost:8080/users/42
//
//	{
//		"id": 42,
//		"message": "User updated successfully"
//	}
//
// See the `ConfigureContainer` for more features regrading
// the dependency injection, mvc and function handlers.
//
// This method is just a shortcut of the `ConfigureContainer().Handle`.
func (api *APIBuilder) HandleFunc(method, relativePath string, handlersFn ...interface{}) *Route {
	return api.ConfigureContainer().Handle(method, relativePath, handlersFn...)
}

// UseFunc registers a function which can accept one or more
// dependencies (see RegisterDependency) and returns an iris.Handler
// or a result of <T> and/or an error.
//
// This method is just a shortcut of the `ConfigureContainer().Use`.
func (api *APIBuilder) UseFunc(handlersFn ...interface{}) {
	api.ConfigureContainer().Use(handlersFn...)
}

// GetRelPath returns the current party's relative path.
// i.e:
// if r := app.Party("/users"), then the `r.GetRelPath()` is the "/users".
// if r := app.Party("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
func (api *APIBuilder) GetRelPath() string {
	return api.relativePath
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
//
//	Party#SetExecutionRules(iris.ExecutionRules {
//	  Begin: iris.ExecutionOptions{Force: true},
//	  Main:  iris.ExecutionOptions{Force: true},
//	  Done:  iris.ExecutionOptions{Force: true},
//	})
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
// Example: https://github.com/kataras/iris/tree/main/_examples/mvc/middleware/without-ctx-next
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
	if relativePath == "" {
		relativePath = "/"
	}

	routes := api.createRoutes(errorCode, []string{method}, relativePath, handlers...)

	var route *Route // the last one is returned.
	var err error
	for _, route = range routes {
		if route == nil {
			continue
		}

		// global

		route.topLink = api.routes.getRelative(route)
		if route, err = api.routes.register(route, api.routeRegisterRule); err != nil {
			api.logger.Error(err)
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
//
//	app.HandleMany("GET", "/user /user/{id:uint64} /user/me", genericUserHandler)
//
// At the other side, with `Handle` we've had to write:
//
//	app.Handle("GET", "/user", userHandler)
//	app.Handle("GET", "/user/{id:uint64}", userByIDHandler)
//	app.Handle("GET", "/user/me", userMeHandler)
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
//	api.HandleDir("/static", iris.Dir("./assets"), iris.DirOptions{IndexName: "/index.html", Compress: true})
//
// Returns all the registered routes, including GET index and path patterm and HEAD.
//
// Usage:
// HandleDir("/public", "./assets", DirOptions{...}) or
// HandleDir("/public", iris.Dir("./assets"), DirOptions{...})
// OR
// //go:embed assets/*
// var filesystem embed.FS
// HandleDir("/public",filesystem, DirOptions{...})
// OR to pick a specific folder of the embedded filesystem:
// import "io/fs"
// subFilesystem, err := fs.Sub(filesystem, "assets")
// HandleDir("/public",subFilesystem, DirOptions{...})
//
// Examples:
// https://github.com/kataras/iris/tree/main/_examples/file-server
func (api *APIBuilder) HandleDir(requestPath string, fsOrDir interface{}, opts ...DirOptions) (routes []*Route) {
	options := DefaultDirOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	fs := context.ResolveHTTPFS(fsOrDir)
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
			api.logger.Error(err)
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

// RemoveRoute deletes a registered route by its name before `Application.Listen`.
// The default naming for newly created routes is: method + subdomain + path.
// Reports whether a route with that name was found and removed successfully.
//
// Note that this method applies to all Parties (sub routers)
// even if each of the Parties have access to this method,
// as the route name is unique per Iris Application.
func (api *APIBuilder) RemoveRoute(routeName string) bool {
	return api.routes.remove(routeName)
}

func (api *APIBuilder) createRoutes(errorCode int, methods []string, relativePath string, handlers ...context.Handler) []*Route {
	if statusCodeSuccessful(errorCode) {
		errorCode = 0
	}

	mainHandlers := context.CopyHandlers(handlers)

	if errorCode == 0 {
		if len(methods) == 0 || methods[0] == "ALL" || methods[0] == "ANY" { // then use like it was .Any
			return api.Any(relativePath, mainHandlers...)
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

	filename, line := hero.GetCaller()

	fullpath := api.relativePath + relativePath // for now, keep the last "/" if any,  "/xyz/"
	if len(mainHandlers) == 0 {
		api.logger.Errorf("missing handlers for route[%s:%d] %s: %s", filename, line, strings.Join(methods, ", "), fullpath)
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

	// before join the middleware + handlers + done handlers and apply the execution rules.

	mainHandlerName, mainHandlerIndex := context.MainHandlerName(mainHandlers)

	mainHandlerFileName, mainHandlerFileNumber := context.HandlerFileLineRel(mainHandlers[mainHandlerIndex])

	// TODO: think of it.
	if mainHandlerFileName == "<autogenerated>" {
		// At PartyConfigure, 2nd+ level of routes it will get <autogenerated> but in reallity will be the same as the caller.
		mainHandlerFileName = filename
		mainHandlerFileNumber = line
	}

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
		route, err := NewRoute(api, errorCode, m, subdomain, path, routeHandlers, *api.macros)
		if err != nil { // template path parser errors:
			api.logger.Errorf("[%s:%d] %v -> %s:%s:%s", filename, line, err, m, subdomain, path)
			continue
		}

		// The caller tiself, if anonymous, it's the first line of `app.X("/path", here)`
		route.RegisterFileName = mainHandlerFileName     // filename
		route.RegisterLineNumber = mainHandlerFileNumber // line

		route.MainHandlerName = mainHandlerName
		route.MainHandlerIndex = mainHandlerIndex

		// The main handler source, could be the same as the register's if anonymous.
		route.SourceFileName = mainHandlerFileName
		route.SourceLineNumber = mainHandlerFileNumber

		// Add UseGlobal & DoneGlobal Handlers
		// route.Use(api.beginGlobalHandlers...)
		// route.Done(api.doneGlobalHandlers...)

		route.NoLog = api.routesNoLog
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
// A Party groups routes which may have the same prefix or subdomain and share same middlewares.
//
// To create a group of routes for subdomains
// use the `Subdomain` or `WildcardSubdomain` methods
// or pass a "relativePath" of "admin." or "*." respectfully.
func (api *APIBuilder) Party(relativePath string, handlers ...context.Handler) Party {
	if relativePath == "" {
		relativePath = "/"
	}

	// if app.Party("/"), root party or app.Party("/user") == app.Party("/user")
	// then just add the middlewares and return itself.
	// if relativePath == "" || api.relativePath == relativePath {
	// 	api.Use(handlers...)
	// 	return api
	// }
	// ^ No, this is wrong, let the developer do its job, if she/he wants a copy let have it,
	// it's a pure check as well, a path can be the same even if it's the same as its parent, i.e.
	// app.Party("/user").Party("/user") should result in a /user/user, not a /user.

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

	// make a copy of the parent properties.
	properties := make(context.Map, len(api.properties))
	for k, v := range api.properties {
		properties[k] = v
	}

	childAPI := &APIBuilder{
		// global/api builder
		logger:              api.logger,
		macros:              api.macros,
		properties:          properties,
		routes:              api.routes,
		routesNoLog:         api.routesNoLog,
		beginGlobalHandlers: api.beginGlobalHandlers,
		doneGlobalHandlers:  api.doneGlobalHandlers,

		// per-party/children
		parent:                api,
		middleware:            middleware,
		middlewareErrorCode:   context.JoinHandlers(api.middlewareErrorCode, context.Handlers{}),
		doneHandlers:          api.doneHandlers[0:],
		routerFilters:         api.routerFilters,
		routerFilterHandlers:  api.routerFilterHandlers,
		partyMatcher:          api.partyMatcher,
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
//
//	app.PartyFunc("/users", func(u iris.Party){
//		u.Use(authMiddleware, logMiddleware)
//		u.Get("/", getAllUsers)
//		u.Post("/", createOrUpdateUser)
//		u.Delete("/", deleteUser)
//	})
//
// Look `Party` for more.
func (api *APIBuilder) PartyFunc(relativePath string, partyBuilderFunc func(p Party)) Party {
	p := api.Party(relativePath)
	partyBuilderFunc(p)
	return p
}

type (
	// PartyConfigurator is an interface which all child parties that are registered
	// through `PartyConfigure` should implement.
	PartyConfigurator interface {
		Configure(parent Party)
	}

	// StrictlyPartyConfigurator is an optional interface which a `PartyConfigurator`
	// can implement to make sure that all exported fields having a not-nin, non-zero
	// value before server starts.
	// StrictlyPartyConfigurator interface {
	// 	Strict() bool
	// }
	// Good idea but a `mvc or bind:"required"` is a better one I think.
)

// PartyConfigure like `Party` and `PartyFunc` registers a new children Party
// but instead it accepts a struct value which should implement the PartyConfigurator interface.
//
// PartyConfigure accepts the relative path of the child party
// (As an exception, if it's empty then all configurators are applied to the current Party)
// and one or more Party configurators and
// executes the PartyConfigurator's Configure method.
//
// If the end-developer registered one or more dependencies upfront through
// RegisterDependencies or ConfigureContainer.RegisterDependency methods
// and "p" is a pointer to a struct then try to bind the unset/zero exported fields
// to the registered dependencies, just like we do with Controllers.
// Useful when the api's dependencies amount are too much to pass on a function.
//
// Usage:
//
//	app.PartyConfigure("/users", &api.UsersAPI{UserRepository: ..., ...})
//
// Where UsersAPI looks like:
//
//	type UsersAPI struct { [...] }
//	func(api *UsersAPI) Configure(router iris.Party) {
//	 router.Get("/{id:uuid}", api.getUser)
//	 [...]
//	}
//
// Usage with (static) dependencies:
//
//	app.RegisterDependency(userRepo, ...)
//	app.PartyConfigure("/users", new(api.UsersAPI))
func (api *APIBuilder) PartyConfigure(relativePath string, partyReg ...PartyConfigurator) Party {
	var child Party

	if relativePath == "" {
		child = api
	} else {
		child = api.Party(relativePath)
	}

	for _, p := range partyReg {
		if p == nil {
			continue
		}

		if len(api.apiBuilderDI.Container.Dependencies) > 0 {
			if typ := reflect.TypeOf(p); typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct {
				api.apiBuilderDI.Container.Struct(p, -1)
			}
		}

		p.Configure(child)
	}

	return child
}

// Subdomain returns a new party which is responsible to register routes to
// this specific "subdomain".
//
// If called from a child party then the subdomain will be prepended to the path instead of appended.
// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
func (api *APIBuilder) Subdomain(subdomain string, middleware ...context.Handler) Party {
	if api.relativePath == SubdomainWildcardIndicator {
		// cannot concat wildcard subdomain with something else
		api.logger.Errorf("cannot concat parent wildcard subdomain with anything else ->  %s , %s",
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
		api.logger.Errorf("cannot concat static subdomain with a dynamic one. Dynamic subdomains should be at the root level -> %s",
			api.relativePath)
		return api
	}
	return api.Subdomain(SubdomainWildcardIndicator, middleware...)
}

// Macros returns the macro collection that is responsible
// to register custom macros with their own parameter types and their macro functions for all routes.
//
// Learn more at:  https://github.com/kataras/iris/tree/main/_examples/routing/dynamic-path
func (api *APIBuilder) Macros() *macro.Macros {
	return api.macros
}

// Properties returns the original Party's properties map,
// it can be modified before server startup but not afterwards.
func (api *APIBuilder) Properties() context.Map {
	if api.properties == nil {
		api.properties = make(context.Map)
	}

	return api.properties
}

// GetRoutes returns the routes information,
// some of them can be changed at runtime some others not.
//
// Needs refresh of the router to Method or Path or Handlers changes to take place.
func (api *APIBuilder) GetRoutes() []*Route {
	return api.routes.getAll()
}

// CountHandlers returns the total number of all unique
// registered route handlers.
func (api *APIBuilder) CountHandlers() int {
	uniqueNames := make(map[string]struct{})

	for _, r := range api.GetRoutes() {
		for _, h := range r.Handlers {
			handlerName := context.HandlerName(h)
			if _, exists := uniqueNames[handlerName]; !exists {
				uniqueNames[handlerName] = struct{}{}
			}
		}
	}

	return len(uniqueNames)
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

// SetRoutesNoLog disables (true) the verbose logging for the next registered
// routes under this Party and its children.
//
// To disable logging for controllers under MVC Application,
// see `mvc/Application.SetControllersNoLog` instead.
//
// Defaults to false when log level is "debug".
func (api *APIBuilder) SetRoutesNoLog(disable bool) Party {
	api.routesNoLog = disable
	return api
}

type (
	// PartyMatcherFunc used to build a filter which decides
	// if the given Party is responsible to fire its `UseRouter` handlers or not.
	// Can be customized through `SetPartyMatcher` method. See `Match` method too.
	PartyMatcherFunc func(*context.Context, Party) bool
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
		Matcher   PartyMatcher             // it's a Party, for freedom that can be changed through a custom matcher which accepts the same filter.
		Skippers  map[*APIBuilder]struct{} // skip execution on these builders ( see `Reset`)
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
	return api.partyMatcher(ctx, api)
}

func defaultPartyMatcher(ctx *context.Context, p Party) bool {
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
	if len(handlers) == 0 || handlers[0] == nil {
		return
	}

	beginHandlers := context.Handlers(handlers)
	// respect any execution rules (begin).
	api.handlerExecutionRules.Begin.apply(&beginHandlers)
	beginHandlers = context.JoinHandlers(api.routerFilterHandlers, beginHandlers)

	if f := api.routerFilters[api]; f != nil && len(f.Handlers) > 0 { // exists.
		beginHandlers = context.UpsertHandlers(f.Handlers, beginHandlers) // remove dupls.
	}

	// we are not using the parent field here,
	// we need to have control over those values in order to be able to `Reset`.
	api.routerFilterHandlers = beginHandlers

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
// The given "handlers" will be executed only on matched routes.
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
// including all parties, subdomains and errors.
// It doesn't care about call order, it will prepend the handlers to all
// existing routes and the future routes that may being registered.
//
// The given "handlers" will be executed only on matched routes and registered errors.
// See `UseRouter` if you want to register middleware that will always run, even on 404 not founds.
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
// The given "handlers" will be executed only on matched routes.
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
// The given "handlers" will be executed only on matched and registered error routes.
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

// MiddlewareExists reports whether the given handler exists in the middleware chain.
func (api *APIBuilder) MiddlewareExists(handlerNameOrFunc any) bool {
	if handlerNameOrFunc == nil {
		return false
	}

	var handlers context.Handlers

	if filter, ok := api.routerFilters[api]; ok {
		handlers = append(handlers, filter.Handlers...)
	}

	handlers = append(handlers, api.middleware...)
	handlers = append(handlers, api.doneHandlers...)
	handlers = append(handlers, api.beginGlobalHandlers...)
	handlers = append(handlers, api.doneGlobalHandlers...)

	return context.HandlerExists(handlers, handlerNameOrFunc)
}

// RemoveHandler deletes a handler from begin and done handlers
// based on its name or the handler pc function.
// Note that UseGlobal and DoneGlobal handlers cannot be removed
// through this method as they were registered to the routes already.
//
// As an exception, if one of the arguments is a pointer to an int,
// then this is used to set the total amount of removed handlers.
//
// Returns the Party itself for chain calls.
//
// Should be called before children routes regitration.
func (api *APIBuilder) RemoveHandler(namesOrHandlers ...interface{}) Party {
	var counter *int

	for _, nameOrHandler := range namesOrHandlers {
		handlerName := ""
		switch h := nameOrHandler.(type) {
		case string:
			handlerName = h
		case context.Handler: //, func(*context.Context):
			handlerName = context.HandlerName(h)
		case *int:
			counter = h
		default:
			panic(fmt.Sprintf("remove handler: unexpected type of %T", h))
		}

		api.middleware = removeHandler(handlerName, api.middleware, counter)
		api.doneHandlers = removeHandler(handlerName, api.doneHandlers, counter)
	}

	return api
}

// Reset removes all the begin and done handlers that may derived from the parent party via `Use` & `Done`,
// and the execution rules.
// Note that the `Reset` will not reset the handlers that are registered via `UseGlobal` & `DoneGlobal`.
//
// Returns this Party.
func (api *APIBuilder) Reset() Party {
	api.middleware = api.middleware[0:0]
	api.middlewareErrorCode = api.middlewareErrorCode[0:0]
	api.ResetRouterFilters()

	api.doneHandlers = api.doneHandlers[0:0]
	api.handlerExecutionRules = ExecutionRules{}
	api.routeRegisterRule = RouteOverride

	// keep container as it's.
	return api
}

// ResetRouterFilters deactivates any previous registered
// router filters and the parents ones for this Party.
//
// Returns this Party.
func (api *APIBuilder) ResetRouterFilters() Party {
	api.routerFilterHandlers = api.routerFilterHandlers[0:0]
	delete(api.routerFilters, api)

	if api.parent == nil {
		// it's the root, stop, nothing else to do here.
		return api
	}

	// Set a filter with empty handlers, the router will find it, execute nothing
	// and continue with the request handling. This works on Reset() and no UseRouter
	// and with Reset().UseRouter.
	subdomain, path := splitSubdomainAndPath(api.relativePath)
	api.routerFilters[api] = &Filter{
		Matcher:   api,
		Handlers:  nil,
		Subdomain: subdomain,
		Path:      path,
	}

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

type (
	// ServerHandler is the interface which all server handlers should implement.
	// The Iris Application implements it.
	// See `Party.HandleServer` method for more.
	ServerHandler interface {
		ServeHTTPC(*context.Context)
	}

	serverBuilder interface {
		Build() error
	}
)

// HandleServer registers a route for all HTTP methods which forwards the requests to the given server.
//
// Usage:
//
//	app.HandleServer("/api/identity/{first:string}/orgs/{second:string}/{p:path}", otherApp)
//
// OR
//
//	app.HandleServer("/api/identity", otherApp)
func (api *APIBuilder) HandleServer(path string, server ServerHandler) {
	if server == nil {
		return
	}

	if app, ok := server.(serverBuilder); ok {
		// Do an extra check for Build() error at any case
		// the end-developer didn't call Build before.
		if err := app.Build(); err != nil {
			panic(err)
		}
	}

	pathParameterName := ""

	// Check and get the last parameter name if it's a wildcard one by the end-developer.
	parsedPath, err := macro.Parse(path, *api.macros)
	if err != nil {
		panic(err)
	}

	if n := len(parsedPath.Params); n > 0 {
		lastParam := parsedPath.Params[n-1]
		if lastParam.IsMacro(macro.Path) {
			pathParameterName = lastParam.Name
			// path remains as it was defined by the end-developer.
		}
	}
	//

	if pathParameterName == "" {
		pathParameterName = fmt.Sprintf("iris_wildcard_path_parameter%d", len(api.routes.routes))
		path = fmt.Sprintf("%s/{%s:path}", path, pathParameterName)
	}

	handler := makeServerHandler(pathParameterName, server.ServeHTTPC)
	api.Any(path, handler)
}

func makeServerHandler(givenPathParameter string, handler context.Handler) context.Handler {
	return func(ctx *context.Context) {
		pathValue := ""
		if givenPathParameter == "" {
			pathValue = ctx.Params().GetEntryAt(ctx.Params().Len() - 1).ValueRaw.(string)
		} else {
			pathValue = ctx.Params().Get(givenPathParameter)
		}

		apiPath := "/" + pathValue

		r := ctx.Request()
		r.URL.Path = apiPath
		r.URL.RawPath = apiPath
		ctx.Params().Reset()

		handler(ctx)
	}
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
		api.logger.Errorf("favicon: file or directory %s not found: %w", favPath, err)
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
		api.logger.Errorf("favicon: couldn't read the data bytes for %s: %w", favPath, err)
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

// RegisterView registers and loads a view engine middleware for this group of routes.
// It overrides any of the application's root registered view engines.
// To register a view engine per handler chain see the `Context.ViewEngine` instead.
// Read `Configuration.ViewEngineContextKey` documentation for more.
func (api *APIBuilder) RegisterView(viewEngine context.ViewEngine) {
	if err := viewEngine.Load(); err != nil {
		api.logger.Error(err)
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

// FallbackView registers one or more fallback views for a template or a template layout.
// Usage:
//
//	FallbackView(iris.FallbackView("fallback.html"))
//	FallbackView(iris.FallbackViewLayout("layouts/fallback.html"))
//	OR
//	FallbackView(iris.FallbackViewFunc(ctx iris.Context, err iris.ErrViewNotExist) error {
//	  err.Name is the previous template name.
//	  err.IsLayout reports whether the failure came from the layout template.
//	  err.Data is the template data provided to the previous View call.
//	  [...custom logic e.g. ctx.View("fallback", err.Data)]
//	})
func (api *APIBuilder) FallbackView(provider context.FallbackViewProvider) {
	handler := func(ctx *context.Context) {
		ctx.FallbackView(provider)
		ctx.Next()
	}
	api.Use(handler)
	api.UseError(handler)
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
//
//		my.Get("/", func(ctx iris.Context) {
//		if err := ctx.View("page1.html"); err != nil {
//		  ctx.HTML("<h3>%s</h3>", err.Error())
//		  return
//	 }
//		})
//
// Examples: https://github.com/kataras/iris/tree/main/_examples/view
func (api *APIBuilder) Layout(tmplLayoutFile string) Party {
	handler := func(ctx *context.Context) {
		ctx.ViewLayout(tmplLayoutFile)
		ctx.Next()
	}

	api.Use(handler)
	api.UseError(handler)

	return api
}
