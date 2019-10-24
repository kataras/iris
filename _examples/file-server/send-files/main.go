package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		file := "./files/first.zip"
		ctx.SendFile(file, "c.zip")
	})

	app.Run(iris.Addr(":8080"))
}
