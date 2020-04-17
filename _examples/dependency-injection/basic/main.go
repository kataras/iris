package main

import "github.com/kataras/iris/v12"

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

func main() {
	app := iris.New()
	app.ConfigureContainer(func(api *iris.APIContainer) {
		api.Post("/{id:int}", handler)
	})
	app.Listen(":8080")
}
