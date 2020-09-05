package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
// $ go-bindata -fs -prefix "public" ./public/...
// $ go run .

func newApp() *iris.Application {
	app := iris.New()
	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.Writef("404 not found here")
	})

	app.HandleDir("/", AssetFile())

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
	app.Listen(":8080")
}
