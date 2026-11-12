package httptest_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// plansController is a Routing value: it registers its own routes and carries
// its own dependencies, so it needs no Server type to be tested.
type plansController struct {
	plans map[string]string
}

func (c *plansController) Routing(r iris.Router) {
	r.Get("/{id}", func(ctx iris.Context) {
		plan, ok := c.plans[ctx.Params().GetString("id")]
		if !ok {
			errors.NotFoundID(ctx, "plan", ctx.Params().GetString("id"))
			return
		}

		ctx.OK(iris.Map{"name": plan}, nil)
	})
}

func TestNewController(t *testing.T) {
	c := &plansController{plans: map[string]string{"1": "starter"}}

	e := httptest.NewController(t, "/plans", c)

	e.GET("/plans/1").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("name", "starter")

	// The rest/errors handler is installed, so the error envelope renders.
	e.GET("/plans/2").Expect().Status(httptest.StatusNotFound).
		JSON().Object().HasValue("detail", "plan 2 not found")
}

func TestNewControllerWithMiddleware(t *testing.T) {
	c := &plansController{plans: map[string]string{"1": "starter"}}

	mw := func(ctx iris.Context) {
		ctx.SetHeader("X-Group", "plans")
		ctx.Next()
	}

	e := httptest.NewController(t, "/plans", c, mw)

	e.GET("/plans/1").Expect().Status(httptest.StatusOK).
		Header("X-Group").IsEqual("plans")
}

// NewController answers a failure the way an application built with
// iris.NewBuilder does: a bare failure status and an unknown path under the
// prefix both render the rest/errors envelope, so a controller test asserts
// what production sends.
func TestNewControllerEnvelope(t *testing.T) {
	routing := func(r iris.Router) {
		r.Get("/forbidden", func(ctx iris.Context) {
			ctx.StopWithStatus(iris.StatusForbidden)
		})
	}

	e := httptest.NewController(t, "/plans", routing)

	// A bare failure status from a route.
	forbidden := e.GET("/plans/forbidden").Expect().Status(httptest.StatusForbidden).
		JSON().Object()
	forbidden.Value("http_error_code").Object().
		HasValue("canonical_name", "PERMISSION_DENIED").
		HasValue("status", httptest.StatusForbidden)
	forbidden.Value("message").String().NotEmpty()

	// An unknown path under the prefix.
	notFound := e.GET("/plans/unknown/path").Expect().Status(httptest.StatusNotFound).
		JSON().Object()
	notFound.Value("http_error_code").Object().
		HasValue("canonical_name", "NOT_FOUND").
		HasValue("status", httptest.StatusNotFound)
	notFound.Value("message").String().NotEmpty()

	// A method the controller does not serve. NewController runs on the
	// default options, where 405 is off, so this is the 404 envelope as well.
	e.POST("/plans/forbidden").Expect().Status(httptest.StatusNotFound).
		JSON().Object().Value("http_error_code").Object().
		HasValue("canonical_name", "NOT_FOUND")
}
