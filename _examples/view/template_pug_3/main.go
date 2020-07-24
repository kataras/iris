package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// $ go-bindata ./templates/...
// $ go build
func main() {
	app := iris.New()

	tmpl := iris.Pug("./templates", ".pug").Binary(Asset, AssetNames)

	app.RegisterView(tmpl)

	app.Get("/", index)

	// http://localhost:8080
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.View("index.pug")
}
