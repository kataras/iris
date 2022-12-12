package main

import "github.com/kataras/iris/v12"

// $ go install github.com/go-bindata/go-bindata/v3/go-bindata@latest
//
// $ go-bindata -fs -prefix "../template_blocks_0/views" ../template_blocks_0/views/...
// $ go run .
//
// # OR: go-bindata -fs -prefix "views" ./views/... if the views dir is rel to the executable.
// # OR: go-bindata -fs -prefix "../template_blocks_0" ../template_blocks_0/views/...
// # with iris.Blocks(AssetFile()).RootDir("/views")
//
// System files are not used, you can optionally delete the folder and run the example now.
func main() {
	app := iris.New()
	app.RegisterView(iris.Blocks(AssetFile(), ".html"))

	app.Get("/", index)
	app.Get("/500", internalServerError)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title": "Page Title",
	}

	ctx.ViewLayout("main")
	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func internalServerError(ctx iris.Context) {
	ctx.StatusCode(iris.StatusInternalServerError)

	data := iris.Map{
		"Code":    iris.StatusInternalServerError,
		"Message": "Internal Server Error",
	}

	ctx.ViewLayout("error")
	if err := ctx.View("500", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
