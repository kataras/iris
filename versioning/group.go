package versioning

import (
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"

	"github.com/blang/semver/v4"
)

// API is a type alias of router.Party.
// This is required in order for a Group instance
// to implement the Party interface without field conflict.
type API = router.Party

// Group represents a group of resources that should
// be handled based on a version requested by the client.
// See `NewGroup` for more.
type Group struct {
	API

	validate    semver.Range
	deprecation DeprecationOptions
}

// NewGroup returns a version Group based on the given "version" constraint.
// Group completes the Party interface.
// The returned Group wraps a cloned Party of the given "r" Party therefore,
// any changes to its parent won't affect this one (e.g. register global middlewares afterwards).
//
// A version is extracted through the versioning.GetVersion function:
//  Accept-Version: 1.0.0
//  Accept: application/json; version=1.0.0
// You can customize it by setting a version based on the request context:
//  api.Use(func(ctx *context.Context) {
// 	 if version := ctx.URLParam("version"); version != "" {
// 	  SetVersion(ctx, version)
// 	 }
//
//   ctx.Next()
//  })
// OR:
//  api.Use(versioning.FromQuery("version", ""))
//
// Examples at: _examples/routing/versioning
// Usage:
//  app := iris.New()
//  api := app.Party("/api")
//  v1 := versioning.NewGroup(api, ">=1.0.0 <2.0.0")
//  v1.Get/Post/Put/Delete...
//
// Valid ranges are:
//   - "<1.0.0"
//   - "<=1.0.0"
//   - ">1.0.0"
//   - ">=1.0.0"
//   - "1.0.0", "=1.0.0", "==1.0.0"
//   - "!1.0.0", "!=1.0.0"
//
// A Range can consist of multiple ranges separated by space:
// Ranges can be linked by logical AND:
//   - ">1.0.0 <2.0.0" would match between both ranges, so "1.1.1" and "1.8.7"
// but not "1.0.0" or "2.0.0"
//   - ">1.0.0 <3.0.0 !2.0.3-beta.2" would match every version between 1.0.0 and 3.0.0
// except 2.0.3-beta.2
//
// Ranges can also be linked by logical OR:
//   - "<2.0.0 || >=3.0.0" would match "1.x.x" and "3.x.x" but not "2.x.x"
//
// AND has a higher precedence than OR. It's not possible to use brackets.
//
// Ranges can be combined by both AND and OR
//
//  - `>1.0.0 <2.0.0 || >3.0.0 !4.2.1` would match `1.2.3`, `1.9.9`, `3.1.1`,
// but not `4.2.1`, `2.1.1`
func NewGroup(r API, version string) *Group {
	version = strings.ReplaceAll(version, ",", " ")
	version = strings.TrimSpace(version)

	verRange, err := semver.ParseRange(version)
	if err != nil {
		r.Logger().Errorf("versioning: %s: %s", r.GetRelPath(), strings.ToLower(err.Error()))
		return &Group{API: r}
	}

	// Clone this one.
	r = r.Party("/")

	// Note that this feature alters the RouteRegisterRule to RouteOverlap
	// the RouteOverlap rule does not contain any performance downside
	// but it's good to know that if you registered other mode, this wanna change it.
	r.SetRegisterRule(router.RouteOverlap)

	handler := makeHandler(verRange)
	// This is required in order to not populate this middleware to the next group.
	r.UseOnce(handler)
	// This is required for versioned custom error handlers,
	// of course if the parent registered one then this will do nothing.
	r.UseError(handler)

	return &Group{
		API:      r,
		validate: verRange,
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

func makeHandler(validate semver.Range) context.Handler {
	return func(ctx *context.Context) {
		if !matchVersionRange(ctx, validate) {
			// The overlapped handler has an exception
			// of a type of context.NotFound (which versioning.ErrNotFound wraps)
			// to clear the status code
			// and the error to ignore this
			// when available match version exists (see `NewGroup`).
			if h := NotFoundHandler; h != nil {
				h(ctx)
				return
			}
		}

		ctx.Next()
	}
}
