package main

import (
	"flag"

	"github.com/kataras/iris/v12"
)

var addr = flag.String("addr", ":8080", "host:port to listen on")

// $ docker-compose up
func main() {
	flag.Parse()

	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<strong>Hello World!</strong>")
	})

	app.Get("/api/values/{id:uint}", func(ctx iris.Context) {
		ctx.Writef("id: %d", ctx.Params().GetUintDefault("id", 0))
	})

	app.Run(iris.Addr(*addr))
}
