// Package versioning routes a request to a handler by the API version it asks
// for, through the Accept-Version header, the Accept header or a custom source.
//
// Its entry point is New(Options), which returns a Handler that matches a
// version to one of the registered handlers. See also `NewMatcher` and the
// `Deprecated` helper.
package versioning

import (
	"strings"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/versioning.*", "iris.versioning")
}

const latestLiteral = "latest"

type (
	// Options is the versioning handler options.
	Options struct {
		// Default is the default version. It is optional.
		// Defaults to the Latest value when Latest is provided, otherwise it stays empty,
		// which intentionally disables the version matching for requests without a version.
		// In strict mode it is always cleared.
		Default string `json:"default" yaml:"Default"`
		// Latest is the latest version. It is optional.
		// If provided, the client can request for the latest version by using the "latest" value.
		// E.g Latest: "3.0.0".
		Latest string `json:"latest" yaml:"Latest"`
		// Strict is the strict mode. It is optional.
		// If true, the Default field has no effect, and the client must request for a specific version, otherwise the router will not be called.
		Strict bool `json:"strict" yaml:"Strict"`

		// Versions is the list of versions. It is optional.
		// The order of the versions matters.
		// The first version that matches the request version is used.
		// If no version matches the request version, the request is not found.
		// If a version is deprecated, the deprecation headers are written to the response.
		//
		// If no provided then all versions will be redirected to the default path pattern which is: "/v$VERSION_MAJOR/$REQUEST_PATH".
		Versions Versions `json:"versions" yaml:"Versions"`

		// Deprecation is the deprecation options. It is optional.
		Deprecation DeprecationOptions `json:"deprecation" yaml:"Deprecation"`
		// Header is the version header key. It is optional.
		// Defaults to "Accept-Version".
		Header string `json:"header" yaml:"Header"`
		// QueryParam is the version query parameter key. It is optional.
		// Defaults to "v".
		QueryParam string `json:"query_param" yaml:"QueryParam"`

		// Subdomain specifies if the framework should lookup subdomain for a version, the form of it should be "v1" ("v" + digit). It is optional.
		// Defaults to false.
		Subdomain bool `json:"subdomain" yaml:"Subdomain"`

		// NotFoundHandler is the version not found handler. It is optional.
		// Defaults to DefaultNotFoundHandler (501 Not Implemented with ErrNotFound).
		NotFoundHandler context.Handler `json:"-" yaml:"-"`
		// Skipper is a function which if it's provided it can report whether the versioning middleware
		// runs or not on specific requests (e.g. by path prefix). It is optional.
		// If it returns true then the versioning middleware must not run.
		// Defaults to nil.
		// Usage:
		// versioning.Options{
		// 	Skipper: func(ctx iris.Context) bool {
		// 		return !strings.HasPrefix(ctx.Path(), "/api")
		// 	},
		// }
		Skipper func(*context.Context) bool `json:"-" yaml:"-"`
	}

	// Version describes a version and the redirect path.
	// It is used to match the request version with the provided version range.
	// If the version is found, the request path is redirected to the provided path pattern.
	Version struct {
		Range      string `json:"range" yaml:"Range"`
		Path       string `json:"path" yaml:"Path"`
		Deprecated bool   `json:"deprecated" yaml:"Deprecated"`
	}

	// Versions is a list of versions.
	Versions = []Version
)

// New returns a new versioning router handler. Register it through the `app.UseRouter` method.
// It matches the request version with the provided versions.
// If the version is not found, the not found handler is called.
// If the version is deprecated, the deprecation headers are written to the response.
// If the version is "latest" then the latest provided version is used.
// If the version is not provided at all and the strict mode is enabled, the not found handler is called, otherwise it continues to the router.
// If no versions are provided, all versions are matched with the default path pattern. The default pattern is: "/v$VERSION_MAJOR/$REQUEST_PATH".
// Usage:
//
//	ver := versioning.New(versioning.Options{
//		Default: "1.0.0",
//		Latest: "2.0.0",
//		Strict: false,
//		Header: "Accept-Version",
//		QueryParam: "v",
//		Subdomain: false,
//		NotFoundHandler: func(ctx iris.Context) {...},
//		Skipper: func(ctx iris.Context) bool { ...},
//		Deprecation: versioning.DeprecationOptions{
//			WarnMessage: "WARNING! You are using a deprecated version of this API.",
//			DeprecationDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
//			DeprecationInfo: "",
//		},
//		Versions: versioning.Versions{
//			{
//				Range:      ">=1.0.0 <2.0.0",
//				Path:       "/v1", // or, the same effect by using pattern: "/v$1/$4" to redirect to the same path but with the version prefix.
//				Deprecated: false,
//			},
//		},
//	})
//	app.UseRouter(ver)
func New(opts Options) context.Handler {
	opts.prepare()

	expressions := parseExpressions(opts.Versions)
	matcher := newMatcher(expressions)

	return func(ctx *context.Context) {
		if skipper := opts.Skipper; skipper != nil && skipper(ctx) {
			ctx.Next()
			return
		}

		version := fetchVersion(ctx, opts.Header, opts.QueryParam, opts.Subdomain)
		if version == latestLiteral { // if the version is "latest" then redirect to the latest version.
			version = opts.Latest
		}
		if version == "" { // no version is requested or the version is "latest" but the latest version is not provided.
			version = opts.Default // set the default version.
		}

		if version == "" { // no version is requested and default is empty.
			if opts.Strict {
				opts.handleNotFound(ctx) // requested version not found.
				return
			}

			// if not strict and no version is provided then just continue to the next handler, which is the routers.
			ctx.Next()
			return
		}

		requestPath := ctx.Request().URL.Path
		newRequestPath, parsedVersion, deprecated, ok := matcher.Match(version, requestPath)
		if !ok {
			opts.handleNotFound(ctx) // requested version not found.
			return
		}

		if deprecated {
			opts.Deprecation.WriteTo(ctx.ResponseWriter()) // write the deprecation headers.
		}

		ctx.Request().URL.Path = strings.TrimSuffix(newRequestPath, "/") // set the new path to the request.
		setVersion(ctx, parsedVersion.String())                          // set the version to the context so the user can access it.
		ctx.Next()                                                       // procceed to the router.

		// if ctx.CurrentRoute().IsNoRoute() { means that the version matched but the route not found. Let the router handle it.
	}
}

func (opts *Options) prepare() {
	// Let's allow it empty, in order to disable the versioning on non-specific request version if needed.
	// if opts.Default == "" {
	// 	opts.Default = "1.0.0"
	// }

	if opts.Strict {
		opts.Default = ""
	} else { // set the default to latest if not strict and provided and vice versa.
		if opts.Default == "" {
			opts.Default = opts.Latest
		} else if opts.Latest == "" {
			opts.Latest = opts.Default
		}
	}

	if opts.Header == "" {
		opts.Header = AcceptVersionHeaderKey
	}

	if opts.QueryParam == "" {
		opts.QueryParam = "v"
	}

	if opts.NotFoundHandler == nil {
		opts.NotFoundHandler = DefaultNotFoundHandler
	}

	if !opts.Deprecation.ShouldHandle() {
		opts.Deprecation = DefaultDeprecationOptions
	}

	if len(opts.Versions) == 0 {
		opts.Versions = append(opts.Versions, Version{ // if no versions are provided, match all versions with v$VERSION_MAJOR/$REQUEST_PATH pattern.
			Range:      ">=0.0.0",
			Path:       defaultPathPattern, // default path pattern.
			Deprecated: false,
		})
	}
}

func (opts *Options) handleNotFound(ctx *context.Context) {
	if handler := getNotFoundHandler(ctx); handler != nil {
		handler(ctx)
	}

	opts.NotFoundHandler(ctx)
}
