package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	app.Get("/api/values/{id}", func(ctx context.Context) {
		ctx.WriteString("value")
	})
	app.Run(iris.Addr(":5000"))
}
