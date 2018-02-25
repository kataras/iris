package main

import "github.com/kataras/iris"

// $ go get -u github.com/shuLhan/go-bindata/...
// $ go-bindata ./public/...
// $ go build
// $ ./embedded-single-page-application-with-other-routes

func newApp() *iris.Application {
	app := iris.New()
	app.OnErrorCode(404, func(ctx iris.Context) {
		ctx.Writef("404 not found here")
	})

	app.StaticEmbedded("/", "./public", Asset, AssetNames)

	// Note:
	// if you want a dynamic index page then see the file-server/embedded-single-page-application
	// which is registering a view engine based on bindata as well and a root route.

	app.Get("/ping", func(ctx iris.Context) {
		ctx.WriteString("pong")
	})
	app.Get("/.well-known", func(ctx iris.Context) {
		ctx.WriteString("well-known")
	})
	app.Get(".well-known/ready", func(ctx iris.Context) {
		ctx.WriteString("ready")
	})
	app.Get(".well-known/live", func(ctx iris.Context) {
		ctx.WriteString("live")
	})
	app.Get(".well-known/metrics", func(ctx iris.Context) {
		ctx.Writef("metrics")
	})
	return app
}

func main() {
	app := newApp()

	// http://localhost:8080/index.html
	// http://localhost:8080/app.js
	// http://localhost:8080/css/main.css
	//
	// http://localhost:8080/ping
	// http://localhost:8080/.well-known
	// http://localhost:8080/.well-known/ready
	// http://localhost:8080/.well-known/live
	// http://localhost:8080/.well-known/metrics
	//
	// Remember: we could use the root wildcard `app.Get("/{param:path}")` and serve the files manually as well.
	app.Run(iris.Addr(":8080"))
}
