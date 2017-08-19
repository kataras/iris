package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/_benchmarks/iris-mvc/controllers"
)

func main() {
	app := iris.New()
	app.Controller("/api/values/{id}", new(controllers.ValuesController))
	app.Run(iris.Addr(":5000"))
}
