package methodoverride

import (
	stdContext "context"
	"net/http"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
)

type options struct {
	getters                      []GetterFunc
	methods                      []string
	saveOriginalMethodContextKey interface{} // if not nil original value will be saved.
}

func (o *options) configure(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func (o *options) canOverride(method string) bool {
	for _, s := range o.methods {
		if s == method {
			return true
		}
	}

	return false
}

func (o *options) get(w http.ResponseWriter, r *http.Request) string {
	for _, getter := range o.getters {
		if v := getter(w, r); v != "" {
			return strings.ToUpper(v)
		}
	}

	return ""
}

// Option sets options for a fresh method override wrapper.
// See `New` package-level function for more.
type Option func(*options)

// Methods can be used to add methods that can be overridden.
// Defaults to "POST".
func Methods(methods ...string) Option {
	for i, s := range methods {
		methods[i] = strings.ToUpper(s)
	}

	return func(opts *options) {
		opts.methods = append(opts.methods, methods...)
	}
}

// SaveOriginalMethod will save the original method
// on Context.Request().Context().Value(requestContextKey).
//
// Defaults to nil, don't save it.
func SaveOriginalMethod(requestContextKey interface{}) Option {
	return func(opts *options) {
		if requestContextKey == nil {
			opts.saveOriginalMethodContextKey = nil
		}
		opts.saveOriginalMethodContextKey = requestContextKey
	}
}

// GetterFunc is the type signature for declaring custom logic
// to extract the method name which a POST request will be replaced with.
type GetterFunc func(http.ResponseWriter, *http.Request) string

// Getter sets a custom logic to use to extract the method name
// to override the POST method with.
// Defaults to nil.
func Getter(customFunc GetterFunc) Option {
	return func(opts *options) {
		opts.getters = append(opts.getters, customFunc)
	}
}

// Headers that client can send to specify a method
// to override the POST method with.
//
// Defaults to:
// X-HTTP-Method
// X-HTTP-Method-Override
// X-Method-Override
func Headers(headers ...string) Option {
	getter := func(w http.ResponseWriter, r *http.Request) string {
		for _, s := range headers {
			if v := r.Header.Get(s); v != "" {
				w.Header().Add("Vary", s)
				return v
			}
		}

		return ""
	}

	return Getter(getter)
}

// FormField specifies a form field to use to determinate the method
// to override the POST method with.
//
// Example Field:
// <input type="hidden" name="_method" value="DELETE">
//
// Defaults to: "_method".
func FormField(fieldName string) Option {
	return FormFieldWithConf(fieldName, nil)
}

// FormFieldWithConf same as `FormField` but it accepts the application's
// configuration to parse the form based on the app core configuration.
func FormFieldWithConf(fieldName string, conf context.ConfigurationReadOnly) Option {
	var (
		postMaxMemory int64 = 32 << 20 // 32 MB
		resetBody           = false
	)

	if conf != nil {
		postMaxMemory = conf.GetPostMaxMemory()
		resetBody = conf.GetDisableBodyConsumptionOnUnmarshal()
	}

	getter := func(w http.ResponseWriter, r *http.Request) string {
		return context.FormValueDefault(r, fieldName, "", postMaxMemory, resetBody)
	}

	return Getter(getter)
}

// Query specifies a url parameter name to use to determinate the method
// to override the POST methos with.
//
// Example URL Query string:
// http://localhost:8080/path?_method=DELETE
//
// Defaults to: "_method".
func Query(paramName string) Option {
	getter := func(w http.ResponseWriter, r *http.Request) string {
		return r.URL.Query().Get(paramName)
	}

	return Getter(getter)
}

// Only clears all default or previously registered values
// and uses only the "o" option(s).
//
// The default behavior is to check for all the following by order:
// headers, form field, query string
// and any custom getter (if set).
// Use this method to override that
// behavior and use only the passed option(s)
// to determinate the method to override with.
//
// Use cases:
// 1. When need to check only for headers and ignore other fields:
//   New(Only(Headers("X-Custom-Header")))
//
// 2. When need to check only for (first) form field and (second) custom getter:
//   New(Only(FormField("fieldName"), Getter(...)))
func Only(o ...Option) Option {
	return func(opts *options) {
		opts.getters = opts.getters[0:0]
		opts.configure(o...)
	}
}

// New returns a new method override wrapper
// which can be registered with `Application.WrapRouter`.
//
// Use this wrapper when you expecting clients
// that do not support certain HTTP operations such as DELETE or PUT for security reasons.
// This wrapper will accept a method, based on criteria, to override the POST method with.
//
//
// Read more at:
// https://github.com/kataras/iris/issues/1325
func New(opt ...Option) router.WrapperFunc {
	opts := new(options)
	// Default values.
	opts.configure(
		Methods(http.MethodPost),
		Headers("X-HTTP-Method", "X-HTTP-Method-Override", "X-Method-Override"),
		FormField("_method"),
		Query("_method"),
	)
	opts.configure(opt...)

	return func(w http.ResponseWriter, r *http.Request, proceed http.HandlerFunc) {
		originalMethod := strings.ToUpper(r.Method)
		if opts.canOverride(originalMethod) {
			newMethod := opts.get(w, r)
			if newMethod != "" {
				if opts.saveOriginalMethodContextKey != nil {
					r = r.WithContext(stdContext.WithValue(r.Context(), opts.saveOriginalMethodContextKey, originalMethod))
				}
				r.Method = newMethod
			}
		}

		proceed(w, r)
	}
}
