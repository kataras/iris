package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1>Hello World!</h1>")
	})

	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}
