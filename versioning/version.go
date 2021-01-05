package versioning

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/v12/context"
)

const (
	// APIVersionResponseHeader the response header which its value contains
	// the normalized semver matched version.
	APIVersionResponseHeader = "X-Api-Version"
	// AcceptVersionHeaderKey is the header key of "Accept-Version".
	AcceptVersionHeaderKey = "Accept-Version"
	// AcceptHeaderKey is the header key of "Accept".
	AcceptHeaderKey = "Accept"
	// AcceptHeaderVersionValue is the Accept's header value search term the requested version.
	AcceptHeaderVersionValue = "version"
	// NotFound is the key that can be used inside a `Map` or inside `ctx.SetVersion(versioning.NotFound)`
	// to tell that a version wasn't found, therefore the `NotFoundHandler` should handle the request instead.
	NotFound = "iris.api.version.notfound"
	// Empty is just an empty string. Can be used as a key for a version alias
	// when the requested version of a resource was not even specified by the client.
	// The difference between NotFound and Empty is important when version aliases are registered:
	// - A NotFound cannot be registered as version alias, it
	//   means that the client sent a version with its request
	//   but that version was not implemented by the server.
	// - An Empty indicates that the client didn't send any version at all.
	Empty = ""
)

// ErrNotFound reports whether a requested version
// does not match with any of the server's implemented ones.
var ErrNotFound = fmt.Errorf("version %w", context.ErrNotFound)

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
// Example of how you can change the default behavior to extract a requested version (which is by headers)
// from a "version" url parameter instead:
//  func(ctx iris.Context) { // &version=1
//   version := ctx.URLParamDefault("version", "1")
//   versioning.SetVersion(ctx, version)
// 	 ctx.Next()
//  }
// See `GetVersion` too.
func SetVersion(ctx *context.Context, constraint string) {
	ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetVersionContextKey(), constraint)
}

// AliasMap is just a type alias of the standard map[string]string.
// Head over to the `Aliases` function below for more.
type AliasMap = map[string]string

// Aliases is a middleware which registers version constraint aliases
// for the children Parties(routers). It's respected by versioning Groups.
//
// Example Code:
//  app := iris.New()
//
//  api := app.Party("/api")
//  api.Use(Aliases(map[string]string{
//   versioning.Empty: "1", // when no version was provided by the client.
//   "beta": "4.0.0",
//   "stage": "5.0.0-alpha"
//  }))
//
//  v1 := NewGroup(api, ">= 1, < 2")
//  v1.Get/Post...
//
//  v4 := NewGroup(api, ">= 4, < 5")
//  v4.Get/Post...
//
//  stage := NewGroup(api, "5.0.0-alpha")
//  stage.Get/Post...
func Aliases(aliases AliasMap) context.Handler {
	cp := make(AliasMap, len(aliases)) // copy the map here so we are safe of later modifications by end-dev.
	for k, v := range aliases {
		cp[k] = v
	}

	return func(ctx *context.Context) {
		SetVersionAliases(ctx, cp, true)
		ctx.Next()
	}
}

// GetVersionAlias returns the version alias of the given "gotVersion"
// or empty. It Reports whether the alias was found.
// See `SetVersionAliases`, `Aliases` and `Match` for more.
func GetVersionAlias(ctx *context.Context, gotVersion string) (string, bool) {
	key := ctx.Application().ConfigurationReadOnly().GetVersionAliasesContextKey()
	if key == "" {
		return "", false
	}

	v := ctx.Values().Get(key)
	if v == nil {
		return "", false
	}

	aliases, ok := v.(AliasMap)
	if !ok {
		return "", false
	}

	version, ok := aliases[gotVersion]
	if !ok {
		return "", false
	}

	return strings.TrimSpace(version), true
}

// SetVersionAliases sets a map of version aliases when a requested
// version of a resource was not implemented by the server.
// Can be used inside a middleware to the parent Party
// and always before the child versioning groups (see `Aliases` function).
//
// The map's key (string) should be the "got version" (by the client)
// and the value should be the "version constraint to match" instead.
// The map's value(string) should be a registered version
// otherwise it will hit the NotFoundHandler (501, "version not found" by default).
//
// The given "aliases" is a type of standard map[string]string and
// should NOT be modified afterwards.
//
// The last "override" input argument indicates whether any
// existing aliases, registered by previous handlers in the chain,
// should be overriden or copied to the previous map one.
func SetVersionAliases(ctx *context.Context, aliases AliasMap, override bool) {
	key := ctx.Application().ConfigurationReadOnly().GetVersionAliasesContextKey()
	if key == "" {
		return
	}

	v := ctx.Values().Get(key)
	if v == nil || override {
		ctx.Values().Set(key, aliases)
		return
	}

	if existing, ok := v.(AliasMap); ok {
		for k, v := range aliases {
			existing[k] = v
		}
	}
}
