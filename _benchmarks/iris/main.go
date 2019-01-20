package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	app.Get("/api/values/{id}", func(ctx iris.Context) {
		ctx.WriteString("value")
	})

	app.Run(iris.Addr(":5000"))
}
