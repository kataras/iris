// Package main shows how to use jet templates embedded in your application with ease using the Iris built-in Jet view engine.
// This example is a customized fork of https://github.com/CloudyKit/jet/tree/master/examples/asset_packaging, so you can
// notice the differences side by side. For example, you don't have to use any external package inside your application,
// Iris manually builds the template loader for binary data when Asset and AssetNames are available via tools like the go-bindata.
package main

import (
	"os"
	"strings"

	"github.com/kataras/iris/v12"
)

// $ go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// $ go-bindata ./views/...
// $ go run .
func main() {
	app := iris.New()
	tmpl := iris.Jet("./views", ".jet").Binary(Asset, AssetNames)
	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		ctx.View("index.jet")
	})

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = ":8080"
	} else if !strings.HasPrefix(":", port) {
		port = ":" + port
	}

	app.Listen(port)
}
