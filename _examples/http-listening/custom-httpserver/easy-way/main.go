package main

import (
	"net/http"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.Writef("Hello from the server")
	})

	app.Get("/mypath", func(ctx context.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	})

	// Any custom fields here. Handler and ErrorLog are setted to the server automatically
	srv := &http.Server{Addr: ":8080"}

	// http://localhost:8080/
	// http://localhost:8080/mypath
	app.Run(iris.Server(srv)) // same as app.Run(iris.Addr(":8080"))

	// More:
	// see "multi" if you need to use more than one server at the same app.
	//
	// for a custom listener use: iris.Listener(net.Listener) or
	// iris.TLS(cert,key) or iris.AutoTLS(), see "custom-listener" example for those.
}
