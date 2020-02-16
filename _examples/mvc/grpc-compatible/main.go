package main

import (
	"context"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

// See https://github.com/kataras/iris/issues/1449
// for more details but in-short you can convert Iris MVC to gRPC methods by
// binding the `context.Context` from `iris.Context.Request().Context()` and gRPC input and output data.

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	// POST: http://localhost:8080/login
	// with request data: {"username": "makis"}
	// and expected output: {"message": "makis logged"}
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	mvc.New(app).
		// Request-scope binding for context.Context-type controller's method or field.
		// (or import github.com/kataras/iris/v12/hero and hero.Register(...))
		Register(func(ctx iris.Context) context.Context {
			return ctx.Request().Context()
		}).
		// Bind loginRequest.
		// Register(func(ctx iris.Context) loginRequest {
		// 	var req loginRequest
		// 	ctx.ReadJSON(&req)
		// 	return req
		// }).
		// OR
		// Bind any other structure or pointer to a structure from request's
		// XML
		// YAML
		// Query
		// Form
		// JSON (default, if not client's "Content-Type" specified otherwise)
		Register(mvc.AutoBinding).
		Handle(&myController{})

	return app
}

type myController struct{}

type loginRequest struct {
	Username string `json:"username"`
}

type loginResponse struct {
	Message string `json:"message"`
}

func (c *myController) PostLogin(ctx context.Context, input loginRequest) (loginResponse, error) {
	// [use of ctx to call a gRpc method or a database call...]
	return loginResponse{
		Message: input.Username + " logged",
	}, nil
}
