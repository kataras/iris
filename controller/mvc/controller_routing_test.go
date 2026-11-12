package mvc_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
)

// A controller's own Routing method is the documented place to add middleware
// "for this controller". Route middleware is snapshotted when a route is
// created, so if Routing runs AFTER the generated CRUD routes are registered,
// an r.Use in there protects only the routes Routing itself adds - and the
// generated Create/Get/Update/Delete/List stay wide open, silently.

type Guarded struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GuardedFilter struct {
	Name string `json:"name"`
}

type GuardedController struct{}

func (c *GuardedController) Create(ctx iris.Context, v Guarded) (Guarded, error) {
	return v, nil
}

func (c *GuardedController) Get(ctx iris.Context, id string) (Guarded, error) {
	return Guarded{ID: id}, nil
}

func (c *GuardedController) Update(ctx iris.Context, id string, v Guarded, onlyColumns ...string) (*Guarded, bool, error) {
	return &v, true, nil
}

func (c *GuardedController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

func (c *GuardedController) List(ctx iris.Context, filter GuardedFilter, limit, offset int) ([]Guarded, int, error) {
	return nil, 0, nil
}

func (c *GuardedController) Routing(r iris.Router) {
	// The idiomatic thing a developer writes to protect "this controller".
	r.Use(func(ctx iris.Context) {
		if ctx.Header("X-Token") != "letmein" {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}
		ctx.Next()
	})

	r.Get("/greet", func(ctx iris.Context) {
		ctx.WriteString("hello")
	})
}

func TestControllerRoutingMiddlewareGuardsGeneratedRoutes(t *testing.T) {
	app := iris.New()
	app.Router("/guarded", mvc.UseController[*GuardedController]())

	e := httptest.New(t, app)

	const id = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	// The route Routing itself registered is guarded - this always worked.
	e.GET("/guarded/greet").Expect().Status(httptest.StatusUnauthorized)
	e.GET("/guarded/greet").WithHeader("X-Token", "letmein").
		Expect().Status(httptest.StatusOK).Body().IsEqual("hello")

	// The generated CRUD routes must be guarded too.
	e.GET("/guarded/" + id).Expect().Status(httptest.StatusUnauthorized)
	e.DELETE("/guarded/" + id).Expect().Status(httptest.StatusUnauthorized)
	e.POST("/guarded").WithJSON(Guarded{Name: "x"}).Expect().Status(httptest.StatusUnauthorized)
	e.POST("/guarded/list").WithJSON(GuardedFilter{}).Expect().Status(httptest.StatusUnauthorized)
	e.PUT("/guarded/" + id).WithJSON(Guarded{Name: "x"}).Expect().Status(httptest.StatusUnauthorized)

	// ...and reachable with the token.
	e.GET("/guarded/"+id).WithHeader("X-Token", "letmein").
		Expect().Status(httptest.StatusOK).JSON().Object().Value("id").IsEqual(id)
	e.DELETE("/guarded/"+id).WithHeader("X-Token", "letmein").
		Expect().Status(httptest.StatusNoContent)
}
