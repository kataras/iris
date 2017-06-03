// for templates + go-bindata checkout the '_examples\intermediate\view\embedding-templates-into-app' folder.
package main

// First of all, execute: $ go get https://github.com/jteeuwen/go-bindata
// Secondly, execute the command: cd $GOPATH/src/github.com/kataras/iris/_examples/intermediate/serve-embedded-files && go-bindata ./assets/...

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<b> Hi from index</b>")
	})

	// executing this go-bindata command creates a source file named 'bindata.go' which
	// gives you the Asset and AssetNames funcs which we will pass into .StaticAssets
	// for more viist: https://github.com/jteeuwen/go-bindata
	// Iris gives you a way to integrade these functions to your web app

	// For the reason that you may use go-bindata to embed more than your assets,
	// you should pass the 'virtual directory path', for example here is the : "./assets"
	// and the request path, which these files will be served to,
	// you can set as "/assets" or "/static" which resulting on http://localhost:8080/static/*anyfile.*extension
	app.StaticEmbedded("/static", "./assets", Asset, AssetNames)

	// that's all
	// this will serve the ./assets (embedded) files to the /static request path for example the favicon.ico will be served as :
	// http://localhost:8080/static/favicon.ico
	// Methods: GET and HEAD

	app.Run(iris.Addr(":8080"))
}

// Navigate to:
// http://localhost:8080/static/favicon.ico
// http://localhost:8080/static/js/jquery-2.1.1.js
// http://localhost:8080/static/css/bootstrap.min.css

// Now, these files are are stored into inside your executable program, no need to keep it in the same location with your assets folder.
