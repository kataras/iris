package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hello World!</h1>")
	})

	err := app.Listen(":8080", iris.WithoutServerError(iris.ErrServerClosed))
	if err != nil {
		// do something
	}
	// same as:
	// err := app.Listen(":8080")
	// import "errors"
	// if errors.Is(err, iris.ErrServerClosed) {
	//     [...]
	// }
}
