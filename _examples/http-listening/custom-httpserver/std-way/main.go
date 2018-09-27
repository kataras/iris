package main

import (
	"net/http"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from the server")
	})

	app.Get("/mypath", func(ctx iris.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	})

	// call .Build before use the 'app' as a http.Handler on a custom http.Server
	app.Build()

	// create our custom server and assign the Handler/Router
	srv := &http.Server{Handler: app, Addr: ":8080"} // you have to set Handler:app and Addr, see "iris-way" which does this automatically.
	// http://localhost:8080/
	// http://localhost:8080/mypath
	println("Start a server listening on http://localhost:8080")
	srv.ListenAndServe() // same as app.Run(iris.Addr(":8080"))

	// Notes:
	// Banner is not shown at all. Same for the Interrupt Handler, even if app's configuration allows them.
	//
	// `.Run` is the only one function that cares about those three.

	// More:
	// see "multi" if you need to use more than one server at the same app.
	//
	// for a custom listener use: iris.Listener(net.Listener) or
	// iris.TLS(cert,key) or iris.AutoTLS(), see "custom-listener" example for those.
}
