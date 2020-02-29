package main

import (
	"context"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

// See https://github.com/kataras/iris/issues/1449
// Iris automatically binds the standard "context" context.Context to `iris.Context.Request().Context()`
// and any other structure that is not mapping to a registered dependency
// as a payload depends on the request, e.g XML, YAML, Query, Form, JSON.
//
// Useful to use gRPC services as Iris controllers fast and without wrappers.

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

	mvc.New(app).Handle(&myController{})

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
