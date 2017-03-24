package main

import (
	"net"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from the server")
	})

	app.Get("/mypath", func(ctx *iris.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	})

	// create our custom listener
	ln, err := net.Listen("tcp4", ":8080")
	if err != nil {
		panic(err)
	}

	// use of the custom listener
	app.Serve(ln)
}
