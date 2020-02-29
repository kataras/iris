package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/hero"
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

func dependency(ctx iris.Context) (in testInput, err error) {
	err = ctx.ReadJSON(&in)
	return
}

func main() {
	app := iris.New()

	c := hero.New()
	c.Register(dependency)
	app.Post("/{id:int}", c.Handler(handler))
	app.Listen(":5000", iris.WithOptimizations)
}
