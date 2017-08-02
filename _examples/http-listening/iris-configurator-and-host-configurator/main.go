package main

import (
	"github.com/kataras/iris/_examples/http-listening/iris-configurator-and-host-configurator/counter"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	app.Configure(counter.Configurator)

	app.Run(iris.Addr(":8080"))
}
