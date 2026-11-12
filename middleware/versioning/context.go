package versioning

import (
	"errors"
	"strings"

	"github.com/kataras/iris/v14/context"
)

const (
	// AcceptVersionHeaderKey is the header key of "Accept-Version".
	AcceptVersionHeaderKey = "Accept-Version"
	// AcceptHeaderKey is the header key of "Accept".
	AcceptHeaderKey = "Accept"
	// acceptHeaderVersionValue is the Accept's header value search term the requested version.
	acceptHeaderVersionValue = "version"
	// apiVersionResponseHeader the response header which its value contains
	// the normalized semver matched version.
	apiVersionResponseHeader = "X-Api-Version"
)

func fetchVersion(ctx *context.Context, headerKey string, queryParamName string, lookupSubdomain bool) string {
	// by context store, if manually set by a middleware.
	if version := Get(ctx); version != "" {
		return version
	}

	// by header, if any.
	if headerKey != "" {
		if version := fetchVersionFromHeader(ctx, headerKey); version != "" {
			return version
		}
	}

	// by query parameter, if any.
	if queryParamName != "" {
		if version := fetchVersionFromURLParam(ctx, queryParamName); version != "" {
			return version
		}
	}

	if lookupSubdomain {
		if version := fetchVersionFromSubdomain(ctx); version != "" {
			return version
		}
	}

	return ""
}

func fetchVersionFromHeader(ctx *context.Context, headerKey string) string {
	// secondly by the "Accept-Version" header.
	if version := ctx.Header(headerKey); version != "" {
		return version
	}

	// thirdly by the "Accept" header which is like"...; version=1.0"
	acceptValue := ctx.Header(AcceptHeaderKey)
	if acceptValue != "" {
		if idx := strings.Index(acceptValue, acceptHeaderVersionValue); idx != -1 {
			rem := acceptValue[idx:]
			startVersion := strings.Index(rem, "=")
			if startVersion == -1 || len(rem) < startVersion+1 {
				return ""
			}

			rem = rem[startVersion+1:]

			end := strings.Index(rem, " ")
			if end == -1 {
				end = strings.Index(rem, ";")
			}
			if end == -1 {
				end = len(rem)
			}

			if version := rem[:end]; version != "" {
				return version
			}
		}
	}

	return ""
}

func fetchVersionFromURLParam(ctx *context.Context, queryParamName string) string {
	return ctx.Query().Get(queryParamName)
}

var isDigit = func(r rune) bool {
	return r >= '0' && r <= '9'
}

func fetchVersionFromSubdomain(ctx *context.Context) string {
	subdomain := ctx.Subdomain()
	if len(subdomain) > 1 {
		if strings.HasPrefix(subdomain, "v") && strings.IndexFunc(subdomain[1:], isDigit) != -1 {
			return subdomain[1:]
		}
	}

	return ""
}

const (
	// versionContextKey is the context key for the version constraint.
	versionContextKey = "iris.api.version"
)

func setVersion(ctx *context.Context, version string) {
	ctx.SetHeader(apiVersionResponseHeader, version)
	ctx.Values().Set(versionContextKey, version)
}

// Get returns the version constraint from the context.
func Get(ctx *context.Context) string {
	return ctx.Values().GetString(versionContextKey)
}

const (
	// notFoundHandlerContextKey is the context key for the version not found handler.
	notFoundHandlerContextKey = "iris.api.version.notfound"
)

// ErrNotFound is returned when a version is not found.
var ErrNotFound = errors.New("version not found")

// DefaultNotFoundHandler is the default version not found handler.
// It returns a 501 Not Implemented status code.
var DefaultNotFoundHandler = func(ctx *context.Context) {
	ctx.StopWithError(501, ErrNotFound)
}

// WithNotFoundHandler sets the version not found handler to the context.
// Can be useful to customize the not found handler per route or per group of routes.
func WithNotFoundHandler(handler context.Handler) context.Handler {
	return func(ctx *context.Context) {
		setNotFoundHandler(ctx, handler)
		ctx.Next()
	}
}

func setNotFoundHandler(ctx *context.Context, handler context.Handler) {
	ctx.Values().Set(notFoundHandlerContextKey, handler)
}

func getNotFoundHandler(ctx *context.Context) context.Handler {
	if value := ctx.Values().Get(notFoundHandlerContextKey); value != nil {
		return value.(context.Handler)
	}

	return nil
}
