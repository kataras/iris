package router

import (
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/dep"
	"github.com/kataras/iris/v14/core/pkg/funcer"
	"github.com/kataras/iris/v14/core/router/macro"
)

// RouterBuilder is the API builder for the Router.
// It's responsible to build the API and the router handler.
// It's the only one that should be used to create a new Router.
// It's been used inside the Iris `Application`.
type RouterBuilder struct {
	// app read-only app can be casted to an *iris.Application.
	app context.Application
	// parent is the creator of this Router.
	// It is nil on Root.
	parent *RouterBuilder // currently it's not used anywhere.

	// the per-router RouterBuilder with DI.
	diContainer *dep.Container

	// the api builder global macros registry
	macros *macro.Macros
	// the per-router (and its children) values map
	// that may help on building the API
	// when source code is splitted between projects.
	// Initialized on Properties method.
	properties context.Map
	// These options are only modified by the Router/RouterBuilder. Per router.
	// If end-developer modifies them inside a route itself, these changes will reflect
	// the rest of the routes too, as we don't clone the request options per route.
	contextOptions *context.ContextOptions
	// the api builder global routes repository.
	routes *RouteRegistry
	// disables the debug logging of routes under a per-router and its children.
	routesNoLog bool

	routerHandlers *context.HandlerList // global handlers, executed before the actual routing logic itself.
	handlers       *context.HandlerList // per router, 404s and 405s included.
	// per router, registered only to matched routes, no 404s and 405s.
	// In other words, the `Handlers` are being executed before the `RoutesHandlers`.
	// Router Builder's Handlers + Router's are the handlers are being executed before the main route handlers.
	routesHandlers *context.HandlerList

	noRouteHandlers  *context.HandlerList // global. The root's "handlers" are prepended.
	noMethodHandlers *context.HandlerList // global. The root's "handlers" are prepended.
	//

	// the per-router relative path.
	relativePath string
	// allowMethods are filled with the `AllowMethods` method.
	// They are used to create new routes
	// per any router's (and its children) routes registered
	// if the method "x" wasn't registered already via  the `Handle` (and its extensions like `Get`, `Post`...).
	allowMethods []string

	// Set on `WrapRouter` method calls.
	wrapperFunc WrapperFunc
	// wrappers to be built on BuildRouter state,
	// first is executed first at this case.
	// Case:
	// - SubdomainRedirect on user call, registers a wrapper, on design state
	// - i18n,if loaded and Subdomain or PathRedirect is true, registers a wrapper too, on build state
	// the SubdomainRedirect should be the first(subdomainWrap(i18nWrap)) wrapper
	// to be executed instead of last(i18nWrap(subdomainWrap)).
	wrapperFuncs []WrapperFunc
	// Set on `BuildRouter` method call.
	http.Handler // the main handler.
	ServeMux
	// dynamic reports whether the router was built with RouterOptions.EnableDynamicHandler,
	// which is what lets `BuildRouter` run again. It is read from the root.
	dynamic bool
}

var (
	_ Router       = (*RouterBuilder)(nil)
	_ ServeMux     = (*RouterBuilder)(nil)
	_ http.Handler = (*RouterBuilder)(nil)
)

// New creates & returns a new router builder
// which is responsible to build the API and the router handler.
func New(app context.Application) *RouterBuilder {
	defaultContextOptions := context.DefaultContextOptions()

	return &RouterBuilder{
		app:            app,
		parent:         nil,
		macros:         macro.Defaults,
		relativePath:   "/",
		routerHandlers: context.NewHandlerList(),
		handlers:       context.NewHandlerList(),
		routesHandlers: context.NewHandlerList(),
		// http errors.
		noRouteHandlers:  context.NewHandlerList(),
		noMethodHandlers: context.NewHandlerList(),
		//
		contextOptions: &defaultContextOptions,
		routes:         new(RouteRegistry),
		diContainer:    dep.NewContainer(),
	}
}

func (router *RouterBuilder) clone(fullpath string, middleware *context.HandlerList) *RouterBuilder {
	return &RouterBuilder{
		app: router.app,
		// global/router builder
		macros: router.macros,
		// make a copy of the parent properties.
		properties: maps.Clone(router.properties),
		// make a copy of the parent context options.
		contextOptions: router.contextOptions.Clone(),
		routes:         router.routes,
		routesNoLog:    router.routesNoLog,
		routerHandlers: router.routerHandlers, //.Clone(),
		handlers:       router.handlers,       // globally.
		// make a copy of the parent handlers and append any middleware.
		routesHandlers:   context.ConcatHandlerLists(router.routesHandlers, middleware),
		noRouteHandlers:  router.noRouteHandlers,
		noMethodHandlers: router.noMethodHandlers,
		Handler:          router.Handler,
		ServeMux:         router.ServeMux,
		// handlerTimeout:    router.handlerTimeout,
		// handlerTimeoutMsg: router.handlerTimeoutMsg,
		wrapperFunc:  router.wrapperFunc,
		wrapperFuncs: router.wrapperFuncs,
		// per-router/children
		parent:       router,
		relativePath: fullpath,
		// the allow methods per router and its children.
		allowMethods: slices.Clone(router.allowMethods),
		// attach a new Container with correct dynamic path parameter start index for input arguments
		// based on the fullpath.
		diContainer: router.diContainer.Clone(),
	}
}

// IsRoot reports whether this Router is the root Application's one.
// It will return false on all children Parties, no exception.
func (router *RouterBuilder) IsRoot() bool {
	return router.parent == nil
}

// GetRoot returns the very first Router (the Application's Router Builder).
// It walks the parent chain to the root RouterBuilder. Safe to call on the
// root itself: in that case the receiver is returned. Not safe to call
// concurrently with router cloning.
func (router *RouterBuilder) GetRoot() *RouterBuilder {
	returnRouter := router
	for returnRouter.parent != nil {
		returnRouter = returnRouter.parent
	}
	return returnRouter
}

// Application returns the read-only application.
func (router *RouterBuilder) Application() context.Application {
	return router.app
}

// ContextOptions returns the context options.
func (router *RouterBuilder) ContextOptions() *context.ContextOptions {
	return router.contextOptions
}

// Routes returns the route repository.
// It's been used by the `ServeMux` and `Application` to register the routes
// on Handle, Get, Post and so on.
func (router *RouterBuilder) Routes() *RouteRegistry {
	return router.routes
}

// ListRoutesReadOnly returns a read-only version of the registered routes.
func (router *RouterBuilder) ListRoutesReadOnly() []context.RouteReadOnly {
	routes := router.routes.List()
	routesReadOnly := make([]context.RouteReadOnly, len(routes))
	for i, r := range routes {
		routesReadOnly[i] = r
	}
	return routesReadOnly
}

// GetRelPath returns the current Router's relative path.
// i.e:
// if r := app.Router("/users"), then the `r.GetRelPath()` is the "/users".
// if r := app.Router("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
func (router *RouterBuilder) GetRelPath() string {
	return router.relativePath
}

// AllowMethods will re-register the future routes that will be registered
// via `Handle`, `Get`, `Post`, ... to the given "methods" on that Router and its children "Routers",
// duplicates are not registered.
//
// Call of `AllowMethods` will override any previous allow methods.
func (router *RouterBuilder) AllowMethods(methods ...string) {
	router.allowMethods = methods
}

// UseRouter appends one or more middlewares to the Application.
// The given "handlers" will be executed before the routing logic itself.
// So before any route, no match routes and no method allowed requests.
// The handlers are registered to the top-level router, even on 404s.
// Useful when you want to log all incoming requests or change the path before mux or add a global rate limiter.
func (router *RouterBuilder) UseRouter(handlers ...context.Handler) {
	for _, h := range handlers {
		if h == nil {
			continue
		}

		if router.relativePath == "" || router.relativePath == "/" {
			router.routerHandlers.Append(h)
			// if root. prepend to all routes.
		} else {
			subdomain, path := SplitSubdomainAndPath(router.relativePath)
			path = staticPath(path)

			var routerHandler context.Handler
			if subdomain != "" {
				subdomain = strings.TrimSuffix(subdomain, ".") // remove the dot.
				routerHandler = func(ctx *context.Context) {
					if requestSubdomain := ctx.Subdomain(); requestSubdomain != "" {
						if subdomain != "*" {
							// Match if the request subdomain equals the registered one,
							// or is a nested child (e.g., "staff.admin" matches "admin").
							// HasSuffix alone would let "badmin" match "admin", which
							// is wrong — enforce a subdomain boundary with a leading dot.
							if requestSubdomain != subdomain &&
								!strings.HasSuffix(requestSubdomain, "."+subdomain) {
								ctx.Next() // subdomain didn't match. Continue to the mux.
								return
							}
						}

						if strings.HasPrefix(ctx.Path(), path) {
							// admin. UseRouter => is executed for admin.control. requests too, as they are child requests.
							h(ctx)
							return
						}
					}

					ctx.Next() // no subdomain requested or path prefix didn't match. Continue to the mux.
				}
			} else {
				routerHandler = func(ctx *context.Context) {
					if strings.HasPrefix(ctx.Path(), path) {
						h(ctx)
						return
					}

					ctx.Next() // path prefix didn't match. Continue to the mux.
				}
			}

			router.routerHandlers.Append(routerHandler, context.HandlerInfo(h).CopyTo)
		}
	}
}

// Use registers middlewares on all routes, including 404 and 405 requests for the root application
// and just matched routes for child routers (subdomains and group routers).
func (router *RouterBuilder) Use(handlers ...context.Handler) { // for children routers, the handlers will be registered to the current router and its children's registered routes, not globally.
	// if router.IsRoot() {
	// 	useAny, useMatched := context.SplitHandlers(handlers)
	// 	router.handlers.AppendSlice(useAny)
	// 	router.routesHandlers.AppendSlice(useMatched)
	// } else { // if not root, then all handlers will be registered to the current router and its children, not globally.
	// 	router.routesHandlers.AppendSlice(context.ClearHandlers(handlers))
	// }
	// Good idea to it as a separate method, it's more clear. Keep this for global,
	// and make a new one, this is the names I have in mind:
	// UseMatched(...)
	// UseRelated(...)
	// UseRelevant(...)
	// UseApplicable(...)
	//
	// BUT, I think it's better to keep it as it is, as end-developers have an experience to use just `Use` method
	// for all routes, including 404s and 405s on root apps and just for matched routes on children (subdomains and group routers).
	// However we also keep the UseRelevant and UseRelevantOnce for those who want to be more specific on the root application level.
	if router.IsRoot() {
		router.handlers.AppendSlice(handlers)
	} else {
		router.UseRelevant(handlers...)
	}
}

// UseRelevant registers middlewares on matched routes only.
func (router *RouterBuilder) UseRelevant(handlers ...context.Handler) {
	router.routesHandlers.AppendSlice(handlers)
}

// UseOnce either inserts a middleware or on the basis of the middleware already existing, replace that existing middleware instead.
// The given "handlers" will be executed on all routes, including 404 and 405 requests.
func (router *RouterBuilder) UseOnce(handlers ...context.Handler) { // for children routers, the handlers will be registered to the current router and its children's registered routes, not globally.
	if router.IsRoot() {
		router.handlers.UpsertSlice(handlers)
	} else {
		router.UseRelevantOnce(handlers...)
	}
}

// UseRelevantOnce either inserts a middleware or on the basis of the middleware already existing, replace that existing middleware instead.
// The given "handlers" will be executed on matched routes only.
func (router *RouterBuilder) UseRelevantOnce(handlers ...context.Handler) {
	router.routesHandlers.UpsertSlice(handlers)
}

// HandleNoRoute registers one or more handlers for the case when no route is found.
// The 404 status code is set by the serve mux before these handlers run; the default handler
// writes the stored public error, if any, otherwise the status text ("Not Found").
func (router *RouterBuilder) HandleNoRoute(handlers ...context.Handler) {
	router.noRouteHandlers.Set(handlers...).PrependList(router.handlers)
}

// HandleNoMethod registers one or more handlers for the case when the request's method is not allowed.
// The 405 status code and the Allow header are set by the serve mux before these handlers run; the
// default handler writes the stored public error, if any, otherwise the status text ("Method Not Allowed").
func (router *RouterBuilder) HandleNoMethod(handlers ...context.Handler) {
	router.noMethodHandlers.Set(handlers...).PrependList(router.handlers)
}

// EnableStatusCodeResponse registers a middleware that will handle the status code responses automatically.
func (router *RouterBuilder) EnableStatusCodeResponse() {
	// The explicit name makes repeated calls idempotent (Upsert replaces by
	// trusted name); the closure's compiler-generated symbol cannot serve as
	// its identity.
	router.routesHandlers.Upsert(func(ctx *context.Context) {
		ctx.Next()

		// TODO: fix when recover (plain 500 or we can 503) this won't execute, even with "defer",
		// shall we move this functionality as a property of the response writer itself?
		defaultHTTPErrorHandler(ctx)
	}, funcer.WithFuncName[context.Handler]("EnableStatusCodeResponse"))
}

// Handle registers a route to this specific group of routes (Router).
// If "method" is empty then the route is registered to all available HTTP Methods.
// The "method" parameter can be any valid HTTP method, including the HTTP verb itself,
// i.e. "GET", "POST", "PUT", "DELETE", "OPTIONS", "CONNECT", "HEAD", "PATCH", "TRACE".
// The method supports multiple HTTP verbs as well, separated by a single space or single comma,
// i.e.: "GET POST", "GET,POST" and so on.
//
// It returns the registered routes.
//
// Patterns can match the host and path of a request.
//   - "/index.html" matches the path "/index.html" for any host and registered method(s)..
//   - "www." matches the root path for the "www" subdomain to the current host (let's call it {app_domain}).
//   - "site.example./index" matches the path "/index" for the host "site.example.{app_domain}".
//   - "*./index" matches the path "/index.html" to any subdomain (* is the wildcard character).
//   - "/lists/{listname}/objects/{objectname:path}" matches paths whose first segment is "lists"
//     and whose third segment is "objects". The name "listname" denotes the second
//     segment and "objectname" denotes the remainder of the (trailing) path.
//
// For more information about the available path parameters and their types please
// read below.
// Standard macro types for parameters:
// +------------------------+
// | {param:string}         |
// +------------------------+
// string type
// anything (single path segmnent)
//
// +-------------------------------+
// | {param:int}                   |
// +-------------------------------+
// int type
// -9223372036854775808 to 9223372036854775807 (x64) or -2147483648 to 2147483647 (x32), depends on the host arch
//
// +------------------------+
// | {param:int8}           |
// +------------------------+
// int8 type
// -128 to 127
//
// +------------------------+
// | {param:int16}          |
// +------------------------+
// int16 type
// -32768 to 32767
//
// +------------------------+
// | {param:int32}          |
// +------------------------+
// int32 type
// -2147483648 to 2147483647
//
// +------------------------+
// | {param:int64}          |
// +------------------------+
// int64 type
// -9223372036854775808 to 9223372036854775807
//
// +------------------------+
// | {param:uint}           |
// +------------------------+
// uint type
// 0 to 18446744073709551615 (x64) or 0 to 4294967295 (x32)
//
// +------------------------+
// | {param:uint8}          |
// +------------------------+
// uint8 type
// 0 to 255
//
// +------------------------+
// | {param:uint16}         |
// +------------------------+
// uint16 type
// 0 to 65535
//
// +------------------------+
// | {param:uint32}          |
// +------------------------+
// uint32 type
// 0 to 4294967295
//
// +------------------------+
// | {param:uint64}         |
// +------------------------+
// uint64 type
// 0 to 18446744073709551615
//
// +---------------------------------+
// | {param:bool} or {param:boolean} |
// +---------------------------------+
// bool type
// only "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False"
//
// +------------------------+
// | {param:alphabetical}   |
// +------------------------+
// alphabetical/letter type
// letters only (upper or lowercase)
//
// +------------------------+
// | {param:file}           |
// +------------------------+
// file type
// letters (upper or lowercase)
// numbers (0-9)
// underscore (_)
// dash (-)
// point (.)
// no spaces ! or other character
//
// +------------------------+
// | {param:path}           |
// +------------------------+
// path type
// anything, should be the last part, can be requested for more than one path segment,
// i.e: "/test/{param:path}" and request: "/test/path1/path2/path3" , ctx.Params().GetString("param") == "path1/path2/path3"
//
// +------------------------+
// | {param:uuid}           |
// +------------------------+
// UUIDv4 (and v1) path parameter validation.
//
// +------------------------+
// | {param:mail}           |
// +------------------------+
// Email without domain validation.
//
// +------------------------+
// | {param:email}           |
// +------------------------+
// Email with domain validation.
//
// +------------------------+
// | {param:date}           |
// +------------------------+
// yyyy/mm/dd format e.g. /blog/{param:date} matches /blog/2022/04/21.
//
// +------------------------+
// | {param:weekday}        |
// +------------------------+
// positive integer 0 to 6 or
// string of time.Weekday longname format ("sunday" to "monday" or "Sunday" to "Monday")
// format e.g. /schedule/{param:weekday} matches /schedule/monday.
//
// If type is missing then parameter's type is defaulted to string, so
// {param} is identical to {param:string}.
//
// If a function not found on that type then the `string` macro type's functions are being used.
//
// Besides the fact that iris provides the basic types and some default "macro funcs"
// you are able to register your own too.
//
// Register a named path parameter function:
//
//	app.Macros().Number.RegisterFunc("min", func(argument int) func(paramValue string) bool {
//	 [...]
//	 return true/false -> true means valid.
//	})
//
// at the func(argument ...) you can have any standard type, it will be validated before the server starts
// so don't care about performance here, the only thing it runs at serve time is the returning func(paramValue string) bool.
//
// {param:string equal(iris)} , "iris" will be the argument here:
//
//	app.Macros().String.RegisterFunc("equal", func(argument string) func(paramValue string) bool {
//		return func(paramValue string) bool { return argument == paramValue }
//	})
//
// Some examples:
//
//	app.Handle("GET", "/user/{id:uint64}", userByIDHandler)
//	app.Handle("GET", "/user/{name}", userNameHandler)
//	app.Handle("GET", "/user/{name:string min(1) else 504}", userNameHandler)
//	app.Handle("GET", "/user/{name:string min(1) else 504}/profile", userProfileHandler)
//	app.Handle("GET", "/user/{id:uuid}/{name}/profile", userProfileHandler)
//	app.Handle("GET", "/api/{version}/users", apiUsersHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}", apiUserByIDHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}/friends", apiUserFriendsHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}", apiUserFriendByIDHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages", apiUserFriendMessagesHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}", apiUserFriendMessageByIDHandler)
//	app.Handle("POST", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/reply", apiUserFriendMessageReplyHandler)
//	app.Handle("POST", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/forward", apiUserFriendMessageForwardHandler)
//	app.Handle("DELETE", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/delete", apiUserFriendMessageDeleteHandler)
//	app.Handle("PUT", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/edit", apiUserFriendMessageEditHandler)
//	app.Handle("POST", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/report", apiUserFriendMessageReportHandler)
//	app.Handle("GET", "/api/{version}/users/{id:uint64}/friends/{friendID:uint64}/messages/{messageID:uint64}/report/{reportID:uint64}", apiUserFriendMessageReportByIDHandler)
//
// Please refer to the _examples directory for even more productive code samples
// for almost every use case you can imagine. The _examples/02-routing/path-parameters/main.go is the main
// example file you could navigate to.
func (router *RouterBuilder) Handle(method string, pattern string, handlers ...context.Handler) *RouteRegistry {
	// at least slash
	// a space
	// at least one other slash for the next path
	patterns := SplitPath(pattern)
	methods := SplitMethod(method)

	registeredRoutes := router.routes.New(len(patterns) * len(methods))
	for _, p := range patterns {
		if p == "" {
			continue
		}

		newRoutes := router.handle(methods, p, handlers...)
		registeredRoutes.CopyRoutes(newRoutes)
	}

	return registeredRoutes
}

func (router *RouterBuilder) handle(methods []string, pattern string, handlers ...context.Handler) *RouteRegistry {
	if pattern == "" {
		pattern = "/"
	}

	routes := router.createRoutes(methods, pattern, handlers...)  // returns a single route if just one method.
	registeredRoutes, err := router.routes.registerRoutes(routes) // register to the global routes.
	if err != nil {
		panic(err)
	}

	router.warnRegisteredTooLate(registeredRoutes)

	return registeredRoutes
}

// warnRegisteredTooLate reports the routes that were registered after the router
// was built without RouterOptions.EnableDynamicHandler: they will never be served.
//
// That router is compiled once, from a snapshot of the registry: Build, Listen
// and Run return early afterwards and a second BuildRouter returns an error, so
// a route added later stays in the registry and is never matched.
//
// With the option on nothing is reported. There a route is meant to be
// registered while the server runs, and the next BuildRouter serves it.
func (router *RouterBuilder) warnRegisteredTooLate(routes *RouteRegistry) {
	root := router.GetRoot()
	if router.app == nil || root.ServeMux == nil || root.dynamic {
		return
	}

	logger := router.app.Logger()
	for _, r := range routes.List() {
		logger.Warnf("route %s %s was registered after the router was built [%s:%d]: it will not be served (see Options.Router.EnableDynamicHandler)",
			r.Method, r.Path, r.RegisterFileName, r.RegisterLineNumber)
	}
}

// CreateRoutes returns a list of Router-based Routes.
// It does NOT registers the route. Use `Handle, Get...` methods instead.
// This method can be used for third-parties Iris helpers packages and tools
// that want a more detailed view of Router-based Routes before take the decision to register them.
func (router *RouterBuilder) CreateRoutes(methods []string, relativePath string, handlers ...context.Handler) *RouteRegistry {
	return router.createRoutes(methods, relativePath, handlers...)
}

// RemoveRoute deletes a registered route by its name before `Application.Listen`.
// The default naming for newly created routes is: method + subdomain + path.
// Reports whether a route with that name was found and removed successfully.
//
// After the router is built the removed route keeps answering until
// `BuildRouter` runs again, which needs RouterOptions.EnableDynamicHandler.
//
// Note that this method applies to all Parties (sub routers)
// even if each of the Parties have access to this method,
// as the route name is unique per Iris Application.
func (router *RouterBuilder) RemoveRoute(routeName string) bool {
	return router.routes.RemoveByName(routeName)
}

func (router *RouterBuilder) createRoutes(methods []string, pattern string, handlers ...context.Handler) *RouteRegistry {
	if pattern == "" {
		pattern = "/"
	}
	// no clean path yet because of subdomain indicator/separator which contains a dot.
	// but remove the first slash if the relative has already ending with a slash
	// it's not needed because later on we do normalize/clean the path, but better do it here too
	// for any future updates.
	if router.relativePath[len(router.relativePath)-1] == '/' {
		// parent path ends with "/". Remove the first slash from pattern in order to solve any "//path".
		pattern = strings.TrimPrefix(pattern, "/")
	}
	fullpath := router.relativePath + pattern // for now, keep the last "/" if any,  "/xyz/"

	callerFileName, callerFileLine := getCaller()

	if len(handlers) == 0 {
		panic(fmt.Errorf("missing main handlers for route[%s:%d] %s: %s", callerFileName, callerFileLine, strings.Join(methods, ", "), fullpath))
		// return nil
	}

	if len(methods) == 0 || methods[0] == "ALL" || methods[0] == "ANY" { // then use like it was .Any
		return router.Any(pattern, handlers...)
	}

	// before join the middleware + handlers + done handlers and apply the execution rules.

	callerMethods := methods
	methods = RemoveDuplicates(append(router.allowMethods, methods...))

	routes := router.routes.New(len(methods)) // a new clean route repository.
	for _, m := range methods {               // single, empty method for error handlers.
		routeMiddlewares := context.ConcatHandlerLists(router.handlers, router.routesHandlers)
		route, err := NewRoute(router, m, router.GetRelPath(), pattern, routeMiddlewares, context.NewHandlerList(handlers...))
		if err != nil { // template path parser errors:
			// here we separate the subdomain and relative path.
			subdomain, path := SplitSubdomainAndPath(fullpath)
			panic(fmt.Errorf("[%s:%d] %v -> %s:%s:%s", callerFileName, callerFileLine, err, m, subdomain, path))
			// continue
		}
		// A route made for AllowMethods fills a missing method only:
		// it never replaces a route that the caller registered.
		route.allowedOnly = !slices.Contains(callerMethods, m)
		if route.allowedOnly && slices.ContainsFunc(router.routes.List(), func(r *Route) bool {
			return !r.allowedOnly && r.DeepEqual(route)
		}) {
			continue
		}
		route.SetCaller(callerFileName, callerFileLine)
		route.NoLog = router.routesNoLog

		routes.appendRoutes(route) // don't use AddRoute as it will set the topLink twice on `handle`.
	}

	return routes
}

// https://golang.org/doc/go1.9#callersframes
func getCaller() (string, int) {
	var pcs [32]uintptr
	n := runtime.Callers(4, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		file := frame.File

		if strings.Contains(file, "go/src/runtime/") {
			continue
		}

		// funcName is something like "github.com/kataras/iris.SomeFunc"
		funcName := frame.Function
		if !strings.HasPrefix(funcName, "github.com/kataras/iris/v14") {
			return file, frame.Line
		}

		if !more {
			break
		}
	}

	return "???", 0
}

// child returns a new child Router based on a relative path which inherits its parent's options and middlewares.
// It accepts a relative path and one or more handlers and returns the child Router.
// If the relative path is empty, the new Router will be a copy of the parent.
// It backs `Router` and `Subdomain`; the public API is `Router(path, handlers...)`.
func (router *RouterBuilder) child(relativePath string, handlers ...context.Handler) Router {
	if relativePath == "" {
		relativePath = "/"
	}

	// if app.child("/"), root Router or app.child("/user") == app.child("/user")
	// then just add the middlewares and return itself.
	// if relativePath == "" || router.relativePath == relativePath {
	// 	router.Use(handlers...)
	// 	return api
	// }
	// ^ No, this is wrong, let the developer do its job, if she/he wants a copy let have it,
	// it's a pure check as well, a path can be the same even if it's the same as its parent, i.e.
	// app.child("/user").child("/user") should result in a /user/user, not a /user.

	parentPath := router.relativePath
	dot := string(SubdomainPrefix[0])
	if len(parentPath) > 0 && parentPath[0] == '/' && strings.HasSuffix(relativePath, dot) {
		// if ends with . , i.e admin., it's subdomain->
		parentPath = parentPath[1:] // remove first slash
	}

	// this is checked later on but for easier debug is better to do it here:
	if router.relativePath[len(router.relativePath)-1] == '/' && relativePath[0] == '/' {
		relativePath = relativePath[1:] // remove first slash if parent ended with / and new one started with /.
	}

	// if it's subdomain then it has priority, i.e:
	// router.relativePath == "admin."
	// relativePath == "panel."
	// then it should be panel.admin.
	// instead of admin.panel.
	if hasSubdomain(parentPath) && hasSubdomain(relativePath) {
		relativePath = relativePath + parentPath
		parentPath = ""
	}

	fullpath := parentPath + relativePath

	// append the parent's + child's handlers
	childHandlers := context.NewHandlerList(handlers...)
	return router.clone(fullpath, childHandlers)
}

// Subdomain returns a new Router which is responsible to register routes to
// this specific "subdomain".
//
// If called from a child Router then the subdomain will be prepended to the path instead of appended.
// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
func (router *RouterBuilder) Subdomain(subdomain string, middleware ...context.Handler) Router {
	if router.relativePath == SubdomainWildcardIndicator {
		// cannot concat wildcard subdomain with something else
		panic(fmt.Errorf("cannot concat parent wildcard subdomain with anything else ->  %s , %s",
			router.relativePath, subdomain))
		// return router
	}
	if l := len(subdomain); l < 1 {
		return router
	} else if subdomain[l-1] != '.' {
		subdomain += "."
	}

	return router.child(subdomain, middleware...)
}

// WildcardSubdomain returns a new Router which is responsible to register routes to
// a dynamic, wildcard(ed) subdomain. A dynamic subdomain is a subdomain which
// can reply to any subdomain requests. Server will accept any subdomain
// (if not static subdomain found) and it will search and execute the handlers of this Router.
func (router *RouterBuilder) WildcardSubdomain(middleware ...context.Handler) Router {
	if hasSubdomain(router.relativePath) {
		// cannot concat static subdomain with a dynamic one, wildcard should be at the root level
		panic(fmt.Errorf("cannot concat static subdomain with a dynamic one. Dynamic subdomains should be at the root level -> %s",
			router.relativePath))
	}
	return router.Subdomain(SubdomainWildcardIndicator, middleware...)
}

// Macros returns the macro collection that is responsible
// to register custom macros with their own parameter types and their macro functions for all routes.
//
// Learn more at:  https://github.com/kataras/iris/tree/main/_examples/02-routing/custom-macro
func (router *RouterBuilder) Macros() *macro.Macros {
	return router.macros
}

// Properties returns the original Router's properties map,
// it can be modified before server startup but not afterwards.
func (router *RouterBuilder) Properties() context.Map {
	if router.properties == nil {
		router.properties = make(context.Map)
	}

	return router.properties
}

// SetRoutesNoLog disables (true) the verbose logging for the next registered
// routes under this Router and its children.
//
// It covers the controllers registered through Router too.
//
// Defaults to false when log level is "debug".
func (router *RouterBuilder) SetRoutesNoLog(disable bool) {
	router.routesNoLog = disable
}

// RouterHandlers returns the list of all registered router middleware handlers information.
// The returned value is the original one, any changes made to the returned value
// will affect the Router's handlers.
func (router *RouterBuilder) RouterHandlers() *context.HandlerList {
	return router.routerHandlers
}

// Handlers returns the list of this Router's registered middleware handlers information.
// The returned value is the original one, any changes made to the returned value
// will affect the next registered route handlers.
func (router *RouterBuilder) Handlers() *context.HandlerList {
	return router.handlers
}

// NoRouteHandlers returns the list the registered no route handlers information.
// The returned value is the original one, any changes made to the returned value
// will affect all the registered no route handlers.
func (router *RouterBuilder) NoRouteHandlers() *context.HandlerList {
	return router.noRouteHandlers
}

// NoMethodHandlers returns the list the registered no method handlers information.
// The returned value is the original one, any changes made to the returned value
// will affect all the registered no method handlers.
func (router *RouterBuilder) NoMethodHandlers() *context.HandlerList {
	return router.noMethodHandlers
}

// RoutesHandlers returns the list of this Router's registered route handlers information.
// The returned value is a cloned one, any changes made to the returned value
// will NOT affect the next registered route handlers.
func (router *RouterBuilder) RoutesHandlers() *context.HandlerList {
	return context.ConcatHandlerLists(router.handlers, router.routesHandlers)
}

// Reset removes all the middleware and error code handlers.
func (router *RouterBuilder) Reset() {
	if router.IsRoot() {
		router.routerHandlers.Reset()
		router.handlers.Reset()
		router.noRouteHandlers.Reset()
		router.noMethodHandlers.Reset()
		router.routes.SetRegisterRule(RouteOverride)
	}

	router.routesHandlers.Reset()
}

// None registers an "offline" route, under the virtual `MethodNone` ("NONE") method.
// The route is stored in the registry and is listed like any other one, but the serve mux
// never matches an incoming request against it, as no request carries a "NONE" method.
// Its handlers only run when something executes them explicitly.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) None(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodNone, relativePath, handlers...).First()
}

// Get registers a route for the Get HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Get(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodGet, relativePath, handlers...).First()
}

// Post registers a route for the Post HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Post(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodPost, relativePath, handlers...).First()
}

// Query registers a route for the QUERY HTTP Method (RFC 10008):
// a safe, idempotent request that carries a body, e.g. a list filter.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Query(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodQuery, relativePath, handlers...).First()
}

// Put registers a route for the Put HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Put(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodPut, relativePath, handlers...).First()
}

// Delete registers a route for the Delete HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Delete(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodDelete, relativePath, handlers...).First()
}

// Connect registers a route for the Connect HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Connect(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodConnect, relativePath, handlers...).First()
}

// Head registers a route for the Head HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Head(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodHead, relativePath, handlers...).First()
}

// Options registers a route for the Options HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Options(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodOptions, relativePath, handlers...).First()
}

// Patch registers a route for the Patch HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Patch(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodPatch, relativePath, handlers...).First()
}

// Trace registers a route for the Trace HTTP Method.
//
// Returns the registered route or nil on failure.
func (router *RouterBuilder) Trace(relativePath string, handlers ...context.Handler) *Route {
	return router.Handle(MethodTrace, relativePath, handlers...).First()
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
// Query
func (router *RouterBuilder) Any(relativePath string, handlers ...context.Handler) *RouteRegistry {
	return router.handle(AllMethods, relativePath, handlers...)
}

// Resource registers a route for the Get and Head HTTP Methods.
// It's a shortcut for router.Handle("GET HEAD", relativePath, handlers...).
func (router *RouterBuilder) Resource(relativePath string, handlers ...context.Handler) *RouteRegistry {
	return router.Handle(JoinMethods(MethodGet, MethodHead), relativePath, handlers...)
}

type (
	// ServerHandler is the interface which all server handlers should implement.
	// The Iris Application implements it.
	// See `Router.HandleServer` method for more.
	ServerHandler interface {
		HandleRequest(*context.Context)
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
func (router *RouterBuilder) HandleServer(relativePath string, server ServerHandler) {
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
	parsedPath, err := macro.Parse(relativePath, *router.macros)
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
		pathParameterName = fmt.Sprintf("iris_wildcard_path_parameter%d", router.routes.Len())
		relativePath = fmt.Sprintf("%s/{%s:path}", relativePath, pathParameterName)
	}

	handler := makeServerHandler(pathParameterName, server.HandleRequest)
	router.Any(relativePath, handler)
}

func makeServerHandler(givenPathParameter string, handler context.Handler) context.Handler {
	return func(ctx *context.Context) {
		pathValue := ""
		if givenPathParameter == "" {
			if param, ok := ctx.Params().GetLastEntry(); ok {
				pathValue = param.String()
			}
		} else {
			if param, ok := ctx.Params().GetEntry(givenPathParameter); ok {
				pathValue = param.String()
			}
		}

		apiPath := "/" + pathValue

		r := ctx.Request()
		r.URL.Path = apiPath
		r.URL.RawPath = apiPath
		ctx.Params().Reset()

		handler(ctx)
	}
}

// StaticContent registers a GET and HEAD method routes to the requestPath
// that are ready to serve raw static bytes, memory cached.
//
// Returns the GET *Route.
func (router *RouterBuilder) StaticContent(reqPath string, cType string, content []byte) *Route {
	modtime := time.Now()
	h := func(ctx *context.Context) {
		ctx.SetContentType(cType)
		if _, err := ctx.WriteWithExpiration(content, modtime); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			// ctx.Application().Logger().Infof("error while serving []byte via StaticContent: %s", err.Error())
		}
	}

	return router.Resource(reqPath, h).Get()
}
