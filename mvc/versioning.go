package mvc

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/versioning"
)

// Version returns a valid `Option` that can be passed to the `Application.Handle` method.
// It requires a specific "version" constraint for a Controller,
// e.g. ">1, <=2" or just "1".
//
//
// Usage:
// 	m := mvc.New(dataRouter)
// 	m.Handle(new(v1Controller), mvc.Version("1"))
// 	m.Handle(new(v2Controller), mvc.Version("2.3"))
// 	m.Handle(new(v3Controller), mvc.Version(">=3, <4"))
// 	m.Handle(new(noVersionController))
//
// See the `versioning` package's documentation for more information on
// how the version is extracted from incoming requests.
//
// Note that this Option will set the route register rule to `RouteOverlap`.
func Version(version string) OptionFunc {
	return func(c *ControllerActivator) {
		c.Router().SetRegisterRule(router.RouteOverlap) // required for this feature.

		c.Use(func(ctx context.Context) {
			if !versioning.Match(ctx, version) {
				ctx.StopExecution()
				return
			}

			ctx.Next()
		})
	}
}
