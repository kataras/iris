package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	usersRouter := app.Party("/users")
	mvc.New(usersRouter).Handle(new(myController))
	// Same as:
	// usersRouter.Get("/{p:path}", func(ctx iris.Context) {
	// 	wildcardPathParameter := ctx.Params().Get("p")
	// 	ctx.JSON(response{
	// 		Message: "The path parameter is: " + wildcardPathParameter,
	// 	})
	// })

	/*
		curl --location --request GET 'http://localhost:8080/users/path_segment_1/path_segment_2'

		Expected Output:
		{
		  "message": "The wildcard is: path_segment_1/path_segment_2"
		}
	*/
	app.Listen(":8080")
}

type myController struct{}

type response struct {
	Message string `json:"message"`
}

func (c *myController) GetByWildcard(wildcardPathParameter string) response {
	return response{
		Message: "The path parameter is: " + wildcardPathParameter,
	}
}
