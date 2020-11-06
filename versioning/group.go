package versioning

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
)

// Group is a group of version-based routes.
// One version per one or more routes.
type Group struct {
	router.Party

	// Information not currently in-use.
	version     string
	deprecation DeprecationOptions
}

// NewGroup returns a ptr to Group based on the given "version".
// It sets the API Version for the "r" Party.
//
// See `Handle` for more.
//
// Example: _examples/routing/versioning
// Usage:
//  api := versioning.NewGroup(Parent_Party, ">= 1, < 2")
//  api.Get/Post/Put/Delete...
func NewGroup(r router.Party, version string) *Group {
	// Note that this feature alters the RouteRegisterRule to RouteOverlap
	// the RouteOverlap rule does not contain any performance downside
	// but it's good to know that if you registered other mode, this wanna change it.
	r.SetRegisterRule(router.RouteOverlap)
	r.UseOnce(Handler(version)) // this is required in order to not populate this middleware to the next group.

	return &Group{
		Party:   r,
		version: version,
	}
}

// Deprecated marks this group and all its versioned routes
// as deprecated versions of that endpoint.
func (g *Group) Deprecated(options DeprecationOptions) *Group {
	// store it for future use, e.g. collect all deprecated APIs and notify the developer.
	g.deprecation = options

	g.Party.UseOnce(func(ctx *context.Context) {
		WriteDeprecated(ctx, options)
		ctx.Next()
	})
	return g
}
