package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
//
// $ go-bindata -fs -prefix "templates" ./templates/...
// $ go run .
//
// System files are not used, you can optionally delete the folder and run the example now.
func main() {
	app := iris.New()

	tmpl := iris.Pug(AssetFile(), ".pug")

	app.RegisterView(tmpl)

	app.Get("/", index)

	// http://localhost:8080
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.View("index.pug")
}
