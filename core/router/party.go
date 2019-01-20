package router

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/macro"
)

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
//
// Look the "APIBuilder" for its implementation.
type Party interface {
	// GetRelPath returns the current party's relative path.
	// i.e:
	// if r := app.Party("/users"), then the `r.GetRelPath()` is the "/users".
	// if r := app.Party("www.") or app.Subdomain("www") then the `r.GetRelPath()` is the "www.".
	GetRelPath() string
	// GetReporter returns the reporter for adding errors
	GetReporter() *errors.Reporter
	// Macros returns the macro collection that is responsible
	// to register custom macros with their own parameter types and their macro functions for all routes.
	//
	// Learn more at:  https://github.com/kataras/iris/tree/master/_examples/routing/dynamic-path
	Macros() *macro.Macros

	// Party groups routes which may have the same prefix and share same handlers,
	// returns that new rich subrouter.
	//
	// You can even declare a subdomain with relativePath as "mysub." or see `Subdomain`.
	Party(relativePath string, middleware ...context.Handler) Party
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
	PartyFunc(relativePath string, partyBuilderFunc func(p Party)) Party
	// Subdomain returns a new party which is responsible to register routes to
	// this specific "subdomain".
	//
	// If called from a child party then the subdomain will be prepended to the path instead of appended.
	// So if app.Subdomain("admin").Subdomain("panel") then the result is: "panel.admin.".
	Subdomain(subdomain string, middleware ...context.Handler) Party

	// Use appends Handler(s) to the current Party's routes and child routes.
	// If the current Party is the root, then it registers the middleware to all child Parties' routes too.
	Use(middleware ...context.Handler)

	// Done appends to the very end, Handler(s) to the current Party's routes and child routes.
	// The difference from .Use is that this/or these Handler(s) are being always running last.
	Done(handlers ...context.Handler)
	// Reset removes all the begin and done handlers that may derived from the parent party via `Use` & `Done`,
	// and the execution rules.
	// Note that the `Reset` will not reset the handlers that are registered via `UseGlobal` & `DoneGlobal`.
	//
	// Returns this Party.
	Reset() Party

	// AllowMethods will re-register the future routes that will be registered
	// via `Handle`, `Get`, `Post`, ... to the given "methods" on that Party and its children "Parties",
	// duplicates are not registered.
	//
	// Call of `AllowMethod` will override any previous allow methods.
	AllowMethods(methods ...string) Party

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
	SetExecutionRules(executionRules ExecutionRules) Party
	// Handle registers a route to the server's router.
	// if empty method is passed then handler(s) are being registered to all methods, same as .Any.
	//
	// Returns the read-only route information.
	Handle(method string, registeredPath string, handlers ...context.Handler) *Route
	// HandleMany works like `Handle` but can receive more than one
	// paths separated by spaces and returns always a slice of *Route instead of a single instance of Route.
	//
	// It's useful only if the same handler can handle more than one request paths,
	// otherwise use `Party` which can handle many paths with different handlers and middlewares.
	//
	// Usage:
	// 	app.HandleMany(iris.MethodGet, "/user /user/{id:uint64} /user/me", userHandler)
	// At the other side, with `Handle` we've had to write:
	// 	app.Handle(iris.MethodGet, "/user", userHandler)
	// 	app.Handle(iris.MethodGet, "/user/{id:uint64}", userHandler)
	// 	app.Handle(iris.MethodGet, "/user/me", userHandler)
	//
	// This method is used behind the scenes at the `Controller` function
	// in order to handle more than one paths for the same controller instance.
	HandleMany(method string, relativePath string, handlers ...context.Handler) []*Route

	// None registers an "offline" route
	// see context.ExecRoute(routeName) and
	// party.Routes().Online(handleResultregistry.*Route, "GET") and
	// Offline(handleResultregistry.*Route)
	//
	// Returns the read-only route information.
	None(path string, handlers ...context.Handler) *Route

	// Get registers a route for the Get http method.
	//
	// Returns the read-only route information.
	Get(path string, handlers ...context.Handler) *Route
	// Post registers a route for the Post http method.
	//
	// Returns the read-only route information.
	Post(path string, handlers ...context.Handler) *Route
	// Put registers a route for the Put http method.
	//
	// Returns the read-only route information.
	Put(path string, handlers ...context.Handler) *Route
	// Delete registers a route for the Delete http method.
	//
	// Returns the read-only route information.
	Delete(path string, handlers ...context.Handler) *Route
	// Connect registers a route for the Connect http method.
	//
	// Returns the read-only route information.
	Connect(path string, handlers ...context.Handler) *Route
	// Head registers a route for the Head http method.
	//
	// Returns the read-only route information.
	Head(path string, handlers ...context.Handler) *Route
	// Options registers a route for the Options http method.
	//
	// Returns the read-only route information.
	Options(path string, handlers ...context.Handler) *Route
	// Patch registers a route for the Patch http method.
	//
	// Returns the read-only route information.
	Patch(path string, handlers ...context.Handler) *Route
	// Trace registers a route for the Trace http method.
	//
	// Returns the read-only route information.
	Trace(path string, handlers ...context.Handler) *Route
	// Any registers a route for ALL of the http methods
	// (Get,Post,Put,Head,Patch,Options,Connect,Delete).
	Any(registeredPath string, handlers ...context.Handler) []*Route

	// StaticHandler returns a new Handler which is ready
	// to serve all kind of static files.
	//
	// Note:
	// The only difference from package-level `StaticHandler`
	// is that this `StaticHandler` receives a request path which
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
	StaticHandler(systemPath string, showList bool, gzip bool) context.Handler

	// StaticServe serves a directory as web resource
	// it's the simpliest form of the Static* functions
	// Almost same usage as StaticWeb
	// accepts only one required parameter which is the systemPath,
	// the same path will be used to register the GET and HEAD method routes.
	// If second parameter is empty, otherwise the requestPath is the second parameter
	// it uses gzip compression (compression on each request, no file cache).
	//
	// Returns the GET *Route.
	StaticServe(systemPath string, requestPath ...string) *Route
	// StaticContent registers a GET and HEAD method routes to the requestPath
	// that are ready to serve raw static bytes, memory cached.
	//
	// Returns the GET *Route.
	StaticContent(requestPath string, cType string, content []byte) *Route

	// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
	// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
	// Second parameter is the (virtual) directory path, for example "./assets"
	// Third parameter is the Asset function
	// Forth parameter is the AssetNames function.
	//
	// Returns the GET *Route.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-files-into-app
	StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) *Route
	// StaticEmbeddedGzip registers a route which can serve embedded gziped files
	// that are embedded using the https://github.com/kataras/bindata tool and only.
	// It's 8 times faster than the `StaticEmbeddedHandler` with `go-bindata` but
	// it sends gzip response only, so the client must be aware that is expecting a gzip body
	// (browsers and most modern browsers do that, so you can use it without fair).
	//
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-gziped-files-into-app
	StaticEmbeddedGzip(requestPath string, vdir string, gzipAssetFn func(name string) ([]byte, error), gzipNamesFn func() []string) *Route
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
	Favicon(favPath string, requestPath ...string) *Route
	// StaticWeb returns a handler that serves HTTP requests
	// with the contents of the file system rooted at directory.
	//
	// first parameter: the route path
	// second parameter: the system directory
	//
	// for more options look router.StaticHandler.
	//
	//     router.StaticWeb("/static", "./static")
	//
	// As a special case, the returned file server redirects any request
	// ending in "/index.html" to the same path, without the final
	// "index.html".
	//
	// StaticWeb calls the `StripPrefix(fullpath, NewStaticHandlerBuilder(systemPath).Listing(false).Build())`.
	//
	// Returns the GET *Route.
	StaticWeb(requestPath string, systemPath string) *Route

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
	Layout(tmplLayoutFile string) Party
}
