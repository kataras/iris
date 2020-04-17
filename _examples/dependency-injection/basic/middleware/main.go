package main

import (
	"errors"

	"github.com/kataras/iris/v12"
)

type (
	testInput struct {
		Email string `json:"email"`
	}

	testOutput struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
)

func handler(id int, in testInput) testOutput {
	return testOutput{
		ID:   id,
		Name: in.Email,
	}
}

var errCustom = errors.New("my_error")

func middleware(in testInput) (int, error) {
	if in.Email == "invalid" {
		// stop the execution and don't continue to "handler"
		// without firing an error.
		return iris.StatusAccepted, iris.ErrStopExecution
	} else if in.Email == "error" {
		// stop the execution and fire a custom error.
		return iris.StatusConflict, errCustom
	}

	return iris.StatusOK, nil
}

func newApp() *iris.Application {
	app := iris.New()

	// handle the route, respond with
	// a JSON and 200 status code
	// or 202 status code and empty body
	// or a 409 status code and "my_error" body.
	app.ConfigureContainer(func(api *iris.APIContainer) {
		api.Use(middleware)
		api.Post("/{id:int}", handler)
	})

	app.Configure(
		iris.WithOptimizations, /* optional */
		iris.WithoutBodyConsumptionOnUnmarshal /* required when more than one handler is consuming request payload(testInput) */)

	return app
}

func main() {
	app := newApp()
	app.Listen(":8080")
}
