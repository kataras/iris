package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// [...]

	// Good when you want to change some of the configuration's field.
	// I use that method :)
	app.Run(iris.Addr(":8080"), iris.WithoutBanner, iris.WithCharset("UTF-8"))

	// or before run:
	// app.Configure(iris.WithoutBanner, iris.WithCharset("UTF-8"))
	// app.Run(iris.Addr(":8080"))
}
