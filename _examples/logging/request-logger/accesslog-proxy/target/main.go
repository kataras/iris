package main

import "github.com/kataras/iris/v12"

// The target server, can be written using any programming language and any web framework, of course.
func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Just a test route which reads some data and responds back with json.
	app.Post("/read-write", readWriteHandler)

	app.Get("/get", getHandler)

	// The target ip:port.
	app.Listen(":9090")
}

func readWriteHandler(ctx iris.Context) {
	var req interface{}
	ctx.ReadBody(&req)

	ctx.JSON(iris.Map{
		"message": "OK",
		"request": req,
	})
}

func getHandler(ctx iris.Context) {
	// ctx.CompressWriter(true)
	ctx.WriteString("Compressed data")
}
