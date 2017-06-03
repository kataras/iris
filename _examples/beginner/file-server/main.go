package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// first parameter is the request path
	// second is the operating system directory
	app.StaticWeb("/static", "./assets")

	// http://localhost:8080/static/css/main.css
	app.Run(iris.Addr(":8080"))
}
