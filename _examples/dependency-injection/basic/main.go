package main

import (
	"github.com/kataras/iris/v12"

	"github.com/go-playground/validator/v10"
)

type (
	testInput struct {
		Email string `json:"email" validate:"required"`
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

func configureAPI(api *iris.APIContainer) {
	/* Here is how you can inject a return value from a handler,
	   in this case the "testOutput":
	api.UseResultHandler(func(next iris.ResultHandler) iris.ResultHandler {
		return func(ctx iris.Context, v interface{}) error {
			return next(ctx, map[string]interface{}{"injected": true})
		}
	})
	*/

	api.Post("/{id:int}", handler)
}

func main() {
	app := iris.New()
	app.Validator = validator.New()
	app.Logger().SetLevel("debug")

	app.ConfigureContainer(configureAPI)
	app.Listen(":8080")
}
