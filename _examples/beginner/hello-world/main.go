package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	app.Handle("GET", "/", func(ctx context.Context) {
		ctx.HTML("<b> Hello world! </b>")
	})
	app.Run(iris.Addr(":8080"), iris.WithTray)
}
