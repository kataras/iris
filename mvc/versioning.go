package mvc

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/versioning"
)

// Version returns a valid `Option` that can be passed to the `Application.Handle` method.
// It requires a specific "version" constraint for a Controller,
// e.g. ">1.0.0 <=2.0.0".
//
//
// Usage:
// 	m := mvc.New(dataRouter)
// 	m.Handle(new(v1Controller), mvc.Version("1.0.0"), mvc.Deprecated(mvc.DeprecationOptions{}))
// 	m.Handle(new(v2Controller), mvc.Version("2.3.0"))
// 	m.Handle(new(v3Controller), mvc.Version(">=3.0.0 <4.0.0"))
// 	m.Handle(new(noVersionController))
//
// See the `versioning` package's documentation for more information on
// how the version is extracted from incoming requests.
//
// Note that this Option will set the route register rule to `RouteOverlap`.
func Version(version string) OptionFunc {
	return func(c *ControllerActivator) {
		c.Router().SetRegisterRule(router.RouteOverlap) // required for this feature.
		// Note: Do not use a group, we need c.Use for the specific controller's routes.
		c.Use(versioning.Handler(version))
	}
}

// Deprecated marks a specific Controller as a deprecated one.
// Deprecated can be used to tell the clients that
// a newer version of that specific resource is available instead.
func Deprecated(options DeprecationOptions) OptionFunc {
	return func(c *ControllerActivator) {
		c.Use(func(ctx *context.Context) {
			versioning.WriteDeprecated(ctx, options)
			ctx.Next()
		})
	}
}
