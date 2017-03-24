package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	app.Get("/servezip", func(c *iris.Context) {
		file := "./files/first.zip"
		c.SendFile(file, "c.zip")
	})

	app.Listen(":8080")
}
