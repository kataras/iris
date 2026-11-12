// Package methodoverride lets clients that cannot issue PUT, PATCH or DELETE
// send a POST carrying the intended method instead.
//
// Its entry point is New(Options), which returns a Handler to register with
// Application.UseRouter. Read the SECURITY WARNING on New before using the
// zero-value Options.
package methodoverride

import (
	"net/http"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

func init() {
	context.SetHandlerName("iris/middleware/methodoverride.*", "iris.methodoverride")
}

// GetterFunc is the type signature for declaring custom logic
// to extract the method name which a POST request will be replaced with.
type GetterFunc func(*context.Context) string

// Options holds the configuration of the method override wrapper.
// The zero value is valid and reproduces the defaults documented on each field.
type Options struct {
	// Methods are the request methods that may be overridden.
	//
	// Defaults to POST only. Values are upper-cased.
	// Set to a non-nil empty slice to disable overriding entirely, which leaves
	// the middleware registered but inert.
	Methods []string

	// Headers a client can send to specify the method to override with,
	// checked in order and before FormField and Query.
	//
	// Defaults to X-HTTP-Method, X-HTTP-Method-Override and X-Method-Override.
	// Set to a non-nil empty slice to never read the method from a header.
	Headers []string

	// HeadersOnly restricts the override to the request headers, which a browser
	// cannot set cross-site without a preflight: FormField and Query are ignored.
	//
	// This is the recommended configuration; see the SECURITY WARNING on New.
	//
	// Defaults to false.
	HeadersOnly bool

	// FormField is the form field carrying the method to override with,
	// checked after Headers.
	//
	// Example field:
	//	<input type="hidden" name="_method" value="DELETE">
	//
	// Defaults to "_method". Ignored when HeadersOnly is true.
	FormField string

	// Query is the URL query parameter carrying the method to override with,
	// checked after FormField.
	//
	// Example URL query string:
	//	http://localhost:8080/path?_method=DELETE
	//
	// Defaults to "_method". Ignored when HeadersOnly is true.
	Query string

	// Getters are custom extractors, consulted in order after the built-in ones.
	// The first non-empty return value wins.
	//
	// Defaults to nil.
	Getters []GetterFunc

	// SaveOriginalMethodContextKey, when not empty, saves the original request
	// method on Context.Values() under this key; read it back with
	// ctx.Values().GetString(key).
	//
	// Defaults to empty: the original method is not saved.
	SaveOriginalMethodContextKey string
}

// build resolves the zero-value fields into the concrete getters and method list.
func (o Options) build() (methods []string, getters []GetterFunc) {
	if o.Methods == nil {
		methods = []string{http.MethodPost}
	} else {
		methods = make([]string, len(o.Methods))
		for i, m := range o.Methods {
			methods[i] = strings.ToUpper(m)
		}
	}

	headers := o.Headers
	if headers == nil {
		headers = []string{"X-HTTP-Method", "X-HTTP-Method-Override", "X-Method-Override"}
	}

	if len(headers) > 0 {
		getters = append(getters, headerGetter(headers))
	}

	if !o.HeadersOnly {
		formField := o.FormField
		if formField == "" {
			formField = "_method"
		}
		getters = append(getters, func(ctx *context.Context) string {
			return ctx.Form().Get(formField) // to parse the form.
		})

		query := o.Query
		if query == "" {
			query = "_method"
		}
		getters = append(getters, func(ctx *context.Context) string {
			return ctx.Query().Get(query)
		})
	}

	for _, getter := range o.Getters {
		if getter != nil {
			getters = append(getters, getter)
		}
	}

	return methods, getters
}

// headerGetter returns a GetterFunc reading the first non-empty of "headers".
func headerGetter(headers []string) GetterFunc {
	return func(ctx *context.Context) string {
		for _, s := range headers {
			if v := ctx.Header(s); v != "" {
				ctx.AddHeader("Vary", s)
				return v
			}
		}

		return ""
	}
}

// New returns a new method override wrapper
// which can be registered with `Application.UseRouter`.
//
// Use this wrapper when you expecting clients
// that do not support certain HTTP operations such as DELETE or PUT for security reasons.
// This wrapper will accept a method, based on criteria, to override the POST method with.
//
// Read more at:
// https://github.com/kataras/iris/issues/1325
// SECURITY WARNING: the defaults accept "_method" from the QUERY STRING and the
// FORM BODY, not only from a header.
//
// A cross-site form POST needs no preflight and carries the victim's cookies, so
//
//	<form method="POST" action="https://victim/api/account?_method=DELETE">
//
// reaches a route that only accepts DELETE - a route whose author may well have
// assumed a browser could never make a cross-site DELETE. A custom header does
// force a preflight, which is why HeadersOnly is the safe configuration:
//
//	app.UseRouter(methodoverride.New(methodoverride.Options{HeadersOnly: true}))
//
// The defaults are kept for backwards compatibility. Only POST is overridable,
// so a GET can never be turned into a DELETE.
func New(opts Options) context.Handler {
	methods, getters := opts.build()
	saveOriginalMethodContextKey := opts.SaveOriginalMethodContextKey

	get := func(ctx *context.Context) string {
		for _, getter := range getters {
			if v := getter(ctx); v != "" {
				return strings.ToUpper(v)
			}
		}

		return ""
	}

	return func(ctx *context.Context) {
		originalMethod := strings.ToUpper(ctx.Method())
		if slices.Contains(methods, originalMethod) {
			newMethod := get(ctx)
			// Only a known HTTP method may be installed. ctx.Header joins duplicate
			// headers with ", ", so two X-HTTP-Method headers used to yield the
			// method "GET, DELETE" - and any string at all could be set.
			if isKnownMethod(newMethod) {
				if saveOriginalMethodContextKey != "" {
					ctx.Values().Set(saveOriginalMethodContextKey, originalMethod)
				}

				ctx.Request().Method = newMethod
			}
		}

		ctx.Next()
	}
}

// knownMethods are the methods an override may install.
var knownMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace, router.MethodQuery,
}

// isKnownMethod reports whether "method" is a standard HTTP method.
func isKnownMethod(method string) bool {
	return slices.Contains(knownMethods, method)
}
