package router

import (
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/macro"
)

// APIContainer is a wrapper of a common `Party` featured by Dependency Injection.
// See `Party.ConfigureContainer` for more.
type APIContainer struct {
	// Self returns the original `Party` without DI features.
	Self Party

	// Container is the per-party (and its children gets a clone) DI container..
	Container *hero.Container
}

// Party returns a child of this `APIContainer` featured with Dependency Injection.
// Like the `Self.Party` method does for the common Router Groups.
func (api *APIContainer) Party(relativePath string, handlersFn ...interface{}) *APIContainer {
	handlers := api.convertHandlerFuncs(relativePath, handlersFn...)
	p := api.Self.Party(relativePath, handlers...)
	return p.ConfigureContainer()
}

// PartyFunc same as `Party` but it accepts a party builder function instead.
// Returns the new Party's APIContainer
func (api *APIContainer) PartyFunc(relativePath string, fn func(*APIContainer)) *APIContainer {
	childContainer := api.Party(relativePath)
	fn(childContainer)
	return childContainer
}

// OnError adds an error handler for this Party's DI Hero Container and its handlers (or controllers).
// The "errorHandler" handles any error may occurred and returned
// during dependencies injection of the Party's hero handlers or from the handlers themselves.
//
// Same as:
// Container.GetErrorHandler = func(ctx iris.Context) hero.ErrorHandler { return errorHandler }
//
// See `RegisterDependency`, `Use`, `Done` and `Handle` too.
func (api *APIContainer) OnError(errorHandler func(*context.Context, error)) {
	errHandler := hero.ErrorHandlerFunc(errorHandler)
	api.Container.GetErrorHandler = func(ctx *context.Context) hero.ErrorHandler {
		return errHandler
	}
}

// RegisterDependency adds a dependency.
// The value can be a single struct value or a function.
// Follow the rules:
// * <T>{structValue}
// * func(accepts <T>)                                 returns <D> or (<D>, error)
// * func(accepts iris.Context)                        returns <D> or (<D>, error)
//
// A Dependency can accept a previous registered dependency and return a new one or the same updated.
// * func(accepts1 <D>, accepts2 <T>)                  returns <E> or (<E>, error) or error
// * func(acceptsPathParameter1 string, id uint64)     returns <T> or (<T>, error)
//
// Usage:
//
// - RegisterDependency(loggerService{prefix: "dev"})
// - RegisterDependency(func(ctx iris.Context) User {...})
// - RegisterDependency(func(User) OtherResponse {...})
//
// See `OnError`, `Use`, `Done` and `Handle` too.
func (api *APIContainer) RegisterDependency(dependency interface{}) *hero.Dependency {
	return api.Container.Register(dependency)
}

// UseResultHandler adds a result handler to the Container.
// A result handler can be used to inject the returned struct value
// from a request handler or to replace the default renderer.
func (api *APIContainer) UseResultHandler(handler func(next hero.ResultHandler) hero.ResultHandler) *APIContainer {
	api.Container.UseResultHandler(handler)
	return api
}

// EnableStrictMode sets the container's DisablePayloadAutoBinding and MarkExportedFieldsAsRequired to true.
// Meaning that all struct's  fields (or function's parameters) should be binded manually (except the path parameters).
//
// Note that children will clone the same properties.
// Call the same method with `false` for children
// to enable automatic binding on missing dependencies.
//
// Strict mode is disabled by default;
// structs or path parameters that don't match to registered dependencies
// are automatically binded from the request context (body and url path parameters respectfully).
func (api *APIContainer) EnableStrictMode(strictMode bool) *APIContainer {
	api.Container.DisablePayloadAutoBinding = strictMode
	api.Container.MarkExportedFieldsAsRequired = strictMode
	return api
}

// EnableStructDependents sets the container's EnableStructDependents to true.
// It's used to automatically fill the dependencies of a struct's fields
// based on the previous registered dependencies, just like function inputs.
func (api *APIContainer) EnableStructDependents() *APIContainer {
	api.Container.EnableStructDependents = true
	return api
}

// SetDependencyMatcher replaces the function that compares equality between
// a dependency and an input (struct field or function parameter).
//
// Defaults to hero.DefaultMatchDependencyFunc.
func (api *APIContainer) SetDependencyMatcher(fn hero.DependencyMatcher) *APIContainer {
	if fn == nil {
		panic("api container: set dependency matcher: fn cannot be nil")
	}

	api.Container.DependencyMatcher = fn
	return api
}

// convertHandlerFuncs accepts Iris hero handlers and returns a slice of native Iris handlers.
func (api *APIContainer) convertHandlerFuncs(relativePath string, handlersFn ...interface{}) context.Handlers {
	fullpath := api.Self.GetRelPath() + relativePath
	paramsCount := macro.CountParams(fullpath, *api.Self.Macros())

	handlers := make(context.Handlers, 0, len(handlersFn))
	for _, h := range handlersFn {
		handlers = append(handlers, api.Container.HandlerWithParams(h, paramsCount))
	}

	// Note: let end-developer to decide that through Party.SetExecutionRules.
	// On that type of handlers the end-developer does not have to include the Context in the handler,
	// so the ctx.Next is automatically called unless an `ErrStopExecution` returned (implementation inside hero pkg).
	//
	// o := ExecutionOptions{Force: true}
	// o.apply(&handlers)

	return handlers
}

func fixRouteInfo(route *Route, handlersFn []interface{}) {
	// Fix main handler name and source modified by execution rules wrapper.
	route.MainHandlerName, route.MainHandlerIndex = context.MainHandlerName(handlersFn...)
	if len(handlersFn) > route.MainHandlerIndex {
		route.SourceFileName, route.SourceLineNumber = context.HandlerFileLineRel(handlersFn[route.MainHandlerIndex])
	}
}

// Handler receives a function which can receive dependencies and output result
// and returns a common Iris Handler, useful for Versioning API integration otherwise
// the `Handle/Get/Post...` methods are preferable.
func (api *APIContainer) Handler(handlerFn interface{}, handlerParamsCount int) context.Handler {
	paramsCount := macro.CountParams(api.Self.GetRelPath(), *api.Self.Macros()) + handlerParamsCount
	return api.Container.HandlerWithParams(handlerFn, paramsCount)
}

// Use same as `Self.Use` but it accepts dynamic functions as its "handlersFn" input.
//
// See `OnError`, `RegisterDependency`, `Done` and `Handle` for more.
func (api *APIContainer) Use(handlersFn ...interface{}) {
	handlers := api.convertHandlerFuncs("/", handlersFn...)
	api.Self.Use(handlers...)
}

// Done same as `Self.Done` but it accepts dynamic functions as its "handlersFn" input.
// See `OnError`, `RegisterDependency`, `Use` and `Handle` for more.
func (api *APIContainer) Done(handlersFn ...interface{}) {
	handlers := api.convertHandlerFuncs("/", handlersFn...)
	api.Self.Done(handlers...)
}

// Handle same as `Self.Handle` but it accepts one or more "handlersFn" functions which each one of them
// can accept any input arguments that match with the Party's registered Container's `Dependencies` and
// any output result; like custom structs <T>, string, []byte, int, error,
// a combination of the above, hero.Result(hero.View | hero.Response) and more.
//
// It's common from a hero handler to not even need to accept a `Context`, for that reason,
// the "handlersFn" will call `ctx.Next()` automatically when not called manually.
// To stop the execution and not continue to the next "handlersFn"
// the end-developer should output an error and return `iris.ErrStopExecution`.
//
// See `OnError`, `RegisterDependency`, `Use`, `Done`, `Get`, `Post`, `Put`, `Patch` and `Delete` too.
func (api *APIContainer) Handle(method, relativePath string, handlersFn ...interface{}) *Route {
	handlers := api.convertHandlerFuncs(relativePath, handlersFn...)
	route := api.Self.Handle(method, relativePath, handlers...)
	fixRouteInfo(route, handlersFn)
	return route
}

// Get registers a route for the Get HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Get(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodGet, relativePath, handlersFn...)
}

// Post registers a route for the Post HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Post(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodPost, relativePath, handlersFn...)
}

// Put registers a route for the Put HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Put(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodPut, relativePath, handlersFn...)
}

// Delete registers a route for the Delete HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Delete(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodDelete, relativePath, handlersFn...)
}

// Connect registers a route for the Connect HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Connect(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodConnect, relativePath, handlersFn...)
}

// Head registers a route for the Head HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Head(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodHead, relativePath, handlersFn...)
}

// Options registers a route for the Options HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Options(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodOptions, relativePath, handlersFn...)
}

// Patch registers a route for the Patch HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Patch(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodPatch, relativePath, handlersFn...)
}

// Trace registers a route for the Trace HTTP Method.
//
// Returns a *Route and an error which will be filled if route wasn't registered successfully.
func (api *APIContainer) Trace(relativePath string, handlersFn ...interface{}) *Route {
	return api.Handle(http.MethodTrace, relativePath, handlersFn...)
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
func (api *APIContainer) Any(relativePath string, handlersFn ...interface{}) (routes []*Route) {
	handlers := api.convertHandlerFuncs(relativePath, handlersFn...)

	for _, m := range AllMethods {
		r := api.Self.HandleMany(m, relativePath, handlers...)
		routes = append(routes, r...)
	}

	return
}

/* TODO: fix those

// OnErrorCode registers a handlers chain for this `Party` for a specific HTTP status code.
// Read more at: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
// Look `OnAnyErrorCode` too.
func (api *APIContainer) OnErrorCode(statusCode int, handlersFn ...interface{}) []*Route {
	handlers := api.convertHandlerFuncs("/{tail:path}", handlersFn...)
	return api.Self.OnErrorCode(statusCode, handlers...)
}

// OnAnyErrorCode registers a handlers chain for all error codes
// (4xxx and 5xxx, change the `ClientErrorCodes` and `ServerErrorCodes` variables to modify those)
// Look `OnErrorCode` too.
func (api *APIContainer) OnAnyErrorCode(handlersFn ...interface{}) []*Route {
	handlers := api.convertHandlerFuncs("/{tail:path}", handlersFn...)
	return api.Self.OnAnyErrorCode(handlers...)
}
*/
