package main

import "github.com/kataras/iris/v12"

// $ go get -u github.com/go-bindata/go-bindata/v3/go-bindata
// $ go-bindata -prefix "../template_blocks_0" ../template_blocks_0/views/...
// $ go run .
// # OR go-bindata -prefix "../template_blocks_0/views" ../template_blocks_0/views/... with iris.Blocks("").Binary(...)
// System files are not used, you can optionally delete the folder and run the example now.

func main() {
	app := iris.New()
	app.RegisterView(iris.Blocks("./views", ".html").Binary(Asset, AssetNames))

	app.Get("/", index)
	app.Get("/500", internalServerError)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title": "Page Title",
	}

	ctx.ViewLayout("main")
	ctx.View("index", data)
}

func internalServerError(ctx iris.Context) {
	ctx.StatusCode(iris.StatusInternalServerError)

	data := iris.Map{
		"Code":    iris.StatusInternalServerError,
		"Message": "Internal Server Error",
	}

	ctx.ViewLayout("error")
	ctx.View("500", data)
}
