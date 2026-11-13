package router_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/httptest"
)

// A route that Router.AllowMethods adds fills a method the caller did not
// register on the path. It never takes the place of a route the caller
// registered, whatever the order and the register rule are.
// See https://github.com/kataras/iris/issues/1487.
func TestAllowMethodsKeepsRegisteredRoutes(t *testing.T) {
	const path = "/{id:int64}"

	getHandler := func(ctx iris.Context) { ctx.WriteString("get") }
	putHandler := func(ctx iris.Context) { ctx.WriteString("put") }

	tests := []struct {
		name     string
		register func(r iris.Router)
	}{
		{"get then put", func(r iris.Router) {
			r.Get(path, getHandler)
			r.Put(path, putHandler)
		}},
		{"put then get", func(r iris.Router) {
			r.Put(path, putHandler)
			r.Get(path, getHandler)
		}},
	}

	for _, rule := range []router.RouteRegisterRule{iris.RouteOverride, iris.RouteSkip, iris.RouteError} {
		for _, tt := range tests {
			t.Run(rule.String()+"/"+tt.name, func(t *testing.T) {
				app := iris.New(iris.Options{
					Router: iris.RouterOptions{
						FireMethodNotAllowed: true,
					},
				})
				app.Routes().SetRegisterRule(rule)

				project := app.Router("/project")
				// Both methods are allowed and the caller registers both:
				// each registration tries to fill the method of the other.
				project.AllowMethods(iris.MethodGet, iris.MethodPut)
				tt.register(project)

				e := httptest.New(t, app, httptest.LogLevel("error"))
				e.GET("/project/42").Expect().Status(httptest.StatusOK).Body().IsEqual("get")
				e.PUT("/project/42").Expect().Status(httptest.StatusOK).Body().IsEqual("put")
				e.POST("/project/42").Expect().Status(httptest.StatusMethodNotAllowed)
			})
		}
	}
}

// An allowed method that no route of the path names is still filled,
// and the routes that are registered by name keep their own handlers.
func TestAllowMethodsFillsMissingMethod(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})

	project := app.Router("/project")
	project.AllowMethods(iris.MethodGet, iris.MethodPost)
	project.Get("/{id:int64}", func(ctx iris.Context) { ctx.WriteString("get") })
	project.Put("/{id:int64}", func(ctx iris.Context) { ctx.WriteString("put") })

	e := httptest.New(t, app)
	e.GET("/project/42").Expect().Status(httptest.StatusOK).Body().IsEqual("get")
	e.PUT("/project/42").Expect().Status(httptest.StatusOK).Body().IsEqual("put")
	e.POST("/project/42").Expect().Status(httptest.StatusOK)
	e.DELETE("/project/42").Expect().Status(httptest.StatusMethodNotAllowed)
}
