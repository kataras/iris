package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/...
// # OR: go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// # to save it to your go.mod file
//
// $ go-bindata -fs -prefix "../template_amber_0/views" ../template_amber_0/views/...
// $ go run .
// # OR: go-bindata -fs -prefix "views" ./views/... if the views dir is rel to the executable.
//
// System files are not used, you can optionally delete the folder and run the example now.

func main() {
	app := iris.New()

	// Read about its markup syntax at: https://github.com/eknkc/amber
	tmpl := iris.Amber(AssetFile(), ".amber")

	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		ctx.View("index.amber", iris.Map{
			"Title": "Title of The Page",
		})
	})

	app.Listen(":8080")
}
