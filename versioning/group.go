package versioning

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
)

// Property to be defined inside the registered
// Party on NewGroup, useful for a party to know its (optional) version
// when the versioning feature is used.
const Property = "iris.party.version"

// API is a type alias of router.Party.
// This is required in order for a Group instance
// to implement the Party interface without field conflict.
type API = router.Party

// Group is a group of version-based routes.
// One version per one or more routes.
type Group struct {
	API

	// Information not currently in-use.
	version     string
	deprecation DeprecationOptions
}

// NewGroup returns a ptr to Group based on the given "version" constraint.
// Group completes the Party interface.
// The returned Group wraps a cloned Party of the given "r" Party therefore,
// any changes to its parent won't affect this one (e.g. register global middlewares afterwards).
//
// Examples at: _examples/routing/versioning
// Usage:
//  app := iris.New()
//  api := app.Party("/api")
//  v1 := versioning.NewGroup(api, ">= 1, < 2")
//  v1.Get/Post/Put/Delete...
//
// See the `GetVersion` function to learn how
// a version is extracted and matched over this.
func NewGroup(r router.Party, version string) *Group {
	r = r.Party("/")
	r.Properties()[Property] = version

	// Note that this feature alters the RouteRegisterRule to RouteOverlap
	// the RouteOverlap rule does not contain any performance downside
	// but it's good to know that if you registered other mode, this wanna change it.
	r.SetRegisterRule(router.RouteOverlap)
	r.UseOnce(Handler(version)) // this is required in order to not populate this middleware to the next group.

	return &Group{
		API:     r,
		version: version,
	}
}

// Deprecated marks this group and all its versioned routes
// as deprecated versions of that endpoint.
func (g *Group) Deprecated(options DeprecationOptions) *Group {
	// store it for future use, e.g. collect all deprecated APIs and notify the developer.
	g.deprecation = options

	g.API.UseOnce(func(ctx *context.Context) {
		WriteDeprecated(ctx, options)
		ctx.Next()
	})
	return g
}

// FromQuery is a simple helper which tries to
// set the version constraint from a given URL Query Parameter.
// The X-Api-Version is still valid.
func FromQuery(urlQueryParameterName string, defaultVersion string) context.Handler {
	return func(ctx *context.Context) {
		version := ctx.URLParam(urlQueryParameterName)
		if version == "" {
			version = defaultVersion
		}

		if version != "" {
			SetVersion(ctx, version)
		}

		ctx.Next()
	}
}
