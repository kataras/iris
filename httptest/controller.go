package httptest

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"

	"github.com/iris-contrib/httpexpect/v2"
)

// NewController builds a throwaway application with the given controller
// registered under prefix and returns what New returns, so a Routing value or
// any other controller can be tested on its own, without the Server type or
// the builder chain the production code goes through.
//
// The middleware is registered on the controller's own group, so it runs for
// its routes and not for the 404 and 405 chains.
//
// A failure answers what an application built with iris.NewBuilder answers,
// so a controller test asserts the body production sends:
//
//   - the path parameter type error handler and the context error handler, so
//     a bad path parameter and a failed response write render the rest/errors
//     body;
//   - rest/errors.Handler before the router, so a bare failure status
//     (ctx.StopWithStatus(403), a middleware that rejects with 401) renders
//     the rest/errors envelope instead of an empty body;
//   - rest/errors.Handler as the 404 handler, so an unknown path renders the
//     envelope too. So does a method the controller does not serve: it
//     answers 404, not 405, because the option that turns 405 on
//     (Options.Router.FireMethodNotAllowed) is off by default and
//     NewController takes no options.
//
// Nothing else the builder chain adds is here: no CORS, no access log, no
// recovery, no compression.
//
// It takes an IrisTesty rather than a *testing.T for the same reason New does,
// so ginkgo and the other testing shims keep working. It takes no
// OptionSetter: Go allows one variadic parameter and the middleware is the one
// a controller test needs; a test which needs httptest.URL or a non-default
// log level builds the application itself and calls New.
//
// Usage:
//
//	e := httptest.NewController(t, "/plans", &api.PlansController{Plans: repo})
//	e.GET("/plans/1").Expect().Status(httptest.StatusOK)
func NewController(t IrisTesty, prefix string, controller any, mw ...context.Handler) *httpexpect.Expect {
	app := iris.New()

	// Before the child Router is created: it copies the context options of
	// its parent, so a context error handler installed later would not reach a
	// prefixed group. It shares the macros with its parent, and a route takes
	// the error handler of its parameter types when it is registered, so the
	// macro handler goes before the routes too. See iris.NewBuilder for the
	// same ordering.
	app.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)
	app.EditOptions(func(opts *iris.Options) {
		opts.Context.ErrorHandler = errors.DefaultContextErrorHandler
	})

	// Before the routes, as the builder registers it: everything under it, the
	// middleware given here included, has its bare failures rendered.
	app.UseRouter(errors.Handler)

	registrations := make([]any, 0, len(mw)+1)
	for _, handler := range mw {
		registrations = append(registrations, handler)
	}
	registrations = append(registrations, controller)

	app.Router(prefix, registrations...)

	// The router's own 404. Only an empty slot is filled, exactly as the builder
	// does it when the application is built. There is no 405 to register a
	// handler for: this application runs on the default options, where a wrong
	// method is a 404.
	if app.NoRouteHandlers().Count() == 0 {
		app.HandleNoRoute(errors.Handler)
	}

	return New(t, app)
}
