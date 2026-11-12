package errors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// Router.UseRouter must register pre-routing middleware: it runs for every
// request under the router's path prefix, including unmatched (404) paths.
// A child router discriminates the semantics: child Use registers on matched
// routes only, while child UseRouter runs before the routing itself.
func TestRouterUseRouterRunsBeforeRouting(t *testing.T) {
	app := iris.New()

	r := errors.NewRouter(app.Router("/api"))
	r.UseRouter(func(ctx iris.Context) error {
		ctx.SetHeader("X-Pre-Routing", "yes")
		return nil
	})
	r.Get("/known", func(ctx iris.Context) error {
		ctx.WriteString("known")
		return nil
	})

	e := httptest.New(t, app)

	// Matched route: middleware runs and the chain continues on nil error.
	e.GET("/api/known").Expect().Status(httptest.StatusOK).
		Header("X-Pre-Routing").IsEqual("yes")

	// Unmatched path under the prefix: pre-routing middleware still runs.
	e.GET("/api/unknown").Expect().Status(httptest.StatusNotFound).
		Header("X-Pre-Routing").IsEqual("yes")
}
