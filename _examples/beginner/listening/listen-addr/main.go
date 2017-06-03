package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1>Index /</h1>")
	})

	if err := app.Run(iris.Addr(":8080")); err != nil {
		panic(err)
	}

}
