package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.Writef("Hello from the SECURE server")
	})

	app.Get("/mypath", func(ctx context.Context) {
		ctx.Writef("Hello from the SECURE server on path /mypath")
	})

	// start the server (HTTPS) on port 443, this is a blocking func
	app.Run(iris.TLS("127.0.0.1:443", "mycert.cert", "mykey.key"))
}
