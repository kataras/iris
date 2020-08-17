package apps

import (
	"github.com/kataras/iris/v12"
)

// Switch returns a new Application
// with the sole purpose of routing the
// matched Applications through the provided "cases".
//
// The cases are filtered in order of register.
//
// Example:
//  switcher := Switch(Hosts{
//      "mydomain.com": app,
//      "test.mydomain.com": testSubdomainApp,
//      "otherdomain.com": "appName",
//  })
//  switcher.Listen(":80")
//
// Note that this is NOT a load balancer. The filters
// are executed by registration order and matched Application
// handles the request.
//
// The returned Switch Application can register routes that will run
// if no application is matched against the given filters.
// The returned Switch Application can also register custom error code handlers,
// e.g. to inject the 404 on not application found.
// It can also be wrapped with its `WrapRouter` method,
// which is really useful for logging and statistics.
func Switch(providers ...SwitchProvider) *iris.Application {
	if len(providers) == 0 {
		panic("iris: switch: empty providers")
	}

	var cases []SwitchCase
	for _, p := range providers {
		for _, c := range p.GetSwitchCases() {
			cases = append(cases, c)
		}
	}

	if len(cases) == 0 {
		panic("iris: switch: empty cases")
	}

	app := iris.New()
	// Try to build the cases apps on app.Build/Listen/Run so
	// end-developers don't worry about it.
	app.OnBuild = func() error {
		for _, c := range cases {
			if err := c.App.Build(); err != nil {
				return err
			}
		}
		return nil
	}
	// If we have a request to support
	// middlewares in that switcher app then
	// we can use app.Get("{p:path}"...) instead.
	app.UseRouter(func(ctx iris.Context) {
		for _, c := range cases {
			if c.Filter(ctx) {
				c.App.ServeHTTP(ctx.ResponseWriter().Naive(), ctx.Request())

				// if c.App.Downgraded() {
				// 	c.App.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
				// } else {
				// Note(@kataras): don't ever try something like that;
				// the context pool is the switcher's one.
				// 	ctx.SetApplication(c.App)
				// 	c.App.ServeHTTPC(ctx)
				// 	ctx.SetApplication(app)
				// }
				return
			}
		}

		// let the "switch app" handle it or fire a custom 404 error page,
		// next is the switch app's router.
		ctx.Next()
	})

	return app
}

type (
	// SwitchCase contains the filter
	// and the matched Application instance.
	SwitchCase struct {
		Filter iris.Filter
		App    *iris.Application
	}

	// A SwitchProvider should return the switch cases.
	// It's an interface instead of a direct slice because
	// we want to make available different type of structures
	// without wrapping.
	SwitchProvider interface {
		GetSwitchCases() []SwitchCase
	}

	// Join returns a new slice which joins different type of switch cases.
	Join []SwitchProvider
)

var _ SwitchProvider = SwitchCase{}

// GetSwitchCases completes the SwitchProvider, it returns itself.
func (sc SwitchCase) GetSwitchCases() []SwitchCase {
	return []SwitchCase{sc}
}

var _ SwitchProvider = Join{}

// GetSwitchCases completes the switch provider.
func (j Join) GetSwitchCases() (cases []SwitchCase) {
	for _, p := range j {
		if p == nil {
			continue
		}

		cases = append(cases, p.GetSwitchCases()...)
	}

	return
}
