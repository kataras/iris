package router

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/dep"
	"github.com/kataras/iris/v14/core/router/macro"
)

// Router is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Router could also be named as 'Join' or 'Node' or 'Group' , Router chosen because it is fun.
//
// Look the `RouterBuilder` structure for its implementation.
type Router interface {
	context.Router

	// Application returns the read-only application.
	// Can be casted as *iris.Application and used to access the application's properties and methods
	// on Router Controllers.
	Application() context.Application
	// IsRoot reports whether this Router is the root Application's one.
	// It will return false on all children Routers, no exception.
	IsRoot() bool

	// GetRoot returns the very first Router (the Application's Router Builder).
	GetRoot() *RouterBuilder

	// Routes returns the route repository.
	// It's been used by the `ServeMux` and `Application` to register the routes
	// on Handle, Get, Post and so on.
	Routes() *RouteRegistry
	// GetRelPath returns the current Router's relative path.
	// i.e:
	// if r := app.Router("/users"), then the `r.GetRelPath()` is the "/users".
	// if r := app.Router("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
	GetRelPath() string
	// Macros returns the macro collection that is responsible
	// to register custom macros with their own parameter types and their macro functions for all routes.
	// It belongs to this application: it starts as a copy of macro.Defaults and
	// what is registered on it does not reach another application.
	//
	// Learn more at:  https://github.com/kataras/iris/tree/main/_examples/02-routing/custom-macro
	Macros() *macro.Macros

	// Properties returns the original Router's properties map,
	// it can be modified before server startup but not afterwards.
	Properties() context.Map

	// SetRoutesNoLog disables (true) the verbose logging for the next registered
	// routes under this Router and its children.
	//
	// It covers the controllers registered through Router too.
	//
	// Defaults to false when log level is "debug".
	SetRoutesNoLog(disable bool)

	// DI returns the Dependency Injection Container of this Router.
	// Useful to configure the dependencies and inject them on the controllers and handlers.
	DI() *dep.Container
	// RegisterDependency registers a single or multiple dependencies to the router's DI Container.
	// It's a shortcut of `router.DI().RegisterDependency`.
	RegisterDependency(deps ...any) error
	// HandlerFunc resolves a function which returns a Handler as a Handler.
	// It accepts a function with arguments that are resolved by the registered dependencies and returns a Handler.
	// It panics if the given function returns nil or a different type than Handler.
	//
	// Example:
	//
	//	app.RegisterDependency(func(otherDependnecies...) *MyService {
	//		return &MyService{...}
	//	})
	//
	//	handler := app.HandlerFunc(func(service *MyService) iris.Handler {
	//		return func(ctx iris.Context) {
	//			ctx.WriteString(service.Get())
	//		}
	//	})
	//
	//	app.Get("/", handler)
	//
	// Or as middleware:
	//
	//	app.Use("/", app.HandlerFunc(func(service *MyService) iris.Handler {
	//		return func(ctx iris.Context) {
	//			// [middleware action...] before the main handler.
	//			ctx.Next()
	//		}
	//	}))
	HandlerFunc(fn any) context.Handler
	// HandlerFuncs resolves a list of functions which return functions with dependencies as Handlers.
	// Read the `HandlerFunc` documentation for more information.
	HandlerFuncs(fns ...any) context.Handlers
	// Router creates and returns a new child Router based on a prefix path.
	// It accepts a relative path and one or more handlers or controllers and returns the child group.
	//
	// Simple group usage:
	// It accepts one or more handlers as middlewares.
	// 	usersAPI := app.Router("/users", authMiddleware, logMiddleware)
	// 	usersAPI.Handle("GET", ...)
	//
	// Router func usage:
	// It accepts a function that receives this created child Router.
	// 	app.Router("/users", func(r iris.Router) {
	//		r.Use(authMiddleware, logMiddleware)
	//		r.Get("/", getAllUsers)
	//		r.Post("/", createOrUpdateUser)
	//		r.Delete("/", deleteUser)
	// 	})
	//
	// Routing interface usage:
	// It accepts a pointer to a struct value which implements the Routing interface: Routing(r Router) method.
	// Useful when the api's dependencies amount are too much to pass on a function.
	//	app.Router("/users", &api.UserController{UserRepository: ..., ...})
	//
	// Where UserController looks like:
	//
	//	type UserController struct { Users UserRepository }
	//	func(c *UserController) Routing(r iris.Router) {
	//	 r.Get("/{id:uuid}", c.getUser)
	//	 [...]
	//	}
	//
	// Usage with (static) dependencies:
	//
	//	app.RegisterDependency(userRepo, ...)
	//	app.Router("/users", new(router.UserController))
	//
	// Controller usage:
	// A controller can be any type of the following:
	//
	//   - A struct which implements the Controller interface.
	//   - A struct which its fields are binded based on the registered dependencies.
	//   - A function which returns a struct which implements the Controller interface.
	//   - A function which accepts a Router and returns nothing.
	//   - A function which accepts a Router and returns a struct which implements the Controller interface.
	//   - A function which returns a struct which implements the Controller interface and accepts a Router.
	//   - A function which returns a struct which implements the Controller interface and accepts a Router and
	//     a pointer to a struct which its fields are binded based on the registered dependencies.
	Router(relativePath string, c ...any) Router
	// Subdomain returns a new Router which is responsible to register routes to this specific "subdomain".
	//
	// If called from a child Router then the subdomain will be prepended to the path instead of appended.
	// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
	Subdomain(subdomain string, middleware ...context.Handler) Router

	// UseRouter appends one or more middlewares to the Application.
	// The given "handlers" will be executed before the routing logic itself.
	// So before any route, no match routes and no method allowed requests.
	// The handlers are registered to the top-level router, even on 404s.
	// Useful when you want to log all incoming requests or change the path before mux or add a global rate limiter.
	UseRouter(handlers ...context.Handler)
	// Use registers middlewares on all routes, including 404 and 405 requests for root application
	// and just matched routes for child routers.
	Use(handlers ...context.Handler)
	// UseRelevant registers middlewares on matched routes only, on the root
	// application as well. Prefer it over Use when the middleware makes no
	// sense for a request that matched nothing, an authentication check for
	// instance: through Use, on the root application, it would answer for
	// every unknown path too.
	UseRelevant(handlers ...context.Handler)
	// UseOnce either inserts a middleware or on the basis of the middleware already existing, replace that existing middleware instead.
	// The given "handlers" will be executed on all routes, including 404 and 405 requests for root application
	// and just matched routes for child routers.
	UseOnce(handlers ...context.Handler)
	// RoutesHandlers returns the list of this Router's registered route handlers information.
	// The returned value is a cloned one, any changes made to the returned value
	// will NOT affect the next registered route handlers.
	RoutesHandlers() *context.HandlerList

	// EnableStatusCodeResponse registers a middleware that will handle the status code responses automatically.
	EnableStatusCodeResponse()
	// AllowMethods will re-register the future routes that will be registered
	// via `Handle`, `Get`, `Post`, ... to the given "methods" on that Router and its children,
	// duplicates are not registered.
	//
	// Call of `AllowMethods` will override any previous allow methods.
	AllowMethods(methods ...string)

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
	//	app.Macros().Get("int").RegisterFunc("min", func(argument int) func(paramValue string) bool {
	//	 [...]
	//	 return true/false -> true means valid.
	//	})
	//
	// at the func(argument ...) you can have any standard type, it will be validated before the server starts
	// so don't care about performance here, the only thing it runs at serve time is the returning func(paramValue string) bool.
	//
	// {param:string equal(iris)} , "iris" will be the argument here:
	//
	//	app.Macros().Get("string").RegisterFunc("equal", func(argument string) func(paramValue string) bool {
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
	Handle(method string, registeredPath string, handlers ...context.Handler) *RouteRegistry

	// None registers an "offline" route: it is added to the registry under the
	// MethodNone method, so the serve mux never matches an incoming request to it.
	//
	// Returns the registered route or nil on failure.
	None(relativePath string, handlers ...context.Handler) *Route

	// Get registers a route for the Get HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Get(relativePath string, handlers ...context.Handler) *Route
	// Post registers a route for the Post HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Post(relativePath string, handlers ...context.Handler) *Route
	// Query registers a route for the QUERY HTTP Method (RFC 10008):
	// a safe, idempotent request that carries a body, e.g. a list filter.
	//
	// Returns the registered route or nil on failure.
	Query(relativePath string, handlers ...context.Handler) *Route
	// Put registers a route for the Put HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Put(relativePath string, handlers ...context.Handler) *Route
	// Delete registers a route for the Delete HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Delete(relativePath string, handlers ...context.Handler) *Route
	// Connect registers a route for the Connect HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Connect(relativePath string, handlers ...context.Handler) *Route
	// Head registers a route for the Head HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Head(relativePath string, handlers ...context.Handler) *Route
	// Options registers a route for the Options HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Options(relativePath string, handlers ...context.Handler) *Route
	// Patch registers a route for the Patch HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Patch(relativePath string, handlers ...context.Handler) *Route
	// Trace registers a route for the Trace HTTP Method.
	//
	// Returns the registered route or nil on failure.
	Trace(relativePath string, handlers ...context.Handler) *Route
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
	Any(registeredPath string, handlers ...context.Handler) *RouteRegistry
	// Resource registers a route for the Get and Head HTTP Methods.
	// It's a shortcut for router.Handle("GET HEAD", relativePath, handlers...).
	Resource(relativePath string, handlers ...context.Handler) *RouteRegistry
	// HandleServer registers a route for all HTTP methods which forwards the requests to the given server.
	//
	// Usage:
	//
	//	app.HandleServer("/api/identity/{first:string}/orgs/{second:string}/{p:path}", otherApp)
	//
	// OR
	//
	//	app.HandleServer("/api/identity", otherApp)
	HandleServer(relativePath string, server ServerHandler)

	// CreateRoutes returns a list of Router-based Routes.
	// It does NOT registers the route. Use `Handle, Get...` methods instead.
	// This method can be used for third-parties Iris helpers packages and tools
	// that want a more detailed view of Router-based Routes before take the decision to register them.
	// CreateRoutes(methods []string, relativePath string, handlers ...context.Handler) *RouteRegistry

	// RemoveRoute deletes a registered route by its name before `Application.Listen`.
	// The default naming for newly created routes is: method + subdomain + path.
	// Reports whether a route with that name was found and removed successfully.
	//
	// After the router is built the removed route keeps answering until
	// `BuildRouter` runs again, which needs RouterOptions.EnableDynamicHandler.
	//
	// Note that this method applies to all children Routers
	// even if each of the Router have access to this method,
	// as the route name is unique per Iris Application.
	RemoveRoute(routeName string) bool

	// StaticContent registers a GET and HEAD method routes to the requestPath
	// that are ready to serve raw static bytes, memory cached.
	//
	// Returns the GET *Route.
	StaticContent(requestPath string, cType string, content []byte) *Route
}
