package versioning

import (
	"errors"
	"strings"

	"github.com/kataras/iris/context"
)

const (
	// AcceptVersionHeaderKey is the header key of "Accept-Version".
	AcceptVersionHeaderKey = "Accept-Version"
	// AcceptHeaderKey is the header key of "Accept".
	AcceptHeaderKey = "Accept"
	// AcceptHeaderVersionValue is the Accept's header value search term the requested version.
	AcceptHeaderVersionValue = "version"

	// Key is the context key of the version, can be used to manually modify the "requested" version.
	// Example of how you can change the default behavior to extract a requested version (which is by headers)
	// from a "version" url parameter instead:
	// func(ctx iris.Context) { // &version=1
	// 	ctx.Values().Set(versioning.Key, ctx.URLParamDefault("version", "1"))
	// 	ctx.Next()
	// }
	//
	// DEPRECATED: Use:
	// version := ctx.URLParamDefault("version", "1")
	// versioning.SetVersion(ctx, version) instead.
	Key = "iris.api.version"
	// NotFound is the key that can be used inside a `Map` or inside `ctx.SetVersion(versioning.NotFound)`
	// to tell that a version wasn't found, therefore the not found handler should handle the request instead.
	NotFound = "iris.api.version.notfound"
)

// ErrNotFound reports whether a requested version
// does not match with any of the server's implemented ones.
var ErrNotFound = errors.New("version not found")

// NotFoundHandler is the default version not found handler that
// is executed from `NewMatcher` when no version is registered as available to dispatch a resource.
var NotFoundHandler = func(ctx *context.Context) {
	// 303 is an option too,
	// end-dev has the chance to change that behavior by using the NotFound in the map:
	//
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
	/*
		10.5.2 501 Not Implemented

		The server does not support the functionality required to fulfill the request.
		This is the appropriate response when the server does not
		recognize the request method and is not capable of supporting it for any resource.
	*/

	ctx.StopWithPlainError(501, ErrNotFound)
}

// GetVersion returns the current request version.
//
// By default the `GetVersion` will try to read from:
// - "Accept" header, i.e Accept: "application/json; version=1.0"
// - "Accept-Version" header, i.e Accept-Version: "1.0"
//
// However, the end developer can also set a custom version for a handler via a middleware by using the context's store key
// for versions (see `Key` for further details on that).
//
// See `SetVersion` too.
func GetVersion(ctx *context.Context) string {
	// firstly by context store, if manually set by a middleware.
	version := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetVersionContextKey())
	if version != "" {
		return version
	}

	// secondly by the "Accept-Version" header.
	if version = ctx.GetHeader(AcceptVersionHeaderKey); version != "" {
		return version
	}

	// thirdly by the "Accept" header which is like"...; version=1.0"
	acceptValue := ctx.GetHeader(AcceptHeaderKey)
	if acceptValue != "" {
		if idx := strings.Index(acceptValue, AcceptHeaderVersionValue); idx != -1 {
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

			if version = rem[:end]; version != "" {
				return version
			}
		}
	}

	return ""
}

// SetVersion force-sets the API Version.
// It can be used inside a middleware.
// See `GetVersion` too.
func SetVersion(ctx *context.Context, constraint string) {
	ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetVersionContextKey(), constraint)
}
