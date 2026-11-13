package router

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"

	"github.com/kataras/golog"
)

// BuildRouter builds the router, it's being called by the main Iris Application Builder.
// It's being called once after all routes have been registered and  before the server starts.
//
// A second call returns an error, unless the router was built with
// RouterOptions.EnableDynamicHandler: then it compiles the routes again.
func (router *RouterBuilder) BuildRouter(options RouterOptions, logger *golog.Logger, contextPool *context.Pool) error {
	if router.Downgraded() {
		return nil // already built, e.g. by `Downgrade` method call.
	}

	if root := router.GetRoot(); root.Handler != nil && root.ServeMux != nil {
		if !root.dynamic {
			// Built already, with a mux that has no lock: compiling the routes
			// into it again is not safe while requests are being served.
			return errors.New("the router is already built: set Options.Router.EnableDynamicHandler before the first build to register routes while the server runs")
		}

		// Built already: the mux in use rebuilds its routes and its 404 and 405 handlers.
		// The main handler, its wrappers and the pre-routing chain are built once:
		// requests read them with no lock, so they must not be written again while serving.
		noRoute := newEmptyRoute(root, NoRouteName, root.noRouteHandlers.Upsert(defaultHTTPErrorHandler))
		noMethod := newEmptyRoute(root, NoMethodName, root.noMethodHandlers.Upsert(defaultHTTPErrorHandler))
		return root.ServeMux.Build(noRoute, noMethod, root.routes)
	}

	if options.ForceLowercaseRouting {
		router.PrependRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			r.Host = strings.ToLower(r.Host)
			r.URL.Host = strings.ToLower(r.URL.Host)
			r.URL.Path = strings.ToLower(r.URL.Path)
			r.URL.RawPath = strings.ToLower(r.URL.RawPath)
			r.URL.RawQuery = strings.ToLower(r.URL.RawQuery)
			next(w, r)
		})
	}

	if options.AllowQuerySemicolons {
		router.PrependRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			// copy of net/http.AllowQuerySemicolons.
			if s := r.URL.RawQuery; strings.Contains(s, ";") {
				r2 := new(http.Request)
				*r2 = *r
				r2.URL = r.URL.Clone()
				r2.URL.RawQuery = strings.ReplaceAll(s, ";", "&")

				next(w, r2)
			} else {
				next(w, r)
			}
		})
	}

	// Build the error handler chain (404 and 405). See serve_mux#HandleRequest.
	// If no handlers are registered then the default handler is used, and we have to prepend the root's middlewares too.
	// If handlers exist, the middlewares already set by the HandleNoRoute&NoMethod methods but we still have
	// to append the default http error handler when not exist.
	root := router.GetRoot()

	noRouteHandlers := root.noRouteHandlers
	if noRouteHandlers.Count() == 0 {
		noRouteHandlers.Set(defaultHTTPErrorHandler).PrependList(root.handlers)
	} else {
		noRouteHandlers.Upsert(defaultHTTPErrorHandler)
	}

	noMethodHandlers := root.noMethodHandlers
	if noMethodHandlers.Count() == 0 {
		noMethodHandlers.Set(defaultHTTPErrorHandler).PrependList(root.handlers)
	} else {
		noMethodHandlers.Upsert(defaultHTTPErrorHandler)
	}

	// Make the pre routing, no route and no method routes so Context can know where it lives when necessary.
	noRoute := newEmptyRoute(root, NoRouteName, root.noRouteHandlers)
	noMethod := newEmptyRoute(root, NoMethodName, root.noMethodHandlers)
	preRoute := newEmptyRoute(root, PreRoutingName, root.routerHandlers)

	mux, err := BuildServeMux(&options, logger, preRoute, noRoute, noMethod, root.routes)
	if err != nil {
		return err
	}

	handler := router.makeDefaultHandlerFunc(contextPool)
	if wrapper := router.buildWrapper(); wrapper != nil {
		handler = wrapHandler(handler, wrapper)
	}

	// Updated by iris.TimeoutOptions and its handled on the main Iris Builder.
	// if timeout, msg := router.handlerTimeout, router.handlerTimeoutMsg; timeout > 0 {
	// 	mainHandler := handler
	// 	handler = http.TimeoutHandler(mainHandler, timeout, msg)
	// }

	// router.ContextPool = contextPool
	router.ServeMux = mux
	router.Handler = handler
	router.dynamic = options.EnableDynamicHandler

	return nil
}

func (router *RouterBuilder) makeDefaultHandlerFunc( /* do not pass the mux here, let the RouterBuilder.ServeMux be customizable. mux ServeMux*/ contextPool *context.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextPool.Acquire()
		writer := context.AcquireResponseWriter(w)
		ctx.SetResponseWriter(writer)
		ctx.SetRequest(r)

		router.ServeMux.HandleRequest(ctx)

		// The handler chain has returned. Anything still writing to this response
		// from another goroutine must stop now, while the writer is still this
		// request's: the next two lines nil it and hand the Context to whoever
		// asks next, and net/http cancels the request context only after this
		// function returns.
		ctx.RunReleaseHooks()

		context.ReleaseResponseWriter(writer)
		contextPool.Release(ctx)
	})
}

// buildWrapper builds and returns the main router's wrapper handler (if any).
func (router *RouterBuilder) buildWrapper() WrapperFunc {
	for _, w := range slices.Backward(router.wrapperFuncs) {
		if w == nil {
			continue
		}
		router.WrapRouter(w)
	}

	if router.wrapperFunc == nil {
		return nil
	}

	return router.wrapperFunc
}

// Downgrade "downgrades", alters the router supervisor service (the RouterBuilder's
// embedded http.Handler, set on `BuildRouter`)
// algorithm to a custom one,
// be aware of the 'ParamStart' (":") and 'WildcardParamStart' ("*") constants
// which the built-in router syntax is based on.
// can be used to implement a custom proxy or
// a custom router which should work with raw ResponseWriter, *Request
// instead of the Context(which again, can be retrieved by the Framework's context pool).
//
// Note: Downgrade will by-pass the Wrapper, the caller is responsible for everything.
// Downgrade is NOT safe for concurrent use: it writes the router's main handler
// without any synchronization, so it must be called on the design state,
// before the server starts serving.
func (router *RouterBuilder) Downgrade(newMainHandler http.HandlerFunc) {
	router.Handler = newMainHandler
}

// Downgraded returns true if this router is downgraded.
func (router *RouterBuilder) Downgraded() bool {
	return router.Handler != nil && router.ServeMux == nil
}

// WrapRouter adds a wrapper on the top of the main router.
// Usually it's useful for third-party middleware
// when need to wrap the entire application with a middleware like CORS.
//
// Developers can add more than one wrappers,
// those wrappers' execution comes from last to first.
// That means that the second wrapper will wrap the first, and so on.
//
// Before build.
func (router *RouterBuilder) WrapRouter(wrapperFunc WrapperFunc) {
	// logger := context.DefaultLogger("router wrapper")
	// file, line := context.HandlerFileLineRel(wrapperFunc)
	// if router.wrapperFunc != nil {
	// 	wrappedFile, wrappedLine := context.HandlerFileLineRel(router.wrapperFunc)
	// 	logger.Infof("%s:%d wraps %s:%d", file, line, wrappedFile, wrappedLine)
	// } else {
	// 	logger.Infof("%s:%d wraps the main router", file, line)
	// }
	router.wrapperFunc = joinWrapperFunc(router.wrapperFunc, wrapperFunc)
}

// AddRouterWrapper adds a router wrapper.
// Unlike `WrapRouter` the first registered will be executed first
// so a wrapper wraps its next not the previous one.
// it defers the wrapping until the `BuildRouter`.
// Redirection wrappers should be added using this method
// e.g. SubdomainRedirect.
func (router *RouterBuilder) AddRouterWrapper(wrapperFunc WrapperFunc) {
	router.wrapperFuncs = append(router.wrapperFuncs, wrapperFunc)
}

// PrependRouterWrapper like `AddRouterWrapper` but this wrapperFunc
// will always be executed before the previous `AddRouterWrapper`.
// Path form (no modification) wrappers should be added using this method
// e.g. ForceLowercaseRouting.
func (router *RouterBuilder) PrependRouterWrapper(wrapperFunc WrapperFunc) {
	router.wrapperFuncs = append([]WrapperFunc{wrapperFunc}, router.wrapperFuncs...)
}
