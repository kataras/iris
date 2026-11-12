package handler_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
)

// TestOptionsBuilderChain proves that a prepared Options value built through
// the chainable NewOptions().With* methods behaves exactly like the
// package-level Option constructors it delegates to.
func TestOptionsBuilderChain(t *testing.T) {
	type Response struct {
		ID string `json:"id"`
	}

	fn := func(ctx iris.Context, id string) (Response, error) {
		return Response{ID: id}, nil
	}

	app := iris.New()

	// The classic package-level constructors.
	app.Post("/constructor/{id:string}", handler.Handler(fn,
		handler.WithStatusCode(iris.StatusCreated),
		handler.WithPathParameter[string]("id")))

	// The equivalent prepared fluent builder.
	opts := handler.NewOptions().
		WithStatusCode(iris.StatusCreated).
		WithPathParameter[string]("id")
	app.Post("/builder/{id:string}", handler.Handler(fn, handler.WithOptions(opts)))

	e := httptest.New(t, app)

	constructorBody := e.POST("/constructor/42").Expect().Status(iris.StatusCreated).Body().Raw()
	builderBody := e.POST("/builder/42").Expect().Status(iris.StatusCreated).Body().Raw()

	if constructorBody != builderBody {
		t.Fatalf("expected the builder-configured handler to respond exactly like the constructor-configured one:\nconstructor: %s\nbuilder: %s",
			constructorBody, builderBody)
	}

	e.POST("/builder/42").Expect().Status(iris.StatusCreated).
		JSON().Object().HasValue("id", "42")
}
