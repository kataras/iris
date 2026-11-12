package context

import (
	"fmt"
	"net/http"
)

// FromStd converts native http.Handler & http.HandlerFunc to Handler.
//
// The standard handler reads the route parameters with r.PathValue and the
// matched route template, for example "/users/{id:uuid}", with r.Pattern.
//
// Under rest/errors.Handler, which iris.NewBuilder registers, a failure
// status the standard handler sets with no body is rendered as the JSON
// envelope unless the handler writes, flushes or declares a Content-Length.
//
// Supported form types:
//
//	.FromStd(h http.Handler)
//	.FromStd(h func(w http.ResponseWriter, r *http.Request) {...})
//	.FromStd(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {...})
func FromStd(handler any) Handler {
	h, err := TryFromStd(handler)
	if err != nil {
		panic(err)
	}

	return h
}

// TryFromStd converts native http.Handler & http.HandlerFunc to Handler.
// It returns an error if the conversion is not possible.
//
// Supported form types:
//
//	.TryFromStd(h http.Handler)
//	.TryFromStd(h func(w http.ResponseWriter, r *http.Request) {...})
//	.TryFromStd(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {...})
func TryFromStd(handler any) (Handler, error) {
	switch h := handler.(type) {
	case Handler:
		return h, nil
	// case func(*Context):
	// 	return h
	case http.Handler:
		// handlerFunc.ServeHTTP(w,r)
		return func(ctx *Context) {
			h.ServeHTTP(ctx.ResponseWriter(), stdRequest(ctx))
		}, nil
	case func(http.ResponseWriter, *http.Request):
		// handlerFunc(w,r)
		return FromStd(http.HandlerFunc(h)), nil
	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		// handlerFunc(w,r, http.HandlerFunc)
		//
		return FromStdWithNext(h), nil
	case func(http.Handler) http.Handler:
		return nil, fmt.Errorf(`
			Passed handler cannot be converted directly:
			- http.Handler(http.Handler)
			---------------------------------------------------------------------
			Please use the Application.WrapRouter method instead, example code:
			app := iris.New()
			// ...
			app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
			    httpThirdPartyHandler(router).ServeHTTP(w, r)
			})`)
	default:
		// No valid handler passed
		return nil, fmt.Errorf(`
			Passed argument is not a func(iris.Context) neither one of these types:
			- http.Handler
			- func(w http.ResponseWriter, r *http.Request)
			- func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
			---------------------------------------------------------------------
			It seems to be a %T points to: %v`, handler, handler)
	}
}

// FromStdWithNext receives a standard handler - middleware form - and returns a
// compatible Handler wrapper.
//
// LIMITATION: only the request the middleware passes to "next" is adopted, not
// the response writer. A standard middleware that works by wrapping the writer,
// such as output escaping, CSP header injection, compression or a response size
// limit, has no effect here. It fails silently too: the middleware still runs
// and still looks installed. Wrap the whole router with Application.WrapRouter
// instead, which keeps the writer the middleware hands down.
//
// Middleware that only reads the request, sets headers or short-circuits by not
// calling "next" converts correctly.
func FromStdWithNext(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)) Handler {
	return func(ctx *Context) {
		next := func(w http.ResponseWriter, r *http.Request) {
			ctx.SetRequest(r)
			ctx.Next()
		}

		h(ctx.ResponseWriter(), stdRequest(ctx), next)
	}
}

// stdRequest prepares the request a standard handler reads: the route
// parameters become path values (r.PathValue) and, when r.Pattern is empty,
// it is set to the matched route's path template as registered, for example
// "/users/{id:uuid}". The built-in 404 and 405 routes and the pre-routing
// slot leave r.Pattern empty.
func stdRequest(ctx *Context) *http.Request {
	r := ctx.Request()
	ctx.Params().ForEachString(r.SetPathValue)

	if r.Pattern == "" {
		if route := ctx.CurrentRoute(); route != nil && route.IsRoute() && !route.IsPreRouting() {
			r.Pattern = route.Tmpl().Src
		}
	}

	return r
}
