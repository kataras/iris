package main

import "github.com/kataras/iris/v12"

// $ go install github.com/go-bindata/go-bindata/v3/go-bindata@latest
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
	if err := ctx.View("index.pug"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
